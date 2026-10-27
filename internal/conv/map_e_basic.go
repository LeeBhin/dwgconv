package conv

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

// Group E: DXF mappings of the non-graphic objects.

func init() {
	register("DICTIONARY", 0x2A, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Dictionary) {
		eDictionary(m, x, b)
	})
	register("ACDBDICTIONARYWDFLT", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DictionaryWithDefault) {
		eDictionary(m, x, &b.Dictionary)
		m.Subclass("AcDbDictionaryWithDefault")
		x.HandleRef(m, 340, &b.Default, 5)
	})
	register("XRECORD", 0x4F, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.XRecord) {
		m.Subclass("AcDbXrecord")
		m.Int16(280, &b.Cloning)
		if !m.Reading() {
			for _, g := range eXRecordGroups(b.Data, x.d.Version.R2007, x.d.Text) {
				m.Writer().Group(g)
			}
			return
		}
		rest := m.Rest()
		n := 0
		for n < len(rest) && rest[n].Code != 1001 {
			n++
		}
		b.Data = eXRecordData(rest[:n], x.d.Version.R2007, x.d.Text)
		m.Seek(m.Pos() + n)
	})
	register("DICTIONARYVAR", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DictionaryVar) {
		m.Subclass("DictionaryVariables")
		m.Byte(280, &b.Schema)
		v := b.Value
		if !m.Reading() {
			v = eDictVarValue(x, o, v)
		}
		m.Str(1, &v)
		b.Value = v
	})
	register("IDBUFFER", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.IDBuffer) {
		m.Subclass("AcDbIdBuffer")
		eRefList(m, x, 330, &b.ObjIDs, 4)
	})
	register("SORTENTSTABLE", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.SortEntsTable) {
		m.Subclass("AcDbSortentsTable")
		x.HandleRef(m, 330, &b.BlockOwner, 4)
		if !m.Reading() {
			// ODA writes the entries ordered by entity handle, one per entity (first occurrence wins).
			idx := make([]int, 0, len(b.Entities))
			seen := map[uint64]bool{}
			for i := range b.Entities {
				if !seen[b.Entities[i].H] {
					seen[b.Entities[i].H] = true
					idx = append(idx, i)
				}
			}
			sort.SliceStable(idx, func(i, j int) bool { return b.Entities[idx[i]].H < b.Entities[idx[j]].H })
			for _, i := range idx {
				x.HandleRef(m, 331, &b.Entities[i], 4)
				x.HandleRef(m, 5, &b.SortHandles[i], 0)
			}
			return
		}
		for m.Has(331) {
			var e, s dwg.Ref
			x.HandleRef(m, 331, &e, 4)
			x.HandleRef(m, 5, &s, 0)
			b.Entities = append(b.Entities, e)
			b.SortHandles = append(b.SortHandles, s)
		}
	})
	register("SCALE", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Scale) {
		m.Subclass("AcDbScale")
		m.Int16(70, &b.Unknown)
		m.Str(300, &b.Name)
		m.Float(140, &b.PaperUnits)
		m.Float(141, &b.DrawingUnits)
		m.Bool(290, &b.HasUnitScale)
	})
	register("LAYER_INDEX", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.LayerIndex) {
		m.Subclass("AcDbIndex")
		eJulian(m, 40, &b.Timestamp1, &b.Timestamp2)
		m.Subclass("AcDbLayerIndex")
		if !m.Reading() {
			for i := range b.IndexNames {
				m.Str(8, &b.IndexNames[i])
				x.HandleRef(m, 360, &b.Entries[i], 5)
				m.Int32(90, &b.IndexLongs[i])
			}
			return
		}
		for m.Has(8) {
			var s string
			var r dwg.Ref
			var n int32
			m.Str(8, &s)
			x.HandleRef(m, 360, &r, 5)
			m.Int32(90, &n)
			b.IndexNames = append(b.IndexNames, s)
			b.Entries = append(b.Entries, r)
			b.IndexLongs = append(b.IndexLongs, n)
		}
	})
	register("ACDBPLACEHOLDER", 0x50, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Placeholder) {})
	register("IMAGEDEF", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.ImageDef) {
		m.Subclass("AcDbRasterImageDef")
		m.Int32(90, &b.ClassVersion)
		m.Str(1, &b.FilePath)
		m.Point2(10, &b.ImageSize)
		m.Point2(11, &b.PixelSize)
		m.Bool(280, &b.IsLoaded)
		m.Byte(281, &b.ResUnits)
	})
	register("FIELDLIST", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.FieldList) {
		m.Subclass("AcDbIdSet")
		n := m.Count(90, len(b.Fields))
		if m.Reading() {
			b.Fields = make([]dwg.Ref, n)
		}
		for i := range b.Fields {
			x.HandleRef(m, 330, &b.Fields[i], 4)
		}
		m.Subclass("AcDbFieldList")
	})
	register("MLINESTYLE", 0x49, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.MLineStyle) {
		m.Subclass("AcDbMlineStyle")
		m.Str(2, &b.Name)
		f := eMLineFlagsToDXF(b.Flags)
		m.Int16(70, &f)
		m.Str(3, &b.Desc)
		eCMC(m, 62, &b.FillColor)
		m.Angle(51, &b.StartAngle)
		m.Angle(52, &b.EndAngle)
		n := m.Count(71, len(b.Lines))
		if m.Reading() {
			b.Flags = eMLineFlagsFromDXF(f)
			b.Lines = make([]dwg.MLineStyleLine, n)
		}
		for i := range b.Lines {
			l := &b.Lines[i]
			m.Float(49, &l.Offset)
			eCMC(m, 62, &l.Color)
			eMLineLtype(m, x, l)
		}
	})
}

