package conv

import (
	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// VISUALSTYLE and MATERIAL DXF records as ODA writes them (layouts inferred from the samples).

func init() {
	vs := register("VISUALSTYLE", 0, dVisualStyleRec)
	vs.skipEED = dSkipSavedByVersion
	register("MATERIAL", 0, dMaterialRec)
}

// dVSDefaults are the R2010+ properties 29-58 ODA writes for R2004 visual styles, which have no
// such fields (the values match the R2010+ samples). Properties 29-33 depend on the type.
func dVSDefaults(typ int16) []dwg.VSProp {
	p := func(i int32) dwg.VSProp { return dwg.VSProp{Int: i, Op: 1} }
	r := func(v float64) dwg.VSProp { return dwg.VSProp{Real: v, Op: 1} }
	bo := func(v bool) dwg.VSProp { return dwg.VSProp{Bool: v, Op: 1} }
	tc := func(rgb int32) dwg.VSProp {
		return dwg.VSProp{Color: dwg.CmColor{RGB: rgb | -0x3E000000}, Op: 1} // 0xC2: true color
	}
	first := []bool{false, true, true, false, false}
	if typ == 4 { // 2dWireframe
		first = []bool{true, true, false, false, false}
	}
	var out []dwg.VSProp
	for _, v := range first {
		out = append(out, bo(v))
	}
	out = append(out, bo(false), bo(false), bo(false), bo(false), // 34-37
		p(50), r(0), r(1), p(0), tc(0), p(50), p(3), tc(0xFF), bo(false), // 38-46
		p(50), p(50), p(50), bo(false), p(50), // 47-51
		dwg.VSProp{Color: dwg.CmColor{RGB: -0x40000000}, Op: 0},                 // 52: ByLayer
		dwg.VSProp{Real: 1, Op: 0},                                              // 53
		p(2), dwg.VSProp{Text: "strokes_ogs.tif", Op: 1}, bo(false), r(1), r(1)) // 54-58
	return out
}

// dVSLegacyProps builds the 58 properties ODA writes for an R2004 visual style.
func dVSLegacyProps(typ int16, l *dwg.VSLegacy) []dwg.VSProp {
	p := func(i int32) dwg.VSProp { return dwg.VSProp{Int: i, Op: 1} }
	r := func(v float64) dwg.VSProp { return dwg.VSProp{Real: v, Op: 1} }
	c := func(v dwg.CmColor) dwg.VSProp { return dwg.VSProp{Color: v, Op: 1} }
	out := []dwg.VSProp{
		p(int32(l.FaceLightingModel)), p(int32(l.FaceLightingQuality)), p(int32(l.FaceColorMode)), p(l.FaceModifier),
		r(l.FaceOpacity), r(l.FaceSpecular), c(l.FaceColor), p(int32(l.EdgeModel)), p(int32(l.EdgeStyle)),
		c(l.EdgeIntersectColor), c(l.EdgeObscuredColor), p(int32(l.EdgeObscuredLtype)), p(int32(l.EdgeIntersectLtype)),
		r(l.EdgeCreaseAngle), p(l.EdgeModifier), c(l.EdgeColor), r(l.EdgeOpacity), p(int32(l.EdgeWidth)),
		p(int32(l.EdgeOverhang)), p(int32(l.EdgeJitter)), c(l.EdgeSilhouetteColor), p(int32(l.EdgeSilhouetteWidth)),
		p(int32(l.EdgeHaloGap)), p(int32(l.EdgeIsolines)), {Bool: l.EdgeHidePrecision, Op: 1},
		p(l.DisplaySettings), r(float64(l.Brightness)), p(int32(l.ShadowType)),
	}
	out = append(out, dVSDefaults(typ)...)
	// The "...Off" styles (20-22) write operation 0 (2 for the edge modifier) for properties
	// 1-33, and for all 58 when the face color is a true color. Observed on 6 objects only
	// (s6, s7, s8); the two cases differ in nothing but the face color.
	if typ >= 20 && typ <= 22 {
		n := 33
		if uint32(l.FaceColor.RGB)>>24 == dTrueColor {
			n = len(out)
		}
		for i := 0; i < n; i++ {
			out[i].Op = 0
		}
		out[14].Op = 2
	}
	return out
}

// dVSLegacyFrom fills the R2004 fields from imported properties.
func dVSLegacyFrom(ps []dwg.VSProp) *dwg.VSLegacy {
	g := func(i int) dwg.VSProp {
		if i-1 < len(ps) {
			return ps[i-1]
		}
		return dwg.VSProp{}
	}
	s := func(i int) int16 { return int16(g(i).Int) }
	return &dwg.VSLegacy{
		FaceLightingModel: s(1), FaceLightingQuality: s(2), FaceColorMode: s(3), FaceModifier: g(4).Int,
		FaceOpacity: g(5).Real, FaceSpecular: g(6).Real, FaceColor: g(7).Color, EdgeModel: s(8), EdgeStyle: s(9),
		EdgeIntersectColor: g(10).Color, EdgeObscuredColor: g(11).Color, EdgeObscuredLtype: s(12),
		EdgeIntersectLtype: s(13), EdgeCreaseAngle: g(14).Real, EdgeModifier: g(15).Int, EdgeColor: g(16).Color,
		EdgeOpacity: g(17).Real, EdgeWidth: s(18), EdgeOverhang: s(19), EdgeJitter: s(20),
		EdgeSilhouetteColor: g(21).Color, EdgeSilhouetteWidth: s(22), EdgeHaloGap: byte(g(23).Int),
		EdgeIsolines: s(24), EdgeHidePrecision: g(25).Bool, DisplaySettings: g(26).Int,
		Brightness: int32(g(27).Real), ShadowType: s(28),
	}
}

func dVisualStyleRec(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.VisualStyle) {
	r2010 := x.d.Version.R2010
	m.Subclass("AcDbVisualStyle")
	m.Str(2, &b.Name)
	m.Int16(70, &b.Type)
	f177 := b.Flags + 1
	if !r2010 {
		f177 = 3
	}
	m.Int16(177, &f177)
	m.Bool(291, &b.Internal)
	props := b.Props
	if !m.Reading() && !r2010 {
		props = dVSLegacyProps(b.Type, b.Legacy)
	}
	n := m.Count(70, len(props))
	if m.Reading() {
		props = make([]dwg.VSProp, n)
	}
	for i := range props {
		code := 0
		if i < len(dwg.VSPropCodes) {
			code = dwg.VSPropCodes[i]
		}
		pr := &props[i]
		switch code {
		case 90:
			m.Int32(90, &pr.Int)
		case 40:
			m.Float(40, &pr.Real)
		case 62:
			dVSColor(m, &pr.Color)
		case 290:
			m.Bool(290, &pr.Bool)
		case 1:
			m.Str(1, &pr.Text)
		}
		m.Int16(176, &pr.Op)
	}
	if m.Reading() {
		if r2010 {
			b.Flags = f177 - 1
			b.Props = props
		} else {
			b.Legacy = dVSLegacyFrom(props)
		}
	}
}

// dVSColor binds a property color: 62, and 420 right after it for a true color.
// (dColor looks further ahead for 420, which here would find a later property's group.)
func dVSColor(m *dxf.Mapper, c *dwg.CmColor) {
	if !m.Reading() {
		dColor(m, c, 62, false)
		return
	}
	var i int16
	m.Int16(62, &i)
	*c = dCmcFromACI(i)
	if m.Peek(420) {
		var rgb int32
		m.Int32(420, &rgb)
		c.RGB = rgb&0xFFFFFF | -0x3E000000 // 0xC2: true color
	}
}

// dSkipSavedByVersion drops the ACAD "AcDbSavedByObjectVersion" EED, which ODA does not write.
func dSkipSavedByVersion(x *ctx, o *dwg.Object, e dwg.EED) bool {
	gs, ok := eedToGroups(e.Data, x.d.Version.R2007, x.d.Text)
	return ok && len(gs) == 2 && gs[0].Code == 1000 && gs[0].Value == "AcDbSavedByObjectVersion"
}

var dIdentity = [16]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}

