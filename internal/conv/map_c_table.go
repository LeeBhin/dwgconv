package conv

import (
	"strings"

	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

// ACAD_TABLE. DXF carries the pre-R2010 structures for every version: the block reference part,
// the grid, the cells with their R2007+ value blocks (301 CELL_VALUE ... 304 ACVALUE_END) and the
// per-cell overrides. An R2010+ DWG holds the table content instead, so the classic fields are
// derived from the content before writing and the content is built from them after reading
// (dwg.Table.ClassicFromContent / ContentFromClassic). Cell groups follow ODA's DXF of the 2007
// and 2010 samples; the table-level override values (93-96 flags only) are still unmapped.

// classic cell override bits and groups per edge (top, right, bottom, left), as in the DWG codec
var (
	cEdgeColorBit = [4]int32{0x40, 0x80, 0x100, 0x200}
	cEdgeLWBit    = [4]int32{0x400, 0x800, 0x1000, 0x2000}
	cEdgeVisBit   = [4]int32{0x4000, 0x8000, 0x10000, 0x20000}
	cEdgeColor    = [4]int{69, 65, 66, 68}
	cEdgeLW       = [4]int{279, 275, 276, 278}
	cEdgeVis      = [4]int{289, 285, 286, 288}
)

func init() {
	register("ACAD_TABLE", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Table) {
		if !m.Reading() && x.d.Version.R2010 {
			classic := *b // the object keeps its content-only form
			classic.ClassicFromContent()
			b = &classic
		}
		m.Subclass("AcDbBlockReference")
		x.NameRef(m, 2, &b.BlockHeader, "BLOCK_RECORD", 5)
		m.Point3(10, &b.InsPt)
		m.FloatDef(41, &b.Scale[0], 1)
		m.FloatDef(42, &b.Scale[1], 1)
		m.FloatDef(43, &b.Scale[2], 1)
		m.AngleDef(50, &b.Rotation, 0)
		m.Extrusion(&b.Extrusion)
		m.Subclass("AcDbTable")
		ver := int16(0)
		m.Int16Def(280, &ver, 0)
		x.HandleRef(m, 342, &b.TableStyle, 5)
		x.HandleRef(m, 343, &b.BlockHeader, 5)
		m.Point3(11, &b.HorDir)
		m.Int16(90, &b.Flags)
		nr := m.Count(91, len(b.RowHeights))
		nc := m.Count(92, len(b.ColWidths))
		// the table-level flags precede the grid; bounded so a missing one does not take a cell's
		end := cGroupIndex(m, 141, 142, 171)
		cInt32Before(m, 93, &b.TableOverride, end)
		cInt32Before(m, 94, &b.BorderColorFlags, end)
		cInt32Before(m, 95, &b.BorderLWFlags, end)
		cInt32Before(m, 96, &b.BorderVisFlags, end)
		if m.Reading() {
			b.RowHeights = make([]float64, nr)
			b.ColWidths = make([]float64, nc)
			b.Cells = make([]dwg.TableCell, nr*nc)
			b.HasTableOverrides = b.TableOverride != 0
			b.HasBorderColor = b.BorderColorFlags != 0
			b.HasBorderLW = b.BorderLWFlags != 0
			b.HasBorderVis = b.BorderVisFlags != 0
		}
		for i := range b.RowHeights {
			m.Float(141, &b.RowHeights[i])
		}
		for i := range b.ColWidths {
			m.Float(142, &b.ColWidths[i])
		}
		values := !m.Reading() || m.Has(301) // R2007+ DXF carries a value block per cell
		for i := range b.Cells {
			t := &b.Cells[i]
			m.Int16(171, &t.Type)
			cellEnd := cGroupIndex(m, 171)
			m.Byte(172, &t.EdgeFlags)
			m.Bool(173, &t.Merged)
			m.Bool(174, &t.Autofit)
			m.Int32(175, &t.MergedWidth)
			m.Int32(176, &t.MergedHeight)
			m.Int32(91, &t.FlagOverride)
			m.Byte(178, &t.VirtualEdge)
			m.Angle(145, &t.Rotation)
			if m.Reading() {
				t.HasOverride = t.FlagOverride != 0
			}
			switch t.Type {
			case 1:
				cHandleBefore(m, x, 344, &t.Value, 5, cellEnd)
				if !values {
					m.Str(1, &t.Text)
				}
			case 2:
				cHandleBefore(m, x, 340, &t.Value, 5, cellEnd)
				m.Float(144, &t.BlockScale)
				n := m.Count(179, len(t.Attrs))
				if m.Reading() {
					t.Attrs = make([]dwg.TableAttr, n)
					t.HasAttrs = n > 0
				}
				for k := range t.Attrs {
					x.HandleRef(m, 331, &t.Attrs[k].AttDef, 4)
					m.Str(300, &t.Attrs[k].Text)
				}
			}
			cCellOverrides(m, x, t)
			if values {
				cCellValue(m, t)
			}
			if m.Reading() {
				if t.CellValue.DataType == 4 && t.Text == "" {
					t.Text = t.CellValue.Str
				}
				switch {
				case !x.d.Version.R2007: // the value is not stored before R2007
					t.CellValue = dwg.TableValue{}
				case t.CellValue.Flags == 0 && t.CellValue.DataType == 0:
					t.CellValue = cTextValue(t.Text)
				}
			}
		}
		if m.Reading() && x.d.Version.R2010 {
			b.ContentFromClassic()
		}
	})
}

