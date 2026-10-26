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
	case i == 29:
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
		if !o.XDicMiss && o.XDic.H != 0 {
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
	lw := lineweightToDXF(e.Lineweight)
	m.Int16Def(370, &lw, -1)
	m.FloatDef(48, &e.LtypeScale, 1)
	m.Int16Def(60, &e.Invisible, 0)
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
		e.Ltype = dwg.Ref{Code: 5, H: x.byName["LTYPE"][strings.ToUpper(s)]}
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
		for _, e := range o.EED {
			gs, ok := eedToGroups(e.Data, r2007, x.d.Text)
			if !ok {
				continue
			}
			w.Str(1001, x.names[e.App.H])
			for _, g := range gs {
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
		app := x.byName["APPID"][strings.ToUpper(rest[i].Value)]
		o.EED = append(o.EED, dwg.EED{App: dwg.Ref{Code: 5, H: app}, Data: groupsToEED(rest[i+1:j], r2007, x.d.Text)})
		i = j - 1
	}
	m.Seek(m.Pos() + len(rest))
}
