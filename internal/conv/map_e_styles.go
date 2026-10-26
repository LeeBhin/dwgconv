package conv

import (
	"sort"
	"strconv"
	"strings"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

func init() {
	mls := register("MLEADERSTYLE", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.MLeaderStyle) {
		r2010 := x.d.Version.R2010
		m.Subclass("AcDbMLeaderStyle")
		// R2004 files lack the R2010 fields; ODA writes their defaults.
		eInt16Ver(m, 179, &b.Version, r2010, 2)
		m.Int16(170, &b.ContentType)
		m.Int16(171, &b.DrawMLeaderOrder)
		m.Int16(172, &b.DrawLeaderOrder)
		m.Int32(90, &b.MaxPoints)
		m.Float(40, &b.FirstSegAngle)
		m.Float(41, &b.SecondSegAngle)
		m.Int16(173, &b.LeaderType)
		m.Int32(91, &b.LeaderColor.RGB)
		x.HandleRef(m, 340, &b.LeaderLtype, 5)
		m.Int32(92, &b.LeaderLineweight)
		m.Bool(290, &b.LandingEnabled)
		m.Float(42, &b.LandingGap)
		m.Bool(291, &b.DoglegEnabled)
		m.Float(43, &b.LandingDistance)
		m.Str(3, &b.Description)
		x.HandleRef(m, 341, &b.ArrowHead, 5)
		m.Float(44, &b.ArrowHeadSize)
		m.Str(300, &b.TextDefault)
		x.HandleRef(m, 342, &b.TextStyle, 5)
		m.Int16(174, &b.LeftAttachment)
		m.Int16(178, &b.RightAttachment)
		m.Int16(175, &b.TextAngleType)
		m.Int16(176, &b.TextAlignType)
		m.Int32(93, &b.TextColor.RGB)
		m.Float(45, &b.TextHeight)
		m.Bool(292, &b.TextFrame)
		m.Bool(297, &b.AlignTextLeft)
		m.Float(46, &b.AlignSpace)
		x.HandleRef(m, 343, &b.Block, 5)
		m.Int32(94, &b.BlockColor.RGB)
		m.Float(47, &b.BlockScale[0])
		m.Float(49, &b.BlockScale[1])
		m.Float(140, &b.BlockScale[2])
		m.Bool(293, &b.BlockScaleOn)
		m.Float(141, &b.BlockRotation)
		m.Bool(294, &b.BlockRotationOn)
		m.Int16(177, &b.BlockConnection)
		m.Float(142, &b.ScaleFactor)
		m.Bool(295, &b.PropertyChanged)
		m.Bool(296, &b.Annotative)
		m.Float(143, &b.BreakSize)
		eInt16Ver(m, 271, &b.AttachDirection, r2010, 0)
		eInt16Ver(m, 272, &b.BottomAttachment, r2010, 9)
		eInt16Ver(m, 273, &b.TopAttachment, r2010, 9)
		if r2010 {
			m.Bool(298, &b.UnknownBool)
		} else {
			// R2004: ODA takes 298 from the ACAD_ROUNDTRIP_2012_MLEADER_STYLE round-trip record (default 0).
			v := eRoundTripValue(x, o, "ACAD_ROUNDTRIP_2012_MLEADER_STYLE", 298) == "1"
			m.Bool(298, &v)
		}
		if m.Reading() {
			b.NewFormat = true // ODA always writes the new-format fields
			if app, ok := x.byName["APPID"]["ACAD_MLEADERVER"]; ok && !r2010 {
				// R2004 marks the new format with ACAD_MLEADERVER EED (1070 = 2), which ODA drops.
				o.EED = append(o.EED, dwg.EED{App: dwg.Ref{Code: 5, H: app},
					Data: groupsToEED([]dxf.Group{{Code: 1070, Value: "2"}}, false, x.d.Text)})
			}
		}
	})
	// ODA consumes the ACAD_MLEADERVER marker EED (new-format flag) and does not write it.
	mls.skipEED = func(x *ctx, o *dwg.Object, e dwg.EED) bool {
		return strings.EqualFold(x.names[e.App.H], "ACAD_MLEADERVER")
	}
}

// eInt16Ver binds a field that only exists in some DWG versions; when absent, def is written.
func eInt16Ver(m *dxf.Mapper, code int, v *int16, present bool, def int16) {
	if present {
		m.Int16(code, v)
		return
	}
	d := def
	m.Int16(code, &d)
}

func eAtoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// eRoundTripValue returns the value of group code after the 102 marker in the object's
// ACAD_XREC_ROUNDTRIP extension record, or "" when absent.
func eRoundTripValue(x *ctx, o *dwg.Object, marker string, code int) string {
	xd := x.d.ByHandle[o.XDic.H]
	if o.XDicMiss || xd == nil || eDict(xd) == nil {
		return ""
	}
	dd := eDict(xd)
	for i, n := range dd.Names {
		if n != "ACAD_XREC_ROUNDTRIP" {
			continue
		}
		xr, ok := x.d.ByHandle[dd.Items[i].H].Body.(*dwg.XRecord)
		if !ok {
			return ""
		}
		in := false
		for _, g := range eXRecordGroups(xr.Data, x.d.Version.R2007, x.d.Text) {
			switch {
			case g.Code == 102:
				in = g.Value == marker
			case in && g.Code == code:
				return g.Value
			}
		}
	}
	return ""
}

func init() {
	register("TABLESTYLE", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.TableStyle) {
		if !m.Reading() && x.d.Version.R2010 {
			cp := *b // classic fields derived for output only
			eTableStyleFromR24(x, &cp)
			b = &cp
		}
		m.Subclass("AcDbTableStyle")
		m.Str(3, &b.Description)
		m.Int16(70, &b.FlowDirection)
		m.Int16(71, &b.BitFlags)
		m.Float(40, &b.HorzMargin)
		m.Float(41, &b.VertMargin)
		m.Bool(280, &b.SuppressTitle)
		m.Bool(281, &b.SuppressHeader)
		r2007 := x.d.Version.R2007
		for i := range b.Rows {
			r := &b.Rows[i]
			x.NameRef(m, 7, &r.TextStyle, "STYLE", 5)
			m.Float(140, &r.TextHeight)
			m.Int16(170, &r.TextAlign)
			eCMC(m, 62, &r.TextColor)
			if !m.Reading() && !r.BgEnabled {
				none := int16(257) // ODA writes "none" when the background fill is off
				m.Int16(63, &none)
			} else {
				eCMC(m, 63, &r.FillColor)
			}
			m.Bool(283, &r.BgEnabled)
			if m.Reading() && !r.BgEnabled && uint32(r.FillColor.RGB)>>24 == 0xC8 {
				r.FillColor = dwg.CmColor{RGB: eRGB(0xC3, 7)} // AutoCAD's stored fill when disabled
			}
			if r2007 {
				m.Int32(90, &r.DataType)
				m.Int32(91, &r.UnitType)
				m.Str(1, &r.FormatStr)
			} else {
				// Not stored before R2007: ODA takes them from the ACAD_ROUNDTRIP_PRE2007_TABLESTYLE
				// round-trip record (row i: 90+2i, 91+2i, 1+i), default 512, 0, "".
				const rt = "ACAD_ROUNDTRIP_PRE2007_TABLESTYLE"
				dt, ut := int32(512), int32(0)
				if v := eRoundTripValue(x, o, rt, 90+2*i); v != "" {
					dt = int32(eAtoi(v))
				}
				if v := eRoundTripValue(x, o, rt, 91+2*i); v != "" {
					ut = int32(eAtoi(v))
				}
				fs := eRoundTripValue(x, o, rt, 1+i)
				m.Int32(90, &dt)
				m.Int32(91, &ut)
				m.Str(1, &fs)
			}
			for k := range r.Borders {
				bd := &r.Borders[k]
				m.Int16(274+k, &bd.Lineweight)
				m.Bool(284+k, &bd.Visible)
				eCMC(m, 64+k, &bd.Color)
			}
		}
		if m.Reading() && x.d.Version.R2010 {
			eTableStyleToR24(b)
			// the classic fields are not stored in R2010+ files
			*b = dwg.TableStyle{Description: b.Description, Unknown1: b.Unknown1, Unknown2: b.Unknown2,
				Unknown3: b.Unknown3, UnknownOwner: b.UnknownOwner, TableCell: b.TableCell, CellStyles: b.CellStyles}
		}
	})
}

