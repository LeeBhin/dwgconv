package dwg

func init() {
	bodyFactory[0x22] = func() Body { return &ViewportEnt{} }
	bodyFactory[0x4A] = func() Body { return &Ole2Frame{} }
	classBodyFactory["OLE2FRAME"] = func() Body { return &Ole2Frame{} }
}

// ViewportEnt is the VIEWPORT entity (§20.4.38, R2000+ form).
type ViewportEnt struct {
	Center        [3]float64 // 10
	Width, Height float64    // 40, 41
	ViewTarget    [3]float64 // 17
	ViewDir       [3]float64 // 16
	TwistAngle    float64    // 51
	ViewHeight    float64    // 45
	LensLength    float64    // 42
	FrontClip     float64    // 43
	BackClip      float64    // 44
	SnapAngle     float64    // 50
	ViewCenter    [2]float64 // 12
	SnapBase      [2]float64 // 13
	SnapSpacing   [2]float64 // 14
	GridSpacing   [2]float64 // 15
	CircleZoom    int16      // 72
	GridMajor     int16      // 61, R2007+
	StatusFlags   int32      // 90
	StyleSheet    string     // 1
	RenderMode    byte       // 281
	UCSAtOrigin   bool       // 74
	UCSPerVport   bool       // 71
	UCSOrigin     [3]float64 // 110
	UCSXAxis      [3]float64 // 111
	UCSYAxis      [3]float64 // 112
	UCSElevation  float64    // 146
	UCSOrthoType  int16      // 79
	ShadePlotMode int16      // 170, R2004+
	UseDefLights  bool       // 292, R2007+
	DefLightType  byte       // 282
	Brightness    float64    // 141
	Contrast      float64    // 142
	AmbientColor  CmColor    // 63
	FrozenLayers  []Ref      // 341
	ClipBoundary  Ref        // 340
	NamedUCS      Ref        // 345
	BaseUCS       Ref        // 346
	Background    Ref        // 332, R2007+
	VisualStyle   Ref        // 348
	ShadePlot     Ref        // 333
	Sun           Ref        // 361
}

func (b *ViewportEnt) data(c *codec, o *Object) {
	c.BD3(&b.Center)
	c.BD(&b.Width)
	c.BD(&b.Height)
	c.BD3(&b.ViewTarget)
	c.BD3(&b.ViewDir)
	c.BD(&b.TwistAngle)
	c.BD(&b.ViewHeight)
	c.BD(&b.LensLength)
	c.BD(&b.FrontClip)
	c.BD(&b.BackClip)
	c.BD(&b.SnapAngle)
	c.RD2(&b.ViewCenter)
	c.RD2(&b.SnapBase)
	c.RD2(&b.SnapSpacing)
	c.RD2(&b.GridSpacing)
	c.BS(&b.CircleZoom)
	if c.R2007 {
		c.BS(&b.GridMajor)
	}
	n := c.BLCount(len(b.FrozenLayers))
	b.FrozenLayers = Slice(c, b.FrozenLayers, n)
	c.BL(&b.StatusFlags)
	c.TV(&b.StyleSheet)
	c.RC(&b.RenderMode)
	c.B(&b.UCSAtOrigin)
	c.B(&b.UCSPerVport)
	c.BD3(&b.UCSOrigin)
	c.BD3(&b.UCSXAxis)
	c.BD3(&b.UCSYAxis)
	c.BD(&b.UCSElevation)
	c.BS(&b.UCSOrthoType)
	c.BS(&b.ShadePlotMode)
	if c.R2007 {
		c.B(&b.UseDefLights)
		c.RC(&b.DefLightType)
		c.BD(&b.Brightness)
		c.BD(&b.Contrast)
		c.CMC(&b.AmbientColor)
	}
}

func (b *ViewportEnt) handles(c *codec, o *Object) {
	for i := range b.FrozenLayers {
		c.H(&b.FrozenLayers[i])
	}
	c.H(&b.ClipBoundary)
	c.H(&b.NamedUCS)
	c.H(&b.BaseUCS)
	if c.R2007 {
		c.H(&b.Background)
		c.H(&b.VisualStyle)
		c.H(&b.ShadePlot)
		c.H(&b.Sun)
	}
}

// Ole2Frame §20.4.88.
type Ole2Frame struct {
	Flags   int16  // 70
	Mode    int16  `conv:"derived"` // not in DXF (72 there is the tile mode)
	Data    []byte // OLE2 data (310)
	Unknown byte
}

func (b *Ole2Frame) data(c *codec, o *Object) {
	c.BS(&b.Flags)
	c.BS(&b.Mode)
	n := c.BLCount(len(b.Data))
	c.Raw(&b.Data, n)
	c.RC(&b.Unknown)
}

func (b *Ole2Frame) handles(c *codec, o *Object) {}