// eDictionary binds the AcDbDictionary part (§20.4.44): 280 hard owner (omitted when 0), 281 cloning,
// then 3/350 pairs (360 when the dictionary hard-owns its entries).
func eDictionary(m *dxf.Mapper, x *ctx, b *dwg.Dictionary) {
	m.Subclass("AcDbDictionary")
	ho := int16(b.HardOwner)
	m.Int16Def(280, &ho, 0)
	m.Int16(281, &b.Cloning)
	b.HardOwner = byte(ho)
	code := 350
	if b.HardOwner != 0 {
		code = 360
	}
	if !m.Reading() {
		for _, e := range eDictEntries(x, b) {
			m.Str(3, &e.name)
			x.HandleRef(m, code, &e.item, 2)
		}
		return
	}
	for m.Has(3) {
		var s string
		var r dwg.Ref
		m.Str(3, &s)
		if m.Peek(360) {
			x.HandleRef(m, 360, &r, 2)
		} else {
			x.HandleRef(m, 350, &r, 2)
		}
		b.Names = append(b.Names, s)
		b.Items = append(b.Items, r)
	}
}

// eStyleVars maps the style-name DICTIONARYVARs to the named-object dictionary holding the styles.
// ODA writes their value as the style's stored name (e.g. STANDARD -> Standard).
var eStyleVars = map[string]string{
	"CMLEADERSTYLE":     "ACAD_MLEADERSTYLE",
	"CTABLESTYLE":       "ACAD_TABLESTYLE",
	"CVIEWDETAILSTYLE":  "ACAD_DETAILVIEWSTYLE",
	"CVIEWSECTIONSTYLE": "ACAD_SECTIONVIEWSTYLE",
}

func eDictVarValue(x *ctx, o *dwg.Object, v string) string {
	owner := x.d.ByHandle[o.Owner.H]
	if owner == nil || eDict(owner) == nil {
		return v
	}
	var key string
	od := eDict(owner)
	for i, it := range od.Items {
		if it.H == o.Handle {
			key = od.Names[i]
		}
	}
	coll, ok := eStyleVars[key]
	if !ok {
		return v
	}
	for _, r := range x.d.Objects {
		rd := eDict(r)
		if rd == nil || r.Owner.H != 0 {
			continue
		}
		for i, n := range rd.Names {
			if n != coll {
				continue
			}
			if sd := x.d.ByHandle[rd.Items[i].H]; sd != nil && eDict(sd) != nil {
				for _, sn := range eDict(sd).Names {
					if strings.EqualFold(sn, v) {
						return sn
					}
				}
			}
		}
	}
	return v
}

type eEntry struct {
	name string
	item dwg.Ref
}

// eDictEntries lists the entries as ODA writes them: entries whose target is not written are dropped,
// anonymous entries (*A...) are renumbered *A<n> by item position, and entries are sorted by name
// (case-insensitively, lower-case order).
func eDictEntries(x *ctx, b *dwg.Dictionary) []eEntry {
	var out []eEntry
	for i := range b.Names {
		if eODAOmits(x, b.Items[i].H) {
			continue
		}
		name := b.Names[i]
		if strings.HasPrefix(name, "*A") {
			name = "*A" + strconv.Itoa(i+1)
		}
		out = append(out, eEntry{name, b.Items[i]})
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].name) < strings.ToLower(out[j].name) })
	return out
}

// eRefList binds a run of handle groups with the same code.
func eRefList(m *dxf.Mapper, x *ctx, code int, rs *[]dwg.Ref, refCode byte) {
	if !m.Reading() {
		for i := range *rs {
			x.HandleRef(m, code, &(*rs)[i], refCode)
		}
		return
	}
	for m.Has(code) {
		var r dwg.Ref
		x.HandleRef(m, code, &r, refCode)
		*rs = append(*rs, r)
	}
}

