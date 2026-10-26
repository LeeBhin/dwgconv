package conv

import (
	"math"
	"strconv"
	"strings"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// dDimStyle binds a DIMSTYLE record. ODA writes a variable only when it differs from the
// AutoCAD default (imperial STANDARD), except DIMCLRT (178) which it always writes.
func dDimStyle(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DimStyle) {
	dRecord(m, &b.TableEntry, "AcDbDimStyleTableRecord", 0)
	r2007 := x.d.Version.R2007
	str := func(code int, v *string) { m.StrOpt(code, v) }
	fl := func(code int, v *float64, def float64) { m.FloatDef(code, v, def) }
	i16 := func(code int, v *int16, def int16) { m.Int16Def(code, v, def) }
	bl := func(code int, v *bool, def bool) {
		if !m.Reading() {
			if *v != def {
				m.Bool(code, v)
			}
			return
		}
		*v = def
		if m.Has(code) {
			m.Bool(code, v)
		}
	}
	str(3, &b.DIMPOST)
	str(4, &b.DIMAPOST)
	fl(40, &b.DIMSCALE, 1)
	fl(41, &b.DIMASZ, 0.18)
	fl(42, &b.DIMEXO, 0.0625)
	fl(43, &b.DIMDLI, 0.38)
	fl(44, &b.DIMEXE, 0.18)
	fl(45, &b.DIMRND, 0)
	fl(46, &b.DIMDLE, 0)
	fl(47, &b.DIMTP, 0)
	fl(48, &b.DIMTM, 0)
	if r2007 {
		fl(49, &b.DIMFXL, 1)
		m.AngleDef(50, &b.DIMJOGANG, math.Pi/4)
		i16(69, &b.DIMTFILL, 0)
		dDimColorDef(m, 70, &b.DIMTFILLCLR, 0)
	}
	ev := dDimEEDVars(x, o)
	switch {
	case r2007:
		i16(90, &b.DIMARCSYM, 0)
	case !m.Reading():
		if v, ok := ev[379]; ok {
			a := int16(v.Int())
			m.Int16(90, &a)
		}
	case m.Has(90):
		var a int16
		m.Int16(90, &a)
		dDimAddEED(x, o, "ACAD_DSTYLE_DIMARC_LENGTH_SYMBOL", 379, dxf.Group{Code: 1070, Value: strconv.Itoa(int(a))})
	}
	bl(71, &b.DIMTOL, false)
	bl(72, &b.DIMLIM, false)
	bl(73, &b.DIMTIH, true)
	bl(74, &b.DIMTOH, true)
	bl(75, &b.DIMSE1, false)
	bl(76, &b.DIMSE2, false)
	i16(77, &b.DIMTAD, 0)
	i16(78, &b.DIMZIN, 0)
	i16(79, &b.DIMAZIN, 0)
	fl(140, &b.DIMTXT, 0.18)
	fl(141, &b.DIMCEN, 0.09)
	fl(142, &b.DIMTSZ, 0)
	fl(143, &b.DIMALTF, 25.4)
	fl(144, &b.DIMLFAC, 1)
	fl(145, &b.DIMTVP, 0)
	fl(146, &b.DIMTFAC, 1)
	fl(147, &b.DIMGAP, 0.09)
	fl(148, &b.DIMALTRND, 0)
	bl(170, &b.DIMALT, false)
	i16(171, &b.DIMALTD, 2)
	bl(172, &b.DIMTOFL, false)
	bl(173, &b.DIMSAH, false)
	bl(174, &b.DIMTIX, false)
	bl(175, &b.DIMSOXD, false)
	clrd := dDimColorDef(m, 176, &b.DIMCLRD, 0)
	clre := dDimColorDef(m, 177, &b.DIMCLRE, 0)
	if !m.Reading() && (clrd || clre) && b.DIMDEC != 4 {
		// ODA repeats a non-default DIMDEC here when a dimension/extension line color is written.
		m.Int16(271, &b.DIMDEC)
	}
	dDimColor(m, 178, &b.DIMCLRT)
	i16(179, &b.DIMADEC, 0)
	i16(271, &b.DIMDEC, 4)
	i16(272, &b.DIMTDEC, 4)
	i16(273, &b.DIMALTU, 2)
	i16(274, &b.DIMALTTD, 2)
	i16(275, &b.DIMAUNIT, 0)
	i16(276, &b.DIMFRAC, 0)
	i16(277, &b.DIMLUNIT, 2)
	i16(278, &b.DIMDSEP, '.')
	i16(279, &b.DIMTMOVE, 0)
	i16(280, &b.DIMJUST, 0)
	bl(281, &b.DIMSD1, false)
	bl(282, &b.DIMSD2, false)
	i16(283, &b.DIMTOLJ, 1)
	i16(284, &b.DIMTZIN, 0)
	i16(285, &b.DIMALTZ, 0)
	i16(286, &b.DIMALTTZ, 0)
	bl(288, &b.DIMUPT, false)
	i16(289, &b.DIMATFIT, 3)
	if r2007 {
		bl(290, &b.DIMFXLON, false)
	}
	if x.d.Version.R2010 {
		bl(294, &b.DIMTXTDIRECTION, false)
	}
	x.HandleRef(m, 340, &b.DIMTXSTY, 5)
	x.HandleRefOpt(m, 341, &b.DIMLDRBLK, 5)
	x.HandleRefOpt(m, 342, &b.DIMBLK, 5)
	x.HandleRefOpt(m, 343, &b.DIMBLK1, 5)
	x.HandleRefOpt(m, 344, &b.DIMBLK2, 5)
	i16(371, &b.DIMLWD, -2)
	i16(372, &b.DIMLWE, -2)
	if r2007 {
		x.HandleRefOpt(m, 345, &b.DIMLTYPE, 5)
		x.HandleRefOpt(m, 346, &b.DIMLTEX1, 5)
		x.HandleRefOpt(m, 347, &b.DIMLTEX2, 5)
		return
	}
	// R2004 keeps the dimension linetypes in EED; ODA writes them as 345-347.
	for i, app := range []string{"ACAD_DSTYLE_DIM_LINETYPE", "ACAD_DSTYLE_DIM_EXT1_LINETYPE", "ACAD_DSTYLE_DIM_EXT2_LINETYPE"} {
		code := 345 + i
		if !m.Reading() {
			if v, ok := ev[380+i]; ok {
				h := v.Handle()
				m.Handle(code, &h)
			}
		} else if m.Has(code) {
			var h uint64
			m.Handle(code, &h)
			dDimAddEED(x, o, app, 380+i, dxf.Group{Code: 1005, Value: strconv.FormatUint(h, 16)})
		}
	}
}

// dDimEEDCodes are the R2004 DIMSTYLE EED blocks (by their 1070 marker) that ODA writes as groups.
var dDimEEDCodes = map[int]bool{379: true, 380: true, 381: true, 382: true}

// dDimEEDVars returns the value group of each R2004 DIMSTYLE EED variable, keyed by its 1070 marker.
func dDimEEDVars(x *ctx, o *dwg.Object) map[int]dxf.Group {
	out := map[int]dxf.Group{}
	if x.d.Version.R2007 {
		return out
	}
	for _, e := range o.EED {
		if k, v, ok := dDimEEDVar(x, e); ok {
			out[k] = v
		}
	}
	return out
}

func dDimEEDVar(x *ctx, e dwg.EED) (int, dxf.Group, bool) {
	gs, ok := eedToGroups(e.Data, false, x.d.Text)
	if !ok || len(gs) != 2 || gs[0].Code != 1070 {
		return 0, dxf.Group{}, false
	}
	k := int(gs[0].Int())
	if !dDimEEDCodes[k] {
		return 0, dxf.Group{}, false
	}
	return k, gs[1], true
}

// dDimSkipEED reports whether an EED block is written as DIMSTYLE groups instead of xdata.
func dDimSkipEED(x *ctx, o *dwg.Object, e dwg.EED) bool {
	if x.d.Version.R2007 {
		return false
	}
	_, _, ok := dDimEEDVar(x, e)
	return ok
}

// dDimAddEED recreates an R2004 DIMSTYLE EED block on import.
func dDimAddEED(x *ctx, o *dwg.Object, app string, marker int, v dxf.Group) {
	gs := []dxf.Group{{Code: 1070, Value: strconv.Itoa(marker)}, v}
	h := x.byName["APPID"][strings.ToUpper(app)]
	o.EED = append(o.EED, dwg.EED{App: dwg.Ref{Code: 5, H: h}, Data: groupsToEED(gs, false, x.d.Text)})
}

// dDimColorDef binds a DIMSTYLE color omitted when its ACI equals def; it reports whether it was written.
func dDimColorDef(m *dxf.Mapper, code int, c *dwg.CmColor, def int16) bool {
	if !m.Reading() {
		if dCmcACI(c) == def && uint32(c.RGB)>>24 != dTrueColor {
			return false
		}
		dDimColor(m, code, c)
		return true
	}
	*c = dCmcFromACI(def)
	if m.Has(code) {
		dDimColor(m, code, c)
	}
	return false
}

func dDimColor(m *dxf.Mapper, code int, c *dwg.CmColor) { dColor(m, c, code, false) }
