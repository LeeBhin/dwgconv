package conv

import (
	"fmt"
	"reflect"
	"strings"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// tableOf maps DWG table record type names to the DXF table name.
var tableOf = map[string]string{
	"LAYER": "LAYER", "LTYPE": "LTYPE", "STYLE": "STYLE", "DIMSTYLE": "DIMSTYLE", "APPID": "APPID",
	"VIEW": "VIEW", "UCS": "UCS", "VPORT": "VPORT", "BLOCK_HEADER": "BLOCK_RECORD",
	"MLINESTYLE": "MLINESTYLE",
}

// ctx holds the lookups shared by export and import.
type ctx struct {
	d      *dwg.Document
	names  map[uint64]string            // table record handle -> name
	byName map[string]map[string]uint64 // DXF table -> upper(name) -> handle
	ms, ps uint64                       // *Model_Space / *Paper_Space block records
	hdr    *dwg.Header
	// measurement is $MEASUREMENT (stored in AcDb:Template, not the header).
	measurement int16
	// blockOwner maps entities (and BLOCK/ENDBLK) to the block record listing them.
	blockOwner map[uint64]uint64
	// auxVersion/auxMaint are the drawing format of the source (AcDb:AuxHeader), written in proxies.
	auxVersion, auxMaint int
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
	if d.File != nil {
		if h, err := dwg.DecodeFileHeader(d.File); err == nil {
			x.hdr = h
			if r := h.Vars.BlockRecordModelSpace.H; r != 0 {
				x.ms = r
			}
			if r := h.Vars.BlockRecordPaperSpace.H; r != 0 {
				x.ps = r
			}
		}
	}
	x.numberBlockNames()
	if x.ps != 0 {
		x.byName["BLOCK_RECORD"]["*PAPER_SPACE"] = x.ps // several records share the name; the active one wins
	}
	x.blockOwner = map[uint64]uint64{}
	for _, o := range d.Objects {
		if bh, ok := o.Body.(*dwg.BlockHeader); ok {
			x.blockOwner[bh.Block.H] = o.Handle
			x.blockOwner[bh.EndBlk.H] = o.Handle
			for _, e := range bh.Entities {
				x.blockOwner[e.H] = o.Handle
			}
		}
	}
	// Proxy data format: the file's version code, and its maintenance byte (0x0B) or, when that is
	// 0 (R2018 files), the AuxHeader maintenance version — as ODA writes groups 71/97.
	x.auxVersion, x.auxMaint = 33, 0
	if d.File != nil {
		x.auxVersion = dwg.VersionCode(d.File.Version)
		x.auxMaint = int(d.File.Prefix[0x0B])
		if s := d.File.Section("AcDb:AuxHeader"); s != nil && x.auxMaint == 0 {
			if a, err := dwg.DecodeAuxHeader(s.Data, d.Version); err == nil {
				x.auxMaint = int(a.Maint)
			}
		}
	}
	return x
}

func (x *ctx) addName(table, name string, h uint64) {
	x.names[h] = name
	if x.byName[table] == nil {
		x.byName[table] = map[string]uint64{}
	}
	x.byName[table][nameKey(name)] = h
}

// nameKey is the lookup key of a table record name: case-insensitive, \U+XXXX escapes resolved.
func nameKey(name string) string { return strings.ToUpper(dxf.UnescapeU(name)) }

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
	if h, ok := x.byName[table][nameKey(s)]; ok {
		*r = dwg.Ref{Code: refCode, H: h}
	}
}

// NameRefDef is NameRef where the record named def is implied when the group is absent (e.g. STANDARD).
func (x *ctx) NameRefDef(m *dxf.Mapper, code int, r *dwg.Ref, table string, refCode byte, def string) {
	dh := x.byName[table][nameKey(def)]
	if !m.Reading() {
		if r.H == dh {
			return
		}
		x.NameRef(m, code, r, table, refCode)
		return
	}
	*r = dwg.Ref{Code: refCode, H: dh}
	x.NameRef(m, code, r, table, refCode)
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

// numberBlockNames gives anonymous blocks and extra paper spaces the numbered names DXF uses:
// DWG stores "*D", "*X", "*U"... and repeated "*Paper_Space"; the DXF name appends the position in
// the block table minus 2 for anonymous blocks, and 0, 1, ... for the extra paper spaces (as ODA).
func (x *ctx) numberBlockNames() {
	var ctl *dwg.Object
	for _, o := range x.d.Objects {
		if o.Type == 0x30 {
			ctl = o
			break
		}
	}
	if ctl == nil {
		return
	}
	paper := 0
	for i, r := range controlEntries(ctl) {
		name, ok := x.names[r.H]
		if !ok {
			continue
		}
		up := strings.ToUpper(name)
		switch {
		case up == "*PAPER_SPACE":
			if paper > 0 {
				x.addName("BLOCK_RECORD", fmt.Sprintf("%s%d", name, paper-1), r.H)
			}
			paper++
		case isAnonymousBase(name):
			x.addName("BLOCK_RECORD", fmt.Sprintf("%s%d", name, i-2), r.H)
		}
	}
}

// isAnonymousBase reports names like "*D", "*X", "*U": '*' followed only by letters, other than the spaces.
func isAnonymousBase(name string) bool {
	if len(name) < 2 || name[0] != '*' {
		return false
	}
	for _, c := range name[1:] {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z') {
			return false
		}
	}
	up := strings.ToUpper(name)
	return up != "*MODEL_SPACE" && up != "*PAPER_SPACE"
}

// dwgBlockName is the name a block record is stored under in DWG: numbered anonymous and paper
// space names lose their number ("*D12" -> "*D", "*Paper_Space0" -> "*Paper_Space").
func dwgBlockName(name string) string {
	if len(name) < 2 || name[0] != '*' {
		return name
	}
	end := len(name)
	for end > 1 && name[end-1] >= '0' && name[end-1] <= '9' {
		end--
	}
	if end == len(name) {
		return name
	}
	base := name[:end]
	if isAnonymousBase(base) || strings.EqualFold(base, "*Paper_Space") {
		return base
	}
	return name
}
