package dwg

func init() {
	classBodyFactory["SPATIAL_INDEX"] = func() Body { return &SpatialIndex{} }
	classBodyFactory["LAYOUT"] = func() Body { return &Layout{} }
	bodyFactory[0x52] = func() Body { return &Layout{} }
	classBodyFactory["PLOTSETTINGS"] = func() Body { return &PlotSettingsObj{} }
	bodyFactory[0x49] = func() Body { return &MLineStyle{} }
	classBodyFactory["MLINESTYLE"] = func() Body { return &MLineStyle{} }
}

// SpatialIndex §20.4.95. The index payload is undocumented and kept verbatim.
type SpatialIndex struct {
	Timestamp1, Timestamp2 int32
	Payload                []byte
	PayloadBits            int
}

func (b *SpatialIndex) data(c *codec, o *Object) {
	c.BL(&b.Timestamp1)
	c.BL(&b.Timestamp2)
	eRestBits(c, &b.Payload, &b.PayloadBits)
}
func (b *SpatialIndex) handles(c *codec, o *Object) {}

// PlotSettings are the plot settings fields shared by PLOTSETTINGS and LAYOUT (§20.4.84, R2004+ form).
type PlotSettings struct {
	PageSetupName     string
	PrinterConfig     string
	PlotLayoutFlags   int16
	LeftMargin        float64
	BottomMargin      float64
	RightMargin       float64
	TopMargin         float64
	PaperWidth        float64
	PaperHeight       float64
	PaperSize         string
	PlotOrigin        [2]float64
	PaperUnits        int16
	PlotRotation      int16
	PlotType          int16
	WindowMin         [2]float64
	WindowMax         [2]float64
	RealWorldUnits    float64
	DrawingUnits      float64
	CurrentStyleSheet string
	ScaleType         int16
	ScaleFactor       float64
	PaperImageOrigin  [2]float64
	ShadePlotMode     int16
	ShadePlotResLvl   int16
	ShadePlotDPI      int16
	PlotView          Ref // hard pointer (handle stream)
}

func (p *PlotSettings) data(c *codec) {
	c.TV(&p.PageSetupName)
	c.TV(&p.PrinterConfig)
	c.BS(&p.PlotLayoutFlags)
	c.BD(&p.LeftMargin)
	c.BD(&p.BottomMargin)
	c.BD(&p.RightMargin)
	c.BD(&p.TopMargin)
	c.BD(&p.PaperWidth)
	c.BD(&p.PaperHeight)
	c.TV(&p.PaperSize)
	c.BD2(&p.PlotOrigin)
	c.BS(&p.PaperUnits)
	c.BS(&p.PlotRotation)
	c.BS(&p.PlotType)
	c.BD2(&p.WindowMin)
	c.BD2(&p.WindowMax)
	c.BD(&p.RealWorldUnits)
	c.BD(&p.DrawingUnits)
	c.TV(&p.CurrentStyleSheet)
	c.BS(&p.ScaleType)
	c.BD(&p.ScaleFactor)
	c.BD2(&p.PaperImageOrigin)
	c.BS(&p.ShadePlotMode)
	c.BS(&p.ShadePlotResLvl)
	c.BS(&p.ShadePlotDPI)
}

// PlotSettingsObj is the standalone PLOTSETTINGS object.
type PlotSettingsObj struct {
	PlotSettings
	VisualStyle Ref
}

func (b *PlotSettingsObj) data(c *codec, o *Object) { b.PlotSettings.data(c) }
func (b *PlotSettingsObj) handles(c *codec, o *Object) {
	c.H(&b.PlotView)
	if c.R2007 {
		c.H(&b.VisualStyle)
	}
}

// Layout §20.4.84.
type Layout struct {
	PlotSettings
	Name          string
	TabOrder      int32
	Flags         int16
	UCSOrigin     [3]float64
	LimMin        [2]float64
	LimMax        [2]float64
	InsBase       [3]float64
	UCSXAxis      [3]float64
	UCSYAxis      [3]float64
	Elevation     float64
	OrthoViewType int16
	ExtMin        [3]float64
	ExtMax        [3]float64
	VisualStyle   Ref
	BlockRecord   Ref
	LastViewport  Ref
	BaseUCS       Ref
	NamedUCS      Ref
	Viewports     []Ref
}

func (b *Layout) data(c *codec, o *Object) {
	b.PlotSettings.data(c)
	c.TV(&b.Name)
	c.BL(&b.TabOrder)
	c.BS(&b.Flags)
	c.BD3(&b.UCSOrigin)
	c.RD2(&b.LimMin)
	c.RD2(&b.LimMax)
	c.BD3(&b.InsBase)
	c.BD3(&b.UCSXAxis)
	c.BD3(&b.UCSYAxis)
	c.BD(&b.Elevation)
	c.BS(&b.OrthoViewType)
	c.BD3(&b.ExtMin)
	c.BD3(&b.ExtMax)
	n := c.BLCount(len(b.Viewports)) // spec says RL; real files use BL
	b.Viewports = Slice(c, b.Viewports, n)
}

func (b *Layout) handles(c *codec, o *Object) {
	c.H(&b.PlotView)
	if c.R2007 {
		c.H(&b.VisualStyle)
	}
	c.H(&b.BlockRecord)
	c.H(&b.LastViewport)
	c.H(&b.BaseUCS)
	c.H(&b.NamedUCS)
	for i := range b.Viewports {
		c.H(&b.Viewports[i])
	}
}

// MLineStyle §20.4.73.
type MLineStyle struct {
	Name       string
	Desc       string
	Flags      int16 // DWG bit layout (see spec for DXF mapping)
	FillColor  CmColor
	StartAngle float64
	EndAngle   float64
	Lines      []MLineStyleLine
}

// MLineStyleLine is one element of an MLINESTYLE.
type MLineStyleLine struct {
	Offset  float64
	Color   CmColor
	LtIndex int16 // before R2018
	Ltype   Ref   // R2018+
}

func (b *MLineStyle) data(c *codec, o *Object) {
	c.TV(&b.Name)
	c.TV(&b.Desc)
	c.BS(&b.Flags)
	c.CMC(&b.FillColor)
	c.BD(&b.StartAngle)
	c.BD(&b.EndAngle)
	n := byte(len(b.Lines))
	c.RC(&n)
	b.Lines = Slice(c, b.Lines, int(n))
	for i := range b.Lines {
		l := &b.Lines[i]
		c.BD(&l.Offset)
		c.CMC(&l.Color)
		if !c.R2018 {
			c.BS(&l.LtIndex)
		}
	}
}

func (b *MLineStyle) handles(c *codec, o *Object) {
	if c.R2018 {
		for i := range b.Lines {
			c.H(&b.Lines[i].Ltype)
		}
	}
}