// dMatColor binds a color channel (method, factor, value), each omitted when default.
func dMatColor(m *dxf.Mapper, codes [3]int, c *dwg.MaterialColor) {
	method := int16(c.Method)
	m.Int16Def(codes[0], &method, 0)
	c.Method = byte(method)
	m.FloatDef(codes[1], &c.Factor, 1)
	if c.Method == 1 {
		m.Int32(codes[2], &c.Value)
	}
}

// dMatMap binds a map: blend, source, file name, projection, tiling, auto transform, matrix.
// srcMode: 0 write the source always, 1 only when not 1, 2 never (refraction).
func dMatMap(m *dxf.Mapper, codes [7]int, mp *dwg.MaterialMap, srcMode int) {
	m.FloatDef(codes[0], &mp.Blend, 1)
	src := int16(mp.Source)
	switch {
	case srcMode == 0:
		m.Int16(codes[1], &src)
	case srcMode == 1:
		m.Int16Def(codes[1], &src, 1)
	case m.Reading():
		src = 0
		if m.Has(codes[1]) {
			m.Int16(codes[1], &src)
		}
	}
	mp.Source = byte(src)
	m.StrOpt(codes[2], &mp.FileName)
	for i, p := range []*byte{&mp.Projection, &mp.Tiling, &mp.AutoTransform} {
		v := int16(*p)
		m.Int16Def(codes[3+i], &v, 1)
		*p = byte(v)
	}
	if !m.Reading() {
		if mp.Matrix != dIdentity {
			for i := range mp.Matrix {
				m.Float(codes[6], &mp.Matrix[i])
			}
		}
		return
	}
	mp.Matrix = dIdentity
	if m.Has(codes[6]) {
		for i := range mp.Matrix {
			m.Float(codes[6], &mp.Matrix[i])
		}
	}
}