// eTableStyleToR24 builds the R24 (R2010+) TABLESTYLE body from the classic DXF fields, filling the
// rest with the values AutoCAD writes for a default table style ("Table" cell style plus _TITLE,
// _HEADER and _DATA). The classic DXF record does not carry the full cell style map.
func eTableStyleToR24(b *dwg.TableStyle) {
	b.Unknown1, b.Unknown2, b.Unknown3 = 0, 0, 101
	b.UnknownOwner = dwg.Ref{Code: 3}
	b.TableCell = dwg.CellStyle{
		Type: 5, DataFlags: 1, BgColor: dwg.CmColor{RGB: eRGB(0xC8, 0)}, LayoutFlags: 1,
		Content: dwg.ContentFormat{ValueType: 512, BlockScale: 1, CellAlignment: 1,
			ContentColor: dwg.CmColor{RGB: eRGB(0xC1, 0)}, TextStyle: dwg.Ref{Code: 5}, TextHeight: 0.18},
		MarginOverride: 1,
		Margins:        [6]float64{b.VertMargin, b.HorzMargin, b.VertMargin, b.HorzMargin, 0.18, 0.18},
		ID:             4, Class: 2, Name: "Table",
	}
	if b.FlowDirection == 1 {
		b.TableCell.PropOverride |= 0x10000
	}
	type spec struct {
		row        int
		id, class  int32
		name       string
		mergeFlags int32
	}
	b.CellStyles = b.CellStyles[:0]
	b.LeadIDs = nil
	for _, sp := range []spec{{1, 1, 1, "_TITLE", 0x8000}, {2, 2, 1, "_HEADER", 0}, {0, 3, 2, "_DATA", 0}} {
		r := &b.Rows[sp.row]
		bg := r.FillColor
		if !r.BgEnabled {
			bg = dwg.CmColor{RGB: eRGB(0xC8, 0)}
		}
		vt := r.DataType
		if vt == 0 || vt == 512 {
			vt = 4
		}
		cs := dwg.CellStyle{
			Type: 5, DataFlags: 1, MergeFlags: sp.mergeFlags, BgColor: bg, LayoutFlags: 1,
			Content: dwg.ContentFormat{ValueType: vt, ValueUnit: r.UnitType, ValueFormat: r.FormatStr, BlockScale: 1,
				CellAlignment: int32(r.TextAlign), ContentColor: r.TextColor, TextStyle: dwg.Ref{Code: 5, H: r.TextStyle.H},
				TextHeight: r.TextHeight},
			ID: sp.id, Class: sp.class, Name: sp.name,
		}
		if b.HorzMargin != 0.06 || b.VertMargin != 0.06 {
			// non-default margins are repeated as overrides on each cell style (observed)
			cs.MarginOverride = 1
			cs.Margins = b.TableCell.Margins
		}
		for k, edge := range eTableBorderEdges {
			bd := r.Borders[k]
			inv := int32(0)
			if !bd.Visible {
				inv = 1
			}
			cs.Borders = append(cs.Borders, dwg.CellBorder{EdgeFlags: edge, Type: 1, Color: bd.Color,
				Lineweight: int32(bd.Lineweight), Ltype: dwg.Ref{Code: 5}, Invisible: inv, DoubleSpace: 0.045})
		}
		// R24 orders borders by edge flag
		sort.Slice(cs.Borders, func(i, j int) bool { return cs.Borders[i].EdgeFlags < cs.Borders[j].EdgeFlags })
		b.CellStyles = append(b.CellStyles, cs)
	}
}

// eTableBorderEdges maps the classic border order (top, inside horizontal, bottom, left, inside vertical,
// right) to the R24 cell style edge flags.
var eTableBorderEdges = [6]int32{1, 0x20, 4, 8, 0x10, 2}

// eTableStyleFromR24 fills the classic TABLESTYLE fields, which ODA writes to DXF, from the R24 cell styles.
func eTableStyleFromR24(x *ctx, b *dwg.TableStyle) {
	b.HorzMargin = b.TableCell.Margins[1]
	b.VertMargin = b.TableCell.Margins[0]
	if b.TableCell.PropOverride&0x10000 != 0 {
		b.FlowDirection = 1
	}
	byID := map[int32]*dwg.CellStyle{}
	for i := range b.CellStyles {
		byID[b.CellStyles[i].ID] = &b.CellStyles[i]
	}
	for i, id := range [3]int32{3, 1, 2} { // data, title, header
		cs := byID[id]
		if cs == nil {
			continue
		}
		r := &b.Rows[i]
		r.TextStyle = cs.Content.TextStyle
		r.TextHeight = cs.Content.TextHeight
		r.TextAlign = int16(cs.Content.CellAlignment)
		r.TextColor = cs.Content.ContentColor
		r.FillColor = cs.BgColor
		r.BgEnabled = uint32(cs.BgColor.RGB)>>24 != 0xC8
		r.DataType = cs.Content.ValueType
		r.UnitType = cs.Content.ValueUnit
		r.FormatStr = cs.Content.ValueFormat
		for k, edge := range eTableBorderEdges {
			bd := &r.Borders[k]
			bd.Lineweight, bd.Visible, bd.Color = -2, true, dwg.CmColor{RGB: eRGB(0xC1, 0)}
			for _, cb := range cs.Borders {
				if cb.EdgeFlags == edge {
					bd.Lineweight = int16(cb.Lineweight)
					bd.Visible = cb.Invisible == 0
					bd.Color = cb.Color
				}
			}
		}
	}
}

