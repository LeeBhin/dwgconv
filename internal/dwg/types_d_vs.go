package dwg

// VISUALSTYLE and MATERIAL are not in the specification. Their layouts were inferred from the
// sample drawings (R2004, R2013, R2018) by pairing the object bits with ODA's DXF records.

func init() {
	classBodyFactory["VISUALSTYLE"] = func() Body { return &VisualStyle{} }
	classBodyFactory["MATERIAL"] = func() Body { return &Material{} }
}

// VSPropCodes is the DXF group code of each R2010+ visual style property, in stored order.
// It also fixes the DWG code: 90 BL, 40 BD, 62 CMC, 290 B, 1 TV.
var VSPropCodes = [58]int{
	90, 90, 90, 90, 40, 40, 62, 90, 90, 62, 62, 90, 90, 40, 90, 62, 40, 90, 90, 90,
	62, 90, 90, 90, 290, 90, 40, 90, 290, 290, 290, 290, 290, 290, 290, 290, 290, 90, 40, 40,
	90, 62, 90, 90, 62, 290, 90, 90, 90, 290, 90, 62, 40, 90, 1, 290, 40, 40,
}

// VSProp is one visual style property: the value in the field its code selects, and the
// operation (DXF 176).
type VSProp struct {
	Int   int32
	Real  float64
	Color CmColor
	Bool  bool
	Text  string
	Op    int16
}

func (p *VSProp) code(c *codec, kind int) {
	switch kind {
	case 90:
		c.BL(&p.Int)
	case 40:
		c.BD(&p.Real)
	case 62:
		c.CMC(&p.Color)
	case 290:
		c.B(&p.Bool)
	case 1:
		c.TV(&p.Text)
	}
	c.BS(&p.Op)
}

// VSLegacy is the R2004 visual style, a fixed field list. The comment gives the R2010+
// property number (1-based) each field corresponds to.
type VSLegacy struct {
	FaceLightingModel   int16   // 1
	FaceLightingQuality int16   // 2
	FaceColorMode       int16   // 3
	FaceOpacity         float64 // 5
	FaceSpecular        float64 // 6
	FaceColor           CmColor // 7
	FaceModifier        int32   // 4
	EdgeModel           int16   // 8
	EdgeStyle           int16   // 9
	EdgeIntersectColor  CmColor // 10
	EdgeObscuredColor   CmColor // 11
	EdgeObscuredLtype   int16   // 12
	EdgeCreaseAngle     float64 // 14
	EdgeModifier        int32   // 15
	EdgeColor           CmColor // 16
	EdgeOpacity         float64 // 17
	EdgeWidth           int16   // 18
	EdgeOverhang        int16   // 19
	EdgeJitter          int16   // 20
	EdgeSilhouetteColor CmColor // 21
	EdgeSilhouetteWidth int16   // 22
	EdgeHaloGap         byte    // 23
	EdgeIsolines        int16   // 24
	EdgeHidePrecision   bool    // 25
	Unknown             int16   `conv:"derived"` // always 0 in the samples; no DXF property
	EdgeIntersectLtype  int16   // 13
	DisplaySettings     int32   // 26
	Brightness          int32   // 27 (an integer here, a real in R2010+)
	ShadowType          int16   // 28
}

func (l *VSLegacy) code(c *codec) {
	c.BS(&l.FaceLightingModel)
	c.BS(&l.FaceLightingQuality)
	c.BS(&l.FaceColorMode)
	c.BD(&l.FaceOpacity)
	c.BD(&l.FaceSpecular)
	c.CMC(&l.FaceColor)
	c.BL(&l.FaceModifier)
	c.BS(&l.EdgeModel)
	c.BS(&l.EdgeStyle)
	c.CMC(&l.EdgeIntersectColor)
	c.CMC(&l.EdgeObscuredColor)
	c.BS(&l.EdgeObscuredLtype)
	c.BD(&l.EdgeCreaseAngle)
	c.BL(&l.EdgeModifier)
	c.CMC(&l.EdgeColor)
	c.BD(&l.EdgeOpacity)
	c.BS(&l.EdgeWidth)
	c.BS(&l.EdgeOverhang)
	c.BS(&l.EdgeJitter)
	c.CMC(&l.EdgeSilhouetteColor)
	c.BS(&l.EdgeSilhouetteWidth)
	c.RC(&l.EdgeHaloGap)
	c.BS(&l.EdgeIsolines)
	c.B(&l.EdgeHidePrecision)
	c.BS(&l.Unknown)
	c.BS(&l.EdgeIntersectLtype)
	c.BL(&l.DisplaySettings)
	c.BL(&l.Brightness)
	c.BS(&l.ShadowType)
}