// eJulian binds a DWG (day, milliseconds) timestamp pair as a DXF Julian date.
func eJulian(m *dxf.Mapper, code int, day, ms *int32) {
	v := float64(*day) + float64(*ms)/86400000
	m.Float(code, &v)
	if m.Reading() {
		d := math.Floor(v)
		*day = int32(d)
		*ms = int32(math.Round((v - d) * 86400000))
	}
}

// eCMC binds a CMC color (§2.11) as DXF 62 (+420 for true color). The color method lives in the RGB high byte:
// 0xC0 ByLayer, 0xC1 ByBlock, 0xC2 true color, 0xC3 ACI, 0xC8 none (DXF 257).
func eCMC(m *dxf.Mapper, code int, c *dwg.CmColor) {
	if !m.Reading() {
		var aci int16
		method := uint32(c.RGB) >> 24
		switch method {
		case 0xC0:
			aci = 256
		case 0xC1:
			aci = 0
		case 0xC2:
			aci = int16(c.Index)
		case 0xC8: // none
			aci = 257
		default:
			aci = int16(uint32(c.RGB) & 0xFF)
		}
		m.Int16(code, &aci)
		if method == 0xC2 {
			rgb := c.RGB & 0xFFFFFF
			m.Int32(420, &rgb)
		}
		return
	}
	var aci int16
	m.Int16(code, &aci)
	c.Index = 0
	switch {
	case m.Peek(420):
		var rgb int32
		m.Int32(420, &rgb)
		c.RGB = eRGB(0xC2, uint32(rgb)&0xFFFFFF)
		c.Index = aci
	case aci == 256:
		c.RGB = eRGB(0xC0, 0)
	case aci == 257:
		c.RGB = eRGB(0xC8, 0)
	case aci == 0:
		c.RGB = eRGB(0xC1, 0)
	default:
		c.RGB = eRGB(0xC3, uint32(aci)&0xFF)
	}
}

func eRGB(method, v uint32) int32 { return int32(method<<24 | v) }

// MLINESTYLE flag bits: DWG bit -> DXF bit (§20.4.73).
var eMLineFlagMap = [][2]int16{{1, 2}, {2, 1}, {16, 16}, {32, 64}, {64, 32}, {256, 256}, {512, 1024}, {1024, 512}}

func eMLineFlagsToDXF(f int16) int16 {
	var out int16
	for _, p := range eMLineFlagMap {
		if f&p[0] != 0 {
			out |= p[1]
		}
	}
	return out
}

func eMLineFlagsFromDXF(f int16) int16 {
	var out int16
	for _, p := range eMLineFlagMap {
		if f&p[1] != 0 {
			out |= p[0]
		}
	}
	return out
}

// eMLineLtype binds group 6 of an MLINESTYLE element: a handle in R2018+, a linetype index before.
func eMLineLtype(m *dxf.Mapper, x *ctx, l *dwg.MLineStyleLine) {
	if !m.Reading() {
		var s string
		if x.d.Version.R2018 {
			s = x.names[l.Ltype.H]
		} else {
			s = eLtypeIndexName(x, l.LtIndex)
		}
		if u := strings.ToUpper(s); u == "BYLAYER" || u == "BYBLOCK" {
			s = u
		}
		m.Str(6, &s)
		return
	}
	var s string
	m.Str(6, &s)
	if x.d.Version.R2018 {
		l.Ltype = dwg.Ref{Code: 5, H: x.byName["LTYPE"][strings.ToUpper(s)]}
	} else {
		l.LtIndex = eLtypeNameIndex(x, s)
	}
}

// Linetype index (pre-R2018 MLINESTYLE): 0x7FFF ByLayer, 0x7FFE ByBlock, otherwise the position in the LTYPE control.
func eLtypeIndexName(x *ctx, i int16) string {
	switch i {
	case 0x7FFF:
		return "BYLAYER"
	case 0x7FFE:
		return "BYBLOCK"
	}
	if c := eLtypeControl(x); c != nil && int(i) >= 0 && int(i) < len(c.Entries) {
		return x.names[c.Entries[i].H]
	}
	return ""
}

func eLtypeNameIndex(x *ctx, s string) int16 {
	switch strings.ToUpper(s) {
	case "BYLAYER", "":
		return 0x7FFF
	case "BYBLOCK":
		return 0x7FFE
	}
	if c := eLtypeControl(x); c != nil {
		h := x.byName["LTYPE"][strings.ToUpper(s)]
		for i, e := range c.Entries {
			if e.H == h {
				return int16(i)
			}
		}
	}
	return 0x7FFF
}

func eLtypeControl(x *ctx) *dwg.TableControl {
	for _, o := range x.d.Objects {
		if o.Type == 0x38 {
			if c, ok := o.Body.(*dwg.TableControl); ok {
				return c
			}
		}
	}
	return nil
}