func init() {
	register("CELLSTYLEMAP", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.CellStyleMap) {
		m.Subclass("AcDbCellStyleMap")
		n := m.Count(90, len(b.CellStyles))
		if m.Reading() {
			b.CellStyles = make([]dwg.CellStyle, n)
		}
		for i := range b.CellStyles {
			eCellStyle(m, x, &b.CellStyles[i])
		}
	})
}

// eMarker writes a fixed marker group (e.g. 1=TABLEFORMAT_BEGIN) or skips it on read.
func eMarker(m *dxf.Mapper, code int, s string) {
	v := s
	m.Str(code, &v)
}

// eCellStyle binds a cell style (§20.4.101.4) in ODA's marker layout.
func eCellStyle(m *dxf.Mapper, x *ctx, s *dwg.CellStyle) {
	eMarker(m, 300, "CELLSTYLE")
	eMarker(m, 1, "TABLEFORMAT_BEGIN")
	m.Int32(90, &s.Type)
	m.Int16(170, &s.DataFlags)
	if s.DataFlags&1 != 0 {
		m.Int32(91, &s.PropOverride)
		m.Int32(92, &s.MergeFlags)
		eCMC(m, 62, &s.BgColor)
		m.Int32(93, &s.LayoutFlags)
		f := &s.Content
		eMarker(m, 300, "CONTENTFORMAT")
		eMarker(m, 1, "CONTENTFORMAT_BEGIN")
		m.Int32(90, &f.PropOverride)
		m.Int32(91, &f.PropFlags)
		m.Int32(92, &f.ValueType)
		m.Int32(93, &f.ValueUnit)
		m.Str(300, &f.ValueFormat)
		m.Float(40, &f.Rotation)
		m.Float(140, &f.BlockScale)
		m.Int32(94, &f.CellAlignment)
		eCMC(m, 62, &f.ContentColor)
		x.HandleRef(m, 340, &f.TextStyle, 5)
		m.Float(144, &f.TextHeight)
		eMarker(m, 309, "CONTENTFORMAT_END")
		m.Int16(171, &s.MarginOverride)
		if s.MarginOverride&1 != 0 {
			eMarker(m, 301, "MARGIN")
			eMarker(m, 1, "CELLMARGIN_BEGIN")
			for k := range s.Margins {
				m.Float(40, &s.Margins[k])
			}
			eMarker(m, 309, "CELLMARGIN_END")
		}
		n := m.Count(94, len(s.Borders))
		if m.Reading() {
			s.Borders = make([]dwg.CellBorder, n)
		}
		for k := range s.Borders {
			bd := &s.Borders[k]
			m.Int32(95, &bd.EdgeFlags)
			if bd.EdgeFlags == 0 {
				continue
			}
			eMarker(m, 302, "GRIDFORMAT")
			eMarker(m, 1, "GRIDFORMAT_BEGIN")
			m.Int32(90, &bd.Override)
			m.Int32(91, &bd.Type)
			eCMC(m, 62, &bd.Color)
			m.Int32(92, &bd.Lineweight)
			x.HandleRef(m, 340, &bd.Ltype, 5)
			m.Int32(93, &bd.Invisible)
			m.Float(40, &bd.DoubleSpace)
			eMarker(m, 309, "GRIDFORMAT_END")
		}
	}
	eMarker(m, 309, "TABLEFORMAT_END")
	eMarker(m, 1, "CELLSTYLE_BEGIN")
	m.Int32(90, &s.ID)
	m.Int32(91, &s.Class)
	m.Str(300, &s.Name)
	eMarker(m, 309, "CELLSTYLE_END")
}