// VisualStyle (class VISUALSTYLE). R2010+ stores a header and 58 properties; R2004 stores
// the legacy field list.
type VisualStyle struct {
	Name     string // 2
	Type     int16  // 70
	Flags    int16  // R2010+; DXF 177 is Flags+1 (always 2 -> 3 in the samples)
	Internal bool   // 291
	Props    []VSProp
	Legacy   *VSLegacy
}

func (b *VisualStyle) data(c *codec, o *Object) {
	c.TV(&b.Name)
	c.BS(&b.Type)
	if !c.R2010 {
		if b.Legacy == nil {
			b.Legacy = &VSLegacy{}
		}
		b.Legacy.code(c)
		c.B(&b.Internal)
		return
	}
	c.BS(&b.Flags)
	c.B(&b.Internal)
	if !c.w {
		b.Props = make([]VSProp, len(VSPropCodes))
	}
	for i, k := range VSPropCodes {
		if i < len(b.Props) {
			b.Props[i].code(c, k)
		}
	}
}

func (b *VisualStyle) handles(c *codec, o *Object) {}

// MaterialColor is a material color channel: method (0 current, 1 override) and factor.
// The value is assumed to follow only for method 1 (no sample overrides a color).
type MaterialColor struct {
	Method byte
	Factor float64
	Value  int32
}

func (m *MaterialColor) code(c *codec) {
	c.RC(&m.Method)
	c.BD(&m.Factor)
	if m.Method == 1 {
		c.BL(&m.Value)
	}
}

// MaterialMap is a material map. The file name is stored only when the source is 1 (image file).
type MaterialMap struct {
	Blend         float64
	Projection    byte
	Tiling        byte
	AutoTransform byte
	Matrix        [16]float64
	Source        byte
	FileName      string
}

func (m *MaterialMap) code(c *codec) {
	c.BD(&m.Blend)
	c.RC(&m.Projection)
	c.RC(&m.Tiling)
	c.RC(&m.AutoTransform)
	for i := range m.Matrix {
		c.BD(&m.Matrix[i])
	}
	c.RC(&m.Source)
	if m.Source == 1 {
		c.TV(&m.FileName)
	}
}

// Material (class MATERIAL).
type Material struct {
	Name            string // 1
	Description     string // 2
	Ambient         MaterialColor
	Diffuse         MaterialColor
	DiffuseMap      MaterialMap
	Specular        MaterialColor
	SpecularMap     MaterialMap
	Gloss           float64 // 44
	ReflectionMap   MaterialMap
	Opacity         float64 // 140
	OpacityMap      MaterialMap
	BumpMap         MaterialMap
	RefractionIndex float64 // 145
	RefractionMap   MaterialMap
	// R2010+ tail: four reals (0 in all samples), the channel flags (DXF 94) and a real.
	Tail         [4]float64 `conv:"derived"`
	ChannelFlags int32      // 94
	Tail2        float64    `conv:"derived"`
}

func (b *Material) data(c *codec, o *Object) {
	c.TV(&b.Name)
	c.TV(&b.Description)
	b.Ambient.code(c)
	b.Diffuse.code(c)
	b.DiffuseMap.code(c)
	b.Specular.code(c)
	b.SpecularMap.code(c)
	c.BD(&b.Gloss)
	b.ReflectionMap.code(c)
	c.BD(&b.Opacity)
	b.OpacityMap.code(c)
	b.BumpMap.code(c)
	c.BD(&b.RefractionIndex)
	b.RefractionMap.code(c)
	if c.R2010 {
		for i := range b.Tail {
			c.BD(&b.Tail[i])
		}
		c.BL(&b.ChannelFlags)
		c.BD(&b.Tail2)
	}
}

func (b *Material) handles(c *codec, o *Object) {}
