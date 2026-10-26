package dwg

func init() {
	bodyFactory[0x14] = func() Body { return &DimOrdinate{} }
	bodyFactory[0x15] = func() Body { return &DimLinear{} }
	bodyFactory[0x16] = func() Body { return &DimAligned{} }
	bodyFactory[0x17] = func() Body { return &DimAng3Pt{} }
	bodyFactory[0x18] = func() Body { return &DimAng2Ln{} }
	bodyFactory[0x19] = func() Body { return &DimRadius{} }
	bodyFactory[0x1A] = func() Body { return &DimDiameter{} }
	classBodyFactory["ARC_DIMENSION"] = func() Body { return &DimArc{} }
	classBodyFactory["LARGE_RADIAL_DIMENSION"] = func() Body { return &DimLargeRadial{} }
}

// DimCommon is the common dimension data (§20.4.22) plus the DIMSTYLE/BLOCK handles.
type DimCommon struct {
	Version       byte // R2010+, 280
	Extrusion     [3]float64
	TextMidpt     [2]float64 // 11
	Elevation     float64
	Flags1        byte
	UserText      string // 1
	TextRot       float64
	HorizDir      float64 // 51
	InsScale      [3]float64
	InsRotation   float64 // 54
	Attachment    int16   // 71
	LSpaceStyle   int16   // 72
	LSpaceFactor  float64 // 41
	ActualMeasure float64 // 42
	Unknown73     bool    // R2007+
	FlipArrow1    bool    // 74
	FlipArrow2    bool    // 75
	Pt12          [2]float64
	DimStyle      Ref // 3
	Block         Ref // 2
}

func (d *DimCommon) dimData(c *codec) {
	if c.R2010 {
		c.RC(&d.Version)
	}
	c.BD3(&d.Extrusion)
	c.RD2(&d.TextMidpt)
	c.BD(&d.Elevation)
	c.RC(&d.Flags1)
	c.TV(&d.UserText)
	c.BD(&d.TextRot)
	c.BD(&d.HorizDir)
	c.BD3(&d.InsScale)
	c.BD(&d.InsRotation)
	c.BS(&d.Attachment)
	c.BS(&d.LSpaceStyle)
	c.BD(&d.LSpaceFactor)
	c.BD(&d.ActualMeasure)
	if c.R2007 {
		c.B(&d.Unknown73)
		c.B(&d.FlipArrow1)
		c.B(&d.FlipArrow2)
	}
	c.RD2(&d.Pt12)
}

func (d *DimCommon) handles(c *codec, o *Object) {
	c.H(&d.DimStyle)
	c.H(&d.Block)
}

// DimOrdinate §20.4.23.
type DimOrdinate struct {
	DimCommon
	Pt10, Pt13, Pt14 [3]float64
	Flags2           byte
}

func (b *DimOrdinate) data(c *codec, o *Object) {
	b.dimData(c)
	c.BD3(&b.Pt10)
	c.BD3(&b.Pt13)
	c.BD3(&b.Pt14)
	c.RC(&b.Flags2)
}

// DimLinear §20.4.24.
type DimLinear struct {
	DimCommon
	Pt13, Pt14, Pt10 [3]float64
	ExtLineRot       float64 // 52
	DimRot           float64 // 50
}

func (b *DimLinear) data(c *codec, o *Object) {
	b.dimData(c)
	c.BD3(&b.Pt13)
	c.BD3(&b.Pt14)
	c.BD3(&b.Pt10)
	c.BD(&b.ExtLineRot)
	c.BD(&b.DimRot)
}

// DimAligned §20.4.25.
type DimAligned struct {
	DimCommon
	Pt13, Pt14, Pt10 [3]float64
	ExtLineRot       float64 // 52
}

func (b *DimAligned) data(c *codec, o *Object) {
	b.dimData(c)
	c.BD3(&b.Pt13)
	c.BD3(&b.Pt14)
	c.BD3(&b.Pt10)
	c.BD(&b.ExtLineRot)
}

// DimAng3Pt §20.4.26.
type DimAng3Pt struct {
	DimCommon
	Pt10, Pt13, Pt14, Pt15 [3]float64
}

func (b *DimAng3Pt) data(c *codec, o *Object) {
	b.dimData(c)
	c.BD3(&b.Pt10)
	c.BD3(&b.Pt13)
	c.BD3(&b.Pt14)
	c.BD3(&b.Pt15)
}

// DimAng2Ln §20.4.27.
type DimAng2Ln struct {
	DimCommon
	Pt16                   [2]float64
	Pt13, Pt14, Pt15, Pt10 [3]float64
}

func (b *DimAng2Ln) data(c *codec, o *Object) {
	b.dimData(c)
	c.RD2(&b.Pt16)
	c.BD3(&b.Pt13)
	c.BD3(&b.Pt14)
	c.BD3(&b.Pt15)
	c.BD3(&b.Pt10)
}

// DimRadius §20.4.28.
type DimRadius struct {
	DimCommon
	Pt10, Pt15 [3]float64
	LeaderLen  float64 // 40
}

func (b *DimRadius) data(c *codec, o *Object) {
	b.dimData(c)
	c.BD3(&b.Pt10)
	c.BD3(&b.Pt15)
	c.BD(&b.LeaderLen)
}

// DimDiameter §20.4.29.
type DimDiameter struct {
	DimCommon
	Pt15, Pt10 [3]float64
	LeaderLen  float64 // 40
}

func (b *DimDiameter) data(c *codec, o *Object) {
	b.dimData(c)
	c.BD3(&b.Pt15)
	c.BD3(&b.Pt10)
	c.BD(&b.LeaderLen)
}

// DimArc ARC_DIMENSION §20.4.19.
type DimArc struct {
	DimCommon
	Pt10, Pt13, Pt14, Center [3]float64 // 10, 13, 14, 15
	IsPartial                bool       // 70
	StartAngle, EndAngle     float64    // 40, 41
	HasLeader                bool       // 71
	Leader1, Leader2         [3]float64 // 16, 17
}

func (b *DimArc) data(c *codec, o *Object) {
	b.dimData(c)
	c.BD3(&b.Pt10)
	c.BD3(&b.Pt13)
	c.BD3(&b.Pt14)
	c.BD3(&b.Center)
	c.B(&b.IsPartial)
	c.BD(&b.StartAngle)
	c.BD(&b.EndAngle)
	c.B(&b.HasLeader)
	c.BD3(&b.Leader1)
	c.BD3(&b.Leader2)
}

// DimLargeRadial LARGE_RADIAL_DIMENSION §20.4.30.
type DimLargeRadial struct {
	DimCommon
	Center, Chord  [3]float64 // 10, 13
	Unknown40      float64
	OverrideCenter [3]float64 // 14
	Jog            [3]float64 // 15
}

func (b *DimLargeRadial) data(c *codec, o *Object) {
	b.dimData(c)
	c.BD3(&b.Center)
	c.BD3(&b.Chord)
	c.BD(&b.Unknown40)
	c.BD3(&b.OverrideCenter)
	c.BD3(&b.Jog)
}
