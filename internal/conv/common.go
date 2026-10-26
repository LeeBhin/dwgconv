package conv

import (
	"strings"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// lineweights maps the DWG lineweight index (RC) to the DXF 370 value.
var lineweights = []int16{0, 5, 9, 13, 15, 18, 20, 25, 30, 35, 40, 50, 53, 60, 70, 80, 90, 100, 106, 120, 140, 158, 200, 211}

func lineweightToDXF(i byte) int16 {
	switch {
	case int(i) < len(lineweights):
		return lineweights[i]
	case i == 28 || i == 29: // ODA treats 28 as ByLayer as well
		return -1
	case i == 30:
		return -2
	default:
		return -3
	}
}

func lineweightFromDXF(v int16) byte {
	switch v {
	case -1:
		return 29
	case -2:
		return 30
	case -3:
		return 31
	}
	for i, w := range lineweights {
		if w == v {
			return byte(i)
		}
	}
	return 31
}

// ownership binds the reactor/xdictionary 102 groups and the 330 owner.
func (x *ctx) ownership(m *dxf.Mapper, o *dwg.Object, owner *uint64) {
	if !m.Reading() {
		w := m.Writer()
		if len(o.Reactors) > 0 {
			w.Str(102, "{ACAD_REACTORS")
			for _, r := range o.Reactors {
				w.Handle(330, r.H)
			}
			w.Str(102, "}")
		}
		if !o.XDicMiss && o.XDic.H != 0 && !x.xdicOmitted(o) {
			w.Str(102, "{ACAD_XDICTIONARY")
			w.Handle(360, o.XDic.H)
			w.Str(102, "}")
		}
		w.Handle(330, *owner)
		return
	}
	o.XDicMiss = true
	for m.Peek(102) {
		g := m.Rest()[0]
		m.Seek(m.Pos() + 1)
		var list []dwg.Ref
		for !m.Peek(102) && len(m.Rest()) > 0 {
			list = append(list, dwg.Ref{Code: 4, H: m.Rest()[0].Handle()})
			m.Seek(m.Pos() + 1)
		}
		m.Seek(m.Pos() + 1) // closing "}"
		switch strings.ToUpper(g.Value) {
		case "{ACAD_REACTORS":
			o.Reactors = list
		case "{ACAD_XDICTIONARY":
			if len(list) > 0 {
				o.XDic = dwg.Ref{Code: 3, H: list[0].H}
				o.XDicMiss = false
			}
		}
	}
	if m.Peek(330) {
		*owner = m.Rest()[0].Handle()
		m.Seek(m.Pos() + 1)
	}
}

// entityCommon binds the common entity groups (after 0/5): ownership, AcDbEntity subclass.
func (x *ctx) entityCommon(m *dxf.Mapper, o *dwg.Object) {
	e := o.Ent
	owner := o.Owner.H
	if !m.Reading() {
		switch e.EntMode {
		case 1:
			owner = x.ps
		case 2:
			owner = x.ms
		}
		if b, ok := x.blockOwner[o.Handle]; ok && e.EntMode != 0 {
			owner = b
		}
	}
	x.ownership(m, o, &owner)
	m.Subclass("AcDbEntity")
	paper := e.EntMode == 1
	if !m.Reading() {
		if paper {
			m.Bool(67, &paper)
		}
	} else if m.Has(67) {
		m.Bool(67, &paper)
	}
	x.NameRef(m, 8, &e.Layer, "LAYER", 5)
	x.ltype(m, e)
	x.color(m, e)
	m.FloatDef(48, &e.LtypeScale, 1)
	if !m.Reading() {
		inv := e.Invisible & 1 // only bit 0 is the visibility (ODA writes nothing else)
		m.Int16Def(60, &inv, 0)
	} else {
		m.Int16Def(60, &e.Invisible, 0)
	}
	lw := lineweightToDXF(e.Lineweight)
	if !m.Reading() {
		if e.Lineweight != 29 { // 29 is the canonical ByLayer and is omitted; 28 is written as -1
			m.Int16(370, &lw)
		}
	} else {
		m.Int16Def(370, &lw, -1)
	}
	if !m.Reading() {
		if e.MatFlags == 3 {
			x.HandleRef(m, 347, &e.Material, 5)
		}
		if e.PlotFlags == 3 {
			x.HandleRef(m, 390, &e.PlotStyle, 5)
		}
	} else {
		e.Lineweight = lineweightFromDXF(lw)
		if m.Has(347) {
			x.HandleRef(m, 347, &e.Material, 5)
			e.MatFlags = 3
		}
		if m.Has(390) {
			x.HandleRef(m, 390, &e.PlotStyle, 5)
			e.PlotFlags = 3
		}
		o.Owner = dwg.Ref{Code: 4, H: owner}
		switch {
		case paper && owner == x.ps:
			e.EntMode = 1
		case owner == x.ms:
			e.EntMode = 2
		default:
			e.EntMode = 0
		}
	}
}

// ltype binds group 6 from the linetype flags (00 ByLayer, 01 ByBlock, 10 Continuous, 11 handle).
func (x *ctx) ltype(m *dxf.Mapper, e *dwg.EntityCommon) {
	if !m.Reading() {
		var s string
		switch e.LtypeFlags {
		case 0:
			return
		case 1:
			s = "ByBlock"
		case 2:
			s = "Continuous"
		default:
			s = x.names[e.Ltype.H]
		}
		m.Str(6, &s)
		return
	}
	var s string
	m.Str(6, &s)
	switch strings.ToUpper(s) {
	case "", "BYLAYER":
		e.LtypeFlags = 0
	case "BYBLOCK":
		e.LtypeFlags = 1
	case "CONTINUOUS":
		e.LtypeFlags = 2
	default:
		e.LtypeFlags = 3
		e.Ltype = dwg.Ref{Code: 5, H: x.byName["LTYPE"][nameKey(s)]}
	}
}

// color binds 62/420/440 from the entity color (ENC).
func (x *ctx) color(m *dxf.Mapper, e *dwg.EntityCommon) {
	c := &e.Color
	const trueColor, transp = 0x8000, 0x2000
	if !m.Reading() {
		aci := int16(c.ACI())
		if uint16(c.Raw)&trueColor != 0 {
			aci = int16(uint16(c.Raw) & 0xFF)
		}
		m.Int16Def(62, &aci, 256)
		if uint16(c.Raw)&trueColor != 0 {
			rgb := c.RGB & 0xFFFFFF
			m.Int32(420, &rgb)
		}
		if uint16(c.Raw)&transp != 0 {
			m.Int32(440, &c.Transparency)
		}
		return
	}
	aci := int16(256)
	m.Int16Def(62, &aci, 256)
	raw := uint16(aci) & 0x1FF
	if m.Has(420) {
		var rgb int32
		m.Int32(420, &rgb)
		c.RGB = rgb | -0x3E000000 // 0xC2000000: true color method
		raw = trueColor | uint16(aci)&0xFF
	}
	if m.Has(440) {
		m.Int32(440, &c.Transparency)
		raw |= transp
	}
	c.Raw = int16(raw)
}

// xdata binds the 1001.. groups from/to the object's EED.
func (x *ctx) xdata(m *dxf.Mapper, o *dwg.Object) {
	r2007 := x.d.Version.R2007
	if !m.Reading() {
		w := m.Writer()
		mp := mappingFor(o)
		for _, e := range o.EED {
			if mp != nil && mp.skipEED != nil && mp.skipEED(x, o, e) {
				continue
			}
			gs, ok := eedToGroups(e.Data, r2007, x.d.Text)
			if !ok {
				continue
			}
			app := x.names[e.App.H]
			if foldedApps[strings.ToUpper(app)] {
				continue
			}
			if _, mtext := o.Body.(*dwg.MText); mtext && strings.EqualFold(app, "ACAD") {
				gs = dropACADBlocks(gs)
				if len(gs) == 0 {
					continue
				}
			}
			w.Str(1001, app)
			for _, g := range gs {
				if g.Code == 1005 && x.d.ByHandle[g.Handle()] == nil {
					g.Value = "0" // dangling reference, as ODA writes it
				}
				if g.Code == 1000 {
					w.Str(1000, g.Value) // \U+ escapes are converted like other strings
					continue
				}
				w.Group(g)
			}
		}
		return
	}
	rest := m.Rest()
	for i := 0; i < len(rest); i++ {
		if rest[i].Code != 1001 {
			continue
		}
		j := i + 1
		for j < len(rest) && rest[j].Code > 1001 {
			j++
		}
		app := x.byName["APPID"][nameKey(rest[i].Value)]
		o.EED = append(o.EED, dwg.EED{App: dwg.Ref{Code: 5, H: app}, Data: groupsToEED(rest[i+1:j], r2007, x.d.Text)})
		i = j - 1
	}
	m.Seek(m.Pos() + len(rest))
}

// foldedApps are xdata applications DXF does not carry as xdata: AcDbAttr is folded into
// ATTRIB/ATTDEF group 280, AcDbSavedByObjectVersion is dropped (as ODA writes R2004 drawings).
var foldedApps = map[string]bool{"ACDBATTR": true, "ACDBSAVEDBYOBJECTVERSION": true}

// foldedACADBlocks are ACAD xdata blocks that DXF carries as MTEXT groups (46, 101 Embedded Object).
var foldedACADBlocks = []string{"ACAD_MTEXT_DEFINED_HEIGHT", "ACAD_MTEXT_COLUMN_INFO"}

// dropACADBlocks removes the folded NAME_BEGIN ... NAME_END blocks from ACAD xdata.
func dropACADBlocks(gs []dxf.Group) []dxf.Group {
	out := gs[:0:0]
	for i := 0; i < len(gs); i++ {
		g := gs[i]
		folded := false
		for _, n := range foldedACADBlocks {
			if g.Code == 1000 && g.Value == n+"_BEGIN" {
				for j := i + 1; j < len(gs); j++ {
					if gs[j].Code == 1000 && gs[j].Value == n+"_END" {
						i = j
						folded = true
						break
					}
				}
			}
		}
		if !folded {
			out = append(out, g)
		}
	}
	return out
}

// consumedXRecord reports xdictionary entries ODA does not write to DXF (it folds or regenerates them).
func (x *ctx) consumedXRecord(name string) bool {
	switch name {
	case "ACAD_LAYOUTSELFREF", "ACAD_MTEXT_2008_RT":
		return true
	case "ACAD_XREC_ROUNDTRIP":
		return !x.d.Version.R2007
	}
	return false
}

// xdicOmitted reports whether an object's extension dictionary is left out of the DXF: when it
// holds only consumed entries, or (R2004 input) when it is empty — as ODA writes it.
func (x *ctx) xdicOmitted(o *dwg.Object) bool {
	if o.XDicMiss || o.XDic.H == 0 {
		return false
	}
	do := x.d.ByHandle[o.XDic.H]
	if do == nil {
		return true
	}
	dict, ok := do.Body.(*dwg.Dictionary)
	if !ok {
		return false
	}
	consumed := 0
	for _, n := range dict.Names {
		if x.consumedXRecord(n) {
			consumed++
		}
	}
	if consumed < len(dict.Names) {
		return false
	}
	return consumed > 0 || !x.d.Version.R2007 || !keepsEmptyXDic(o)
}

// keepsEmptyXDic: ODA keeps an empty extension dictionary only on table control objects (R2010+).
func keepsEmptyXDic(o *dwg.Object) bool {
	_, ok := o.Body.(*dwg.TableControl)
	return ok
}
