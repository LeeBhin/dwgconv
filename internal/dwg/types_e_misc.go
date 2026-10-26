package dwg

func init() {
	classBodyFactory["MLEADERSTYLE"] = func() Body { return &MLeaderStyle{} }
	classBodyFactory["IMAGEDEF"] = func() Body { return &ImageDef{} }
	classBodyFactory["FIELDLIST"] = func() Body { return &FieldList{} }
}

// MLeaderStyle §20.4.87.
type MLeaderStyle struct {
	Version          int16 // R2010+
	ContentType      int16
	DrawMLeaderOrder int16
	DrawLeaderOrder  int16
	MaxPoints        int32
	FirstSegAngle    float64
	SecondSegAngle   float64
	LeaderType       int16
	LeaderColor      CmColor
	LeaderLtype      Ref
	LeaderLineweight int32
	LandingEnabled   bool
	LandingGap       float64
	DoglegEnabled    bool
	LandingDistance  float64
	Description      string
	ArrowHead        Ref
	ArrowHeadSize    float64
	TextDefault      string
	TextStyle        Ref
	LeftAttachment   int16
	RightAttachment  int16
	TextAngleType    int16 // new format only
	TextAlignType    int16
	TextColor        CmColor
	TextHeight       float64
	TextFrame        bool
	AlignTextLeft    bool // new format only
	AlignSpace       float64
	Block            Ref
	BlockColor       CmColor
	BlockScale       [3]float64
	BlockScaleOn     bool
	BlockRotation    float64
	BlockRotationOn  bool
	BlockConnection  int16
	ScaleFactor      float64
	PropertyChanged  bool
	Annotative       bool
	BreakSize        float64
	AttachDirection  int16 // R2010+
	TopAttachment    int16 // R2010+
	BottomAttachment int16 // R2010+
	UnknownBool      bool  // R2010+, undocumented trailing bit
	NewFormat        bool  // R2010+ or ACAD_MLEADERVER EED present (R2004)
}

func (b *MLeaderStyle) data(c *codec, o *Object) {
	if c.R2010 {
		b.NewFormat = true
		c.BS(&b.Version)
	} else if !c.w {
		// §20.4.87: pre-R2010 files use the new layout when ACAD_MLEADERVER EED is attached.
		b.NewFormat = len(o.EED) > 0
	}
	c.BS(&b.ContentType)
	c.BS(&b.DrawMLeaderOrder)
	c.BS(&b.DrawLeaderOrder)
	c.BL(&b.MaxPoints)
	c.BD(&b.FirstSegAngle)
	c.BD(&b.SecondSegAngle)
	c.BS(&b.LeaderType)
	c.CMC(&b.LeaderColor)
	c.BL(&b.LeaderLineweight)
	c.B(&b.LandingEnabled)
	c.BD(&b.LandingGap)
	c.B(&b.DoglegEnabled)
	c.BD(&b.LandingDistance)
	c.TV(&b.Description)
	c.BD(&b.ArrowHeadSize)
	c.TV(&b.TextDefault)
	c.BS(&b.LeftAttachment)
	c.BS(&b.RightAttachment)
	if b.NewFormat {
		c.BS(&b.TextAngleType)
	}
	c.BS(&b.TextAlignType)
	c.CMC(&b.TextColor)
	c.BD(&b.TextHeight)
	c.B(&b.TextFrame)
	if b.NewFormat {
		c.B(&b.AlignTextLeft)
	}
	c.BD(&b.AlignSpace)
	c.CMC(&b.BlockColor)
	c.BD3(&b.BlockScale)
	c.B(&b.BlockScaleOn)
	c.BD(&b.BlockRotation)
	c.B(&b.BlockRotationOn)
	c.BS(&b.BlockConnection)
	c.BD(&b.ScaleFactor)
	c.B(&b.PropertyChanged)
	c.B(&b.Annotative)
	c.BD(&b.BreakSize)
	if c.R2010 {
		c.BS(&b.AttachDirection)
		c.BS(&b.TopAttachment)
		c.BS(&b.BottomAttachment)
		// Not in the spec: R2013/R2018 files carry one more bit here (no AC1024 sample to confirm R2010).
		c.B(&b.UnknownBool)
	}
}

func (b *MLeaderStyle) handles(c *codec, o *Object) {
	c.H(&b.LeaderLtype)
	c.H(&b.ArrowHead)
	c.H(&b.TextStyle)
	c.H(&b.Block)
}

// ImageDef §20.4.81.
type ImageDef struct {
	ClassVersion int32
	ImageSize    [2]float64
	FilePath     string
	IsLoaded     bool
	ResUnits     byte
	PixelSize    [2]float64
}

func (b *ImageDef) data(c *codec, o *Object) {
	c.BL(&b.ClassVersion)
	c.RD2(&b.ImageSize)
	c.TV(&b.FilePath)
	c.B(&b.IsLoaded)
	c.RC(&b.ResUnits)
	c.RD2(&b.PixelSize)
}
func (b *ImageDef) handles(c *codec, o *Object) {}

// FieldList §20.4.77.
type FieldList struct {
	Unknown bool
	Fields  []Ref
}

func (b *FieldList) data(c *codec, o *Object) {
	n := c.BLCount(len(b.Fields))
	c.B(&b.Unknown)
	b.Fields = Slice(c, b.Fields, n)
}

func (b *FieldList) handles(c *codec, o *Object) {
	for i := range b.Fields {
		c.H(&b.Fields[i])
	}
}
