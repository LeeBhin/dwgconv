package conv

import (
	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// ACAD_TABLE (pre-R2010 body only; R2010+ tables are not decoded). Unverified against ODA.
// Covers the block reference part, the grid and the basic cell contents; per-cell and
// table-level property overrides follow the AutoCAD DXF reference codes.

func init() {
	register("ACAD_TABLE", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Table) {
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
		m.Int16(280, &ver)
		x.HandleRef(m, 342, &b.TableStyle, 5)
		x.HandleRef(m, 343, &b.BlockHeader, 5)
		m.Point3(11, &b.HorDir)
		m.Int16(90, &b.Flags)
		nr := m.Count(91, len(b.RowHeights))
		nc := m.Count(92, len(b.ColWidths))
		m.Int32(93, &b.TableOverride)
		m.Int32(94, &b.BorderColorFlags)
		m.Int32(95, &b.BorderLWFlags)
		m.Int32(96, &b.BorderVisFlags)
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
		for i := range b.Cells {
			t := &b.Cells[i]
			m.Int16(171, &t.Type)
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
				x.HandleRefOpt(m, 344, &t.Value, 5)
				m.Str(1, &t.Text)
			case 2:
				x.HandleRefOpt(m, 340, &t.Value, 5)
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
		}
	})
}
