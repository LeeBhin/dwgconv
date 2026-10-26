package conv

import (
	"reflect"
	"strings"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// tableOf maps DWG table record type names to the DXF table name.
var tableOf = map[string]string{
	"LAYER": "LAYER", "LTYPE": "LTYPE", "STYLE": "STYLE", "DIMSTYLE": "DIMSTYLE", "APPID": "APPID",
	"VIEW": "VIEW", "UCS": "UCS", "VPORT": "VPORT", "BLOCK_HEADER": "BLOCK_RECORD",
}

// ctx holds the lookups shared by export and import.
type ctx struct {
	d      *dwg.Document
	names  map[uint64]string            // table record handle -> name
	byName map[string]map[string]uint64 // DXF table -> upper(name) -> handle
	ms, ps uint64                       // *Model_Space / *Paper_Space block records
}

func newCtx(d *dwg.Document) *ctx {
	x := &ctx{d: d, names: map[uint64]string{}, byName: map[string]map[string]uint64{}}
	for _, o := range d.Objects {
		t, ok := tableOf[o.TypeName(d.Classes.Classes)]
		if !ok {
			continue
		}
		if n, ok := entryName(o); ok {
			x.addName(t, n, o.Handle)
		}
	}
	x.ms = x.byName["BLOCK_RECORD"]["*MODEL_SPACE"]
	x.ps = x.byName["BLOCK_RECORD"]["*PAPER_SPACE"]
	return x
}

func (x *ctx) addName(table, name string, h uint64) {
	x.names[h] = name
	if x.byName[table] == nil {
		x.byName[table] = map[string]uint64{}
	}
	x.byName[table][strings.ToUpper(name)] = h
}

// entryName reads the exported Name field of a table record body.
func entryName(o *dwg.Object) (string, bool) {
	if o.Body == nil {
		return "", false
	}
	v := reflect.ValueOf(o.Body).Elem()
	f := v.FieldByName("Name")
	if !f.IsValid() || f.Kind() != reflect.String {
		return "", false
	}
	return f.String(), true
}

// NameRef binds a table record reference written in DXF as its name (e.g. 7 = text style).
func (x *ctx) NameRef(m *dxf.Mapper, code int, r *dwg.Ref, table string, refCode byte) {
	if !m.Reading() {
		if r.H == 0 {
			return
		}
		s := x.names[r.H]
		m.Str(code, &s)
		return
	}
	var s string
	m.Str(code, &s)
	if s == "" {
		return
	}
	if h, ok := x.byName[table][strings.ToUpper(s)]; ok {
		*r = dwg.Ref{Code: refCode, H: h}
	}
}

// HandleRef binds a handle reference written as hex (330/340/350/360...).
func (x *ctx) HandleRef(m *dxf.Mapper, code int, r *dwg.Ref, refCode byte) {
	h := r.H
	m.Handle(code, &h)
	if m.Reading() && h != 0 {
		*r = dwg.Ref{Code: refCode, H: h}
	}
}

// HandleRefOpt is HandleRef that skips null handles on write.
func (x *ctx) HandleRefOpt(m *dxf.Mapper, code int, r *dwg.Ref, refCode byte) {
	if !m.Reading() && r.H == 0 {
		return
	}
	x.HandleRef(m, code, r, refCode)
}
