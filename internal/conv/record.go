package conv

import (
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
	w.Handle(5, o.Handle)
	m := dxf.NewWriteMapper(w)
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
	o := &dwg.Object{Handle: r.Handle(5), IsEntity: entity, Body: mp.newBody(), Type: mp.dwgType}
	m := dxf.NewReadMapper(r)
	var h uint64
	m.Handle(5, &h)
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
	return o
}
