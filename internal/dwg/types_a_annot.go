package dwg

func init() {
	bodyFactory[0x2C] = func() Body { return &MText{} }
	bodyFactory[0x21] = func() Body { return &Shape{} }
	bodyFactory[0x2E] = func() Body { return &Tolerance{} }
	bodyFactory[0x2D] = func() Body { return &Leader{} }
}

// MText §20.4.46.
type MText struct {
	Insertion     [3]float64
	Extrusion     [3]float64
	XAxis         [3]float64
	RectWidth     float64
	RectHeight    float64 // R2007+
	TextHeight    float64
	Attachment    int16
	DrawingDir    int16
	ExtHeight     float64 `conv:"derived"`
	ExtWidth      float64 `conv:"derived"`
	Value         string
	LineSpStyle   int16
	LineSpFactor  float64
	UnknownBit    bool  `conv:"derived"`
	BgFlags       int32 // R2004+
	BgScale       float64
	BgColor       CmColor
	BgTransp      int32
	NotAnnotative bool       `conv:"derived"` // R2018+
	AVersion      int16      `conv:"derived"`
	ADefault      bool       `conv:"derived"`
	ARegApp       Ref        `conv:"derived"`
	AAttachment   int32      `conv:"derived"`
	AXAxis        [3]float64 `conv:"derived"`
	AInsertion    [3]float64 `conv:"derived"`
	ARectWidth    float64    `conv:"derived"`
	ARectHeight   float64    `conv:"derived"`
	AExtWidth     float64    `conv:"derived"`
	AExtHeight    float64    `conv:"derived"`
	ColType       int16
	ColHeights    []float64
	ColCount      int32
	ColWidth      float64
	ColGutter     float64
	ColAutoHeight bool
	ColFlowRev    bool
	Style         Ref
}

func (m *MText) data(c *codec, o *Object) {
	c.BD3(&m.Insertion)
	c.BD3(&m.Extrusion)
	c.BD3(&m.XAxis)
	c.BD(&m.RectWidth)
	if c.R2007 {
		c.BD(&m.RectHeight)
	}
	c.BD(&m.TextHeight)
	c.BS(&m.Attachment)
	c.BS(&m.DrawingDir)
	c.BD(&m.ExtHeight)
	c.BD(&m.ExtWidth)
	c.TV(&m.Value)
	c.BS(&m.LineSpStyle)
	c.BD(&m.LineSpFactor)
	c.B(&m.UnknownBit)
	c.BL(&m.BgFlags)
	if m.BgFlags&1 != 0 || (c.R2018 && m.BgFlags&0x10 != 0) {
		c.BD(&m.BgScale)
		c.CMC(&m.BgColor)
		c.BL(&m.BgTransp)
	}
	if !c.R2018 {
		return
	}
	c.B(&m.NotAnnotative)
	if !m.NotAnnotative {
		return
	}
	c.BS(&m.AVersion)
	c.B(&m.ADefault)
	c.BL(&m.AAttachment)
	c.BD3(&m.AXAxis)
	c.BD3(&m.AInsertion)
	c.BD(&m.ARectWidth)
	c.BD(&m.ARectHeight)
	c.BD(&m.AExtWidth)
	c.BD(&m.AExtHeight)
	c.BS(&m.ColType)
	if m.ColType != 0 {
		var n int
		if m.ColType == 2 && !m.ColAutoHeight {
			n = len(m.ColHeights)
		}
		cnt := int32(n)
		if c.w && m.ColType != 2 {
			cnt = m.ColCount
		}
		c.BL(&cnt)
		m.ColCount = cnt
		c.BD(&m.ColWidth)
		c.BD(&m.ColGutter)
		c.B(&m.ColAutoHeight)
		c.B(&m.ColFlowRev)
		if !m.ColAutoHeight && m.ColType == 2 {
			m.ColHeights = Slice(c, m.ColHeights, int(cnt))
			for i := range m.ColHeights {
				c.BD(&m.ColHeights[i])
			}
		}
	}
}

func (m *MText) handles(c *codec, o *Object) {
	c.H(&m.Style)
	if c.R2018 && m.NotAnnotative {
		c.H(&m.ARegApp)
	}
}

// Shape §20.4.37.
type Shape struct {
	Insertion [3]float64
	Scale     float64
	Rotation  float64
	WidthFact float64
	Oblique   float64
	Thickness float64
	ShapeNo   int16
	Extrusion [3]float64
	ShapeFile Ref
}

func (b *Shape) data(c *codec, o *Object) {
	c.BD3(&b.Insertion)
	c.BD(&b.Scale)
	c.BD(&b.Rotation)
	c.BD(&b.WidthFact)
	c.BD(&b.Oblique)
	c.BD(&b.Thickness)
	c.BS(&b.ShapeNo)
	c.BD3(&b.Extrusion)
}
func (b *Shape) handles(c *codec, o *Object) { c.H(&b.ShapeFile) }

// Tolerance §20.4.49.
type Tolerance struct {
	Insertion [3]float64
	XDir      [3]float64
	Extrusion [3]float64
	Value     string
	DimStyle  Ref
}

func (b *Tolerance) data(c *codec, o *Object) {
	c.BD3(&b.Insertion)
	c.BD3(&b.XDir)
	c.BD3(&b.Extrusion)
	c.TV(&b.Value)
}
func (b *Tolerance) handles(c *codec, o *Object) { c.H(&b.DimStyle) }

// Leader §20.4.47.
type Leader struct {
	UnknownBit  bool `conv:"derived"`
	AnnotType   int16
	PathType    int16
	Points      [][3]float64
	Origin      [3]float64 `conv:"derived"` // not in DXF
	Extrusion   [3]float64
	XDir        [3]float64
	OffsetBlock [3]float64
	EndPtProj   [3]float64
	BoxHeight   float64
	BoxWidth    float64
	HookXDir    bool
	ArrowheadOn bool
	Unk1        int16 `conv:"derived"`
	Unk2, Unk3  bool  `conv:"derived"`
	Annotation  Ref
	DimStyle    Ref
}

func (b *Leader) data(c *codec, o *Object) {
	c.B(&b.UnknownBit)
	c.BS(&b.AnnotType)
	c.BS(&b.PathType)
	n := c.BLCount(len(b.Points))
	b.Points = Slice(c, b.Points, n)
	for i := range b.Points {
		c.BD3(&b.Points[i])
	}
	c.BD3(&b.Origin)
	c.BD3(&b.Extrusion)
	c.BD3(&b.XDir)
	c.BD3(&b.OffsetBlock)
	c.BD3(&b.EndPtProj)
	c.BD(&b.BoxHeight)
	c.BD(&b.BoxWidth)
	c.B(&b.HookXDir)
	c.B(&b.ArrowheadOn)
	// Spec lists "R2000+: BS, B, B" here; R2013 data ends right after ArrowheadOn.
	// Kept for R2004 per spec (unverified, no R2004 LEADER in samples).
	if !c.R2010 {
		c.BS(&b.Unk1)
		c.B(&b.Unk2)
		c.B(&b.Unk3)
	}
}

func (b *Leader) handles(c *codec, o *Object) {
	c.H(&b.Annotation)
	c.H(&b.DimStyle)
}
