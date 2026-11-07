package dwg

// Conversions between the two ACAD_TABLE layouts: the pre-R2010 cell structures (§20.4.96.1), which
// DXF always carries, and the embedded table content (§20.4.96.2) of R2010+ files. The DXF mapping
// derives the classic fields before writing an R2010+ table and builds the content after reading a
// table for an R2010+ file. Only what the classic structures can express survives the trip.

// tableCellHasValue is the classic cell override bit AutoCAD sets on every cell that holds a value.
const tableCellHasValue = 0x40000

// content format / cell style property override bits (§20.4.101.3)
const (
	cfDataType  = 0x1
	cfFormat    = 0x2
	cfRotation  = 0x4
	cfScale     = 0x8
	cfAlign     = 0x10
	cfColor     = 0x20
	cfTextStyle = 0x40
	cfHeight    = 0x80
	cfBgColor   = 0x200
)

// border property override bits (§20.4.101.4)
const (
	cbLineweight = 0x2
	cbColor      = 0x8
	cbInvisible  = 0x10
)

// ClassicFromContent fills the pre-R2010 fields (grid, cells, style handle) from Content.
func (b *Table) ClassicFromContent() {
	t := &b.Content
	b.TableStyle = t.TableStyle
	if b.Flags == 0 {
		b.Flags = 22
	}
	nc := len(t.Columns)
	b.ColWidths = make([]float64, nc)
	for i := range t.Columns {
		b.ColWidths[i] = t.Columns[i].Width
	}
	b.RowHeights = make([]float64, len(t.Rows))
	b.Cells = make([]TableCell, len(t.Rows)*nc)
	for r := range t.Rows {
		row := &t.Rows[r]
		b.RowHeights[r] = row.Height
		for c := range row.Cells {
			if c >= nc {
				break
			}
			cell := &row.Cells[c]
			tc := &b.Cells[r*nc+c]
			tc.Type, tc.MergedWidth, tc.MergedHeight = 1, 1, 1
			tc.CellValue = TableValue{Flags: 7}
			if len(cell.Contents) > 0 { // the classic cell holds one content
				ct := &cell.Contents[0]
				switch ct.Type {
				case 1:
					tc.CellValue = ct.Value
					tc.Text = ct.Value.Str
					tc.HasOverride, tc.FlagOverride = true, tc.FlagOverride|tableCellHasValue
				case 2:
					tc.Value = ct.Field
				case 4:
					tc.Type, tc.Value, tc.BlockScale = 2, ct.Block, 1
					tc.HasAttrs = len(ct.Attrs) > 0
					for _, a := range ct.Attrs {
						tc.Attrs = append(tc.Attrs, TableAttr{Index: int16(a.Index), Text: a.Value, AttDef: a.Def})
					}
					tc.HasOverride, tc.FlagOverride = true, tc.FlagOverride|tableCellHasValue
				}
				if ct.HasFormat != 0 {
					if ct.Format.PropOverride&cfRotation != 0 {
						tc.Rotation = ct.Format.Rotation
					}
					if ct.Format.PropOverride&cfScale != 0 {
						tc.BlockScale = ct.Format.BlockScale
					}
					tc.classicFormat(&ct.Format, ct.Format.PropOverride)
				}
			}
			if cell.Style.DataFlags&1 != 0 {
				s := &cell.Style
				tc.classicFormat(&s.Content, s.PropOverride|s.Content.PropOverride)
				if s.PropOverride&cfBgColor != 0 {
					tc.HasOverride = true
					tc.FlagOverride |= 0x04
					tc.BgColor = s.BgColor
					if uint32(s.BgColor.RGB)>>24 == 0xC8 { // none
						tc.FlagOverride |= 0x02
						tc.BgFillNone = true
					}
				}
				for i := range s.Borders {
					bd := &s.Borders[i]
					e := edgeIndex(bd.EdgeFlags)
					if e < 0 {
						continue
					}
					if bd.Override&cbColor != 0 {
						tc.HasOverride = true
						tc.FlagOverride |= edgeColorBit[e]
						tc.EdgeColor[e] = bd.Color
					}
					if bd.Override&cbLineweight != 0 {
						tc.HasOverride = true
						tc.FlagOverride |= edgeLWBit[e]
						tc.EdgeLW[e] = int16(bd.Lineweight)
					}
					if bd.Override&cbInvisible != 0 {
						tc.HasOverride = true
						tc.FlagOverride |= edgeVisBit[e]
						tc.EdgeVis[e] = int16(1 - bd.Invisible&1)
					}
				}
			}
		}
	}
	for _, m := range t.Merged { // top row, left column, bottom row, right column
		for r := m[0]; r <= m[2]; r++ {
			for c := m[1]; c <= m[3]; c++ {
				if r < 0 || c < 0 || int(r) >= len(t.Rows) || int(c) >= nc {
					continue
				}
				tc := &b.Cells[int(r)*nc+int(c)]
				if r == m[0] && c == m[1] {
					tc.MergedWidth, tc.MergedHeight = m[3]-m[1]+1, m[2]-m[0]+1
				} else {
					tc.Merged = true
				}
			}
		}
	}
}

// classicFormat applies the overridden content format properties to the classic cell.
func (tc *TableCell) classicFormat(f *ContentFormat, over int32) {
	if over&cfAlign != 0 {
		tc.HasOverride, tc.FlagOverride, tc.Alignment = true, tc.FlagOverride|0x01, int16(f.CellAlignment)
	}
	if over&cfColor != 0 {
		tc.HasOverride, tc.FlagOverride, tc.ContentColor = true, tc.FlagOverride|0x08, f.ContentColor
	}
	if over&cfTextStyle != 0 && f.TextStyle.H != 0 {
		tc.HasOverride, tc.FlagOverride, tc.TextStyle = true, tc.FlagOverride|0x10, f.TextStyle
	}
	if over&cfHeight != 0 {
		tc.HasOverride, tc.FlagOverride, tc.TextHeight = true, tc.FlagOverride|0x20, f.TextHeight
	}
}