func dMaterialRec(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Material) {
	m.Subclass("AcDbMaterial")
	m.Str(1, &b.Name)
	m.StrOpt(2, &b.Description)
	dMatColor(m, [3]int{70, 40, 90}, &b.Ambient)
	dMatColor(m, [3]int{71, 41, 91}, &b.Diffuse)
	dMatMap(m, [7]int{42, 72, 3, 73, 74, 75, 43}, &b.DiffuseMap, 0)
	m.FloatDef(44, &b.Gloss, 0.5)
	dMatColor(m, [3]int{76, 45, 92}, &b.Specular)
	dMatMap(m, [7]int{46, 77, 4, 78, 79, 170, 47}, &b.SpecularMap, 1)
	dMatMap(m, [7]int{48, 171, 6, 172, 173, 174, 49}, &b.ReflectionMap, 1)
	m.FloatDef(140, &b.Opacity, 1)
	dMatMap(m, [7]int{141, 175, 7, 176, 177, 178, 142}, &b.OpacityMap, 1)
	dMatMap(m, [7]int{143, 179, 8, 270, 271, 272, 144}, &b.BumpMap, 1)
	m.FloatDef(145, &b.RefractionIndex, 1)
	dMatMap(m, [7]int{146, 273, 9, 274, 275, 276, 147}, &b.RefractionMap, 2)
	flags := b.ChannelFlags
	if !x.d.Version.R2010 {
		// R2004 has no channel flags; ODA writes 127 when the diffuse map source is 0 and 63
		// otherwise (the R2010+ samples hold the same pairing).
		flags = 63
		if b.DiffuseMap.Source == 0 {
			flags = 127
		}
	}
	m.Int32(94, &flags)
	if m.Reading() && x.d.Version.R2010 {
		b.ChannelFlags = flags
	}
}
