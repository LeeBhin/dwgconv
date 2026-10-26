package conv

import (
	"reflect"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// exportRecord writes one entity or object record (0, 5, common groups, fields, xdata).
func (x *ctx) exportRecord(w *dxf.Writer, o *dwg.Object) bool {
	mp := mappingFor(o)
	if mp == nil {
		return false
	}
	name := mp.dxfName
	if mp.name != nil {
		name = mp.name(o)
	}
	w.Str(0, name)
	m := dxf.NewWriteMapper(w)
	if mp.preHandle != nil {
		mp.preHandle(m, x, o)
	}
	w.Handle(mp.handleCode, o.Handle)
	if o.IsEntity {
		x.entityCommon(m, o)
	} else {
		owner := o.Owner.H
		x.ownership(m, o, &owner)
	}
	mp.fields(m, x, o, o.Body)
	x.xdata(m, o)
	return true
}

// importRecord builds an object from a DXF record using the registered mapping for its type.
func (x *ctx) importRecord(r *dxf.Record, entity bool) *dwg.Object {
	mps := byDXFName[r.Type]
	if len(mps) == 0 {
		return nil
	}
	mp := mps[0]
	for _, c := range mps { // prefer the fixed-type mapping when one exists
		if c.dwgType != 0 && c.match == nil {
			mp = c
			break
		}
	}
	for _, c := range mps {
		if c.match != nil && c.match(r) {
			mp = c
			break
		}
	}
	if mp.match != nil && !mp.match(r) {
		return nil // e.g. a proxy whose data is in another DWG version: cannot be restored
	}
	o := &dwg.Object{Handle: r.Handle(mp.handleCode), IsEntity: entity, Body: mp.newBody(), Type: mp.dwgType}
	m := dxf.NewReadMapper(r)
	if mp.preHandle != nil {
		mp.preHandle(m, x, o)
	}
	var h uint64
	m.Handle(mp.handleCode, &h)
	if entity {
		o.Ent = &dwg.EntityCommon{}
		x.entityCommon(m, o)
	} else {
		owner := uint64(0)
		x.ownership(m, o, &owner)
		o.Owner = dwg.Ref{Code: 4, H: owner}
	}
	mp.fields(m, x, o, o.Body)
	x.xdata(m, o)
	// table records keep a null external-reference block handle as a hard pointer (code 5)
	if f := reflect.ValueOf(o.Body).Elem().FieldByName("XrefBlock"); f.IsValid() {
		if r := f.Interface().(dwg.Ref); r.H == 0 {
			f.Set(reflect.ValueOf(dwg.Ref{Code: 5}))
		}
	}
	// table records written by AutoCAD carry the "referenced" bit (64); DXF does not, so set it
	if f := reflect.ValueOf(o.Body).Elem().FieldByName("Flag64"); f.IsValid() && f.Kind() == reflect.Bool {
		f.SetBool(true)
	}
	fixNullRefs(reflect.ValueOf(o.Body))
	if _, ok := o.Body.(*dwg.BlockHeader); ok {
		if n, ok := entryName(o); ok {
			setNameField(o, dwgBlockName(n)) // "*D12" is stored as "*D" in DWG
		}
	}
	return o
}

// fixNullRefs gives null handle references a pointer code: AutoCAD never writes code 0 for a
// handle field, and readers reject it ("object improperly read").
func fixNullRefs(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			fixNullRefs(v.Elem())
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(dwg.Ref{}) {
			if r := v.Interface().(dwg.Ref); r.H == 0 && r.Code == 0 && v.CanSet() {
				v.Set(reflect.ValueOf(dwg.Ref{Code: 5}))
			}
			return
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() || v.Type().Field(i).Anonymous {
				fixNullRefs(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			fixNullRefs(v.Index(i))
		}
	}
}