// edgeIndex maps a border edge flag to the classic edge index (top, right, bottom, left).
func edgeIndex(flags int32) int {
	switch flags {
	case 1:
		return 0
	case 2:
		return 1
	case 4:
		return 2
	case 8:
		return 3
	}
	return -1
}

// ContentFromClassic builds Content and the other R2010+ fields from the pre-R2010 fields.
func (b *Table) ContentFromClassic() {
	nc, nr := len(b.ColWidths), len(b.RowHeights)
	t := &b.Content
	*t = TableContent{TableStyle: b.TableStyle, Style: CellStyle{Type: 4}}
	t.Columns = make([]TableColumn, nc)
	for i := range t.Columns {
		t.Columns[i] = TableColumn{Style: CellStyle{Type: 3}, Width: b.ColWidths[i]}
	}
	t.Rows = make([]ContentRow, nr)
	for r := range t.Rows {
		row := &t.Rows[r]
		row.Style, row.StyleID, row.Height = CellStyle{Type: 2}, int32(min(r+1, 3)), b.RowHeights[r]
		row.Cells = make([]ContentCell, nc)
		for c := range row.Cells {
			cell := &row.Cells[c]
			cell.Style = CellStyle{Type: 1}
			if r*nc+c >= len(b.Cells) {
				continue
			}
			tc := &b.Cells[r*nc+c]
			switch {
			case tc.Type == 2:
				ct := CellContent{Type: 4, Block: tc.Value}
				for k, a := range tc.Attrs {
					idx := int32(a.Index)
					if idx == 0 { // the classic index is not in DXF; content indexes start at 1
						idx = int32(k + 1)
					}
					ct.Attrs = append(ct.Attrs, CellAttr{Def: a.AttDef, Value: a.Text, Index: idx})
				}
				cell.Contents = []CellContent{ct}
			case tc.Value.H != 0 && tc.CellValue.DataType == 0 && tc.Text == "":
				cell.Contents = []CellContent{{Type: 2, Field: tc.Value}}
			case tc.CellValue.DataType != 0 && tc.CellValue.Flags&1 == 0:
				cell.Contents = []CellContent{{Type: 1, Value: tc.CellValue}}
			case tc.Text != "":
				cell.Contents = []CellContent{{Type: 1, Value: TableValue{Flags: 6, DataType: 4, Str: tc.Text, ValueStr: tc.Text}}}
			}
			if tc.HasOverride {
				cell.contentOverrides(tc)
			}
			if !tc.Merged && (tc.MergedWidth > 1 || tc.MergedHeight > 1) {
				t.Merged = append(t.Merged, [4]int32{int32(r), int32(c), int32(r) + max(tc.MergedHeight, 1) - 1, int32(c) + max(tc.MergedWidth, 1) - 1})
			}
		}
	}
	b.Unknown2010B = true
	b.UnknownH = Ref{Code: 4}
	b.Unknown38 = 38 // the spec's default, as ODA writes it (AutoCAD's 2010 sample has 6)
	b.Ranges = []TableRowRange{{EndRow: int32(nr) - 1}}
	b.ColWidths, b.RowHeights, b.Cells = nil, nil, nil // an R2010+ table holds the content only
}

// contentOverrides turns the classic cell overrides into a cell style with data.
func (cell *ContentCell) contentOverrides(tc *TableCell) {
	s := &cell.Style
	f := &s.Content
	fo := tc.FlagOverride
	var over int32
	if fo&0x01 != 0 {
		over, f.CellAlignment = over|cfAlign, int32(tc.Alignment)
	}
	if fo&0x08 != 0 {
		over, f.ContentColor = over|cfColor, tc.ContentColor
	}
	if fo&0x10 != 0 && tc.TextStyle.H != 0 {
		over, f.TextStyle = over|cfTextStyle, tc.TextStyle
	}
	if fo&0x20 != 0 {
		over, f.TextHeight = over|cfHeight, tc.TextHeight
	}
	if fo&0x04 != 0 {
		over, s.BgColor = over|cfBgColor, tc.BgColor
	}
	for e := 0; e < 4; e++ {
		if fo&(edgeColorBit[e]|edgeLWBit[e]|edgeVisBit[e]) == 0 {
			continue
		}
		bd := CellBorder{EdgeFlags: 1 << e, Type: 1}
		if fo&edgeColorBit[e] != 0 {
			bd.Override, bd.Color = bd.Override|cbColor, tc.EdgeColor[e]
		}
		if fo&edgeLWBit[e] != 0 {
			bd.Override, bd.Lineweight = bd.Override|cbLineweight, int32(tc.EdgeLW[e])
		}
		if fo&edgeVisBit[e] != 0 {
			bd.Override, bd.Invisible = bd.Override|cbInvisible, int32(1-tc.EdgeVis[e]&1)
		}
		s.Borders = append(s.Borders, bd)
	}
	if over == 0 && len(s.Borders) == 0 {
		return
	}
	s.DataFlags, s.PropOverride, f.PropOverride = 1, over, over&(cfAlign|cfColor|cfTextStyle|cfHeight)
	if f.TextStyle.H == 0 {
		f.TextStyle = Ref{Code: 5}
	}
}