// cGroupIndex returns the index of the first group with one of the codes from the cursor on, or
// of the next subclass marker, or the record end (reading only).
func cGroupIndex(m *dxf.Mapper, codes ...int) int {
	if !m.Reading() {
		return 0
	}
	gs := m.Record().Groups
	for i := m.Pos(); i < len(gs); i++ {
		if gs[i].Code == 100 {
			return i
		}
		for _, c := range codes {
			if gs[i].Code == c {
				return i
			}
		}
	}
	return len(gs)
}

// cTakeBefore takes the next group with code before index end (reading only).
func cTakeBefore(m *dxf.Mapper, code, end int) (dxf.Group, bool) {
	gs := m.Record().Groups
	for i := m.Pos(); i < end && i < len(gs); i++ {
		if gs[i].Code == code {
			m.Seek(i + 1)
			return gs[i], true
		}
	}
	return dxf.Group{}, false
}

func cInt32Before(m *dxf.Mapper, code int, v *int32, end int) {
	if !m.Reading() {
		m.Int32(code, v)
		return
	}
	if g, ok := cTakeBefore(m, code, end); ok {
		*v = int32(g.Int())
	}
}

// cHandleBefore binds an optional handle group that must belong to the current cell.
func cHandleBefore(m *dxf.Mapper, x *ctx, code int, r *dwg.Ref, refCode byte, end int) {
	if !m.Reading() {
		x.HandleRefOpt(m, code, r, refCode)
		return
	}
	if g, ok := cTakeBefore(m, code, end); ok && g.Handle() != 0 {
		*r = dwg.Ref{Code: refCode, H: g.Handle()}
	}
}

// cCellOverrides binds the per-cell property overrides selected by the cell override flags.
func cCellOverrides(m *dxf.Mapper, x *ctx, t *dwg.TableCell) {
	f := t.FlagOverride
	if f&0x01 != 0 {
		m.Int16(170, &t.Alignment)
	}
	if f&0x02 != 0 {
		m.Bool(283, &t.BgFillNone)
	}
	if f&0x04 != 0 {
		dColor(m, &t.BgColor, 63, false)
	}
	if f&0x08 != 0 {
		dColor(m, &t.ContentColor, 64, false)
	}
	if f&0x10 != 0 {
		x.NameRef(m, 7, &t.TextStyle, "STYLE", 5)
	}
	if f&0x20 != 0 {
		m.Float(140, &t.TextHeight)
	}
	for e := 0; e < 4; e++ {
		if f&cEdgeColorBit[e] != 0 {
			dColor(m, &t.EdgeColor[e], cEdgeColor[e], false)
		}
		if f&cEdgeLWBit[e] != 0 {
			m.Int16(cEdgeLW[e], &t.EdgeLW[e])
		}
		if f&cEdgeVisBit[e] != 0 {
			m.Int16(cEdgeVis[e], &t.EdgeVis[e])
		}
	}
}

// cCellValue binds the R2007+ value block of a cell: 92 extended flags, 301 CELL_VALUE, 93 flags,
// 90 data type, the value (91 long, 140 double, 1 string), 94 unit, 300 format, 302 value string,
// 304 ACVALUE_END. Reading is bounded by the 304 group so a missing group never takes a later one.
func cCellValue(m *dxf.Mapper, t *dwg.TableCell) {
	v := &t.CellValue
	if !m.Reading() {
		val := *v
		if val.Flags == 0 && val.DataType == 0 { // pre-R2007 source: the text is the value
			val = cTextValue(t.Text)
		}
		w := m.Writer()
		w.Int(92, 0)
		w.Str(301, "CELL_VALUE")
		w.Int(93, int64(val.Flags))
		w.Int(90, int64(val.DataType))
		switch val.DataType {
		case 0, 1:
			w.Int(91, int64(val.Long))
		case 2:
			w.Float(140, val.Double)
		case 4:
			w.Str(1, val.Str)
		}
		w.Int(94, int64(val.Unit))
		w.Str(300, val.Format)
		w.Str(302, val.ValueStr)
		w.Str(304, "ACVALUE_END")
		return
	}
	end := cGroupIndex(m, 304)
	gs := m.Record().Groups
	if end >= len(gs) || gs[end].Code != 304 {
		return
	}
	if _, ok := cTakeBefore(m, 301, end); !ok {
		m.Seek(end + 1)
		return
	}
	if g, ok := cTakeBefore(m, 93, end); ok {
		v.Flags = int32(g.Int())
	}
	if g, ok := cTakeBefore(m, 90, end); ok {
		v.DataType = int32(g.Int())
	}
	switch v.DataType {
	case 0, 1:
		if g, ok := cTakeBefore(m, 91, end); ok {
			v.Long = int32(g.Int())
		}
	case 2:
		if g, ok := cTakeBefore(m, 140, end); ok {
			v.Double = g.Float()
		}
	case 4:
		if g, ok := cTakeBefore(m, 1, end); ok {
			v.Str = g.Value
		}
	}
	if g, ok := cTakeBefore(m, 94, end); ok {
		v.Unit = int32(g.Int())
	}
	if g, ok := cTakeBefore(m, 300, end); ok {
		v.Format = g.Value
	}
	if g, ok := cTakeBefore(m, 302, end); ok {
		v.ValueStr = g.Value
	}
	m.Seek(end + 1)
}

// cTextValue is the value of a classic text cell: a string, or the empty general value (flags as
// AutoCAD writes them in the 2007 sample).
func cTextValue(text string) dwg.TableValue {
	if text == "" {
		return dwg.TableValue{Flags: 7}
	}
	return dwg.TableValue{Flags: 6, DataType: 4, Str: text, ValueStr: text}
}

// isTableRoundTrip reports the XRECORD AutoCAD and ODA write under ACAD_XREC_ROUNDTRIP to carry
// the separate TABLECONTENT / TABLEGEOMETRY objects of a table in DXF.
func isTableRoundTrip(r *dxf.Record) bool {
	if r.Type != "XRECORD" {
		return false
	}
	for _, g := range r.Groups {
		if g.Code == 102 && strings.HasPrefix(g.Value, "ACAD_ROUNDTRIP_2008_TABLE") {
			return true
		}
	}
	return false
}
