package dwg

// Layouts of classes the specification does not document, inferred by pairing the object
// bits with ODA's typed DXF records (AC1018/AC1027/AC1032 samples). Field names follow the
// DXF meaning; where two fields carry equal sample values their order is a best guess (noted).

func init() {
	classBodyFactory["ACDBSECTIONVIEWSTYLE"] = func() Body { return &SectionViewStyle{} }
	classBodyFactory["ACDBDETAILVIEWSTYLE"] = func() Body { return &DetailViewStyle{} }
	classBodyFactory["RAPIDRTRENDERSETTINGS"] = func() Body { return &RapidRTRenderSettings{} }
	classBodyFactory["MENTALRAYRENDERSETTINGS"] = func() Body { return &MentalRayRenderSettings{} }
	classBodyFactory["PDFDEFINITION"] = func() Body { return &PdfDefinition{} }
}

// ModelDocViewStyle is the AcDbModelDocViewStyle part. Before R2018 the object stores one
// string (Name); R2018 adds Description and a BS (written as AcDbSectionViewStyle/
// AcDbDetailViewStyle group 70 — its owner subclass cannot be told from the samples).
type ModelDocViewStyle struct {
	Flags       int16  // 70
	Name        string // 3
	Modified    bool   // 290
	Description string // 300, R2018+
	Flags90     int32  // 90
	Version70   int16  // subclass 70, R2018+
}

func (m *ModelDocViewStyle) code(c *codec) {
	c.BS(&m.Flags)
	c.TV(&m.Name)
	c.B(&m.Modified)
	if c.R2018 {
		c.TV(&m.Description)
	}
	c.BL(&m.Flags90)
	if c.R2018 {
		c.BS(&m.Version70)
	}
}

// SectionViewStyle is ACDBSECTIONVIEWSTYLE (AcDbSectionViewStyle). The two 0-valued BLs and
// the two 2.5 BDs near the end are assigned by the DXF order (all samples hold defaults).
type SectionViewStyle struct {
	ModelDocViewStyle
	Flags           int32   // 90
	IDStyle         Ref     // 340
	IDColor         CmColor // 62
	IDHeight        float64 // 40
	ArrowStart      Ref     // 340
	ArrowEnd        Ref     // 340
	ArrowColor      CmColor // 62
	ArrowSize       float64 // 40
	ExcludeChars    string  // 300
	ArrowExtLen     float64 // 40
	IDPosition      int32   // 90
	IDOffset        float64 // 40
	ArrowPosition   int32   // 90
	PlaneLtype      Ref     // 340
	PlaneLineweight int32   // 90
	PlaneColor      CmColor // 62
	BendLtype       Ref     // 340
	BendLineweight  int32   // 90
	BendColor       CmColor // 62
	BendLength      float64 // 40
	EndOvershoot    float64 // 40
	EndLength       float64 // 40
	LabelStyle      Ref     // 340
	LabelColor      CmColor // 62
	LabelHeight     float64 // 40
	LabelAttach     int32   // 90
	LabelOffset     float64 // 40
	LabelAlign      int32   // 90
	LabelPattern    string  // 300
	HatchColor      CmColor // 62
	HatchBgColor    CmColor // 62
	HatchPattern    string  // 300
	HatchScale      float64 // 40
	HatchTransp     int32   // 90
	ShowHatch       bool    // 290
	Unknown290      bool    // 290
	HatchAngles     []float64
}

func (b *SectionViewStyle) data(c *codec, o *Object) {
	b.ModelDocViewStyle.code(c)
	c.BL(&b.Flags)
	c.CMC(&b.IDColor)
	c.BD(&b.IDHeight)
	c.CMC(&b.ArrowColor)
	c.BD(&b.ArrowSize)
	c.TV(&b.ExcludeChars)
	c.BD(&b.ArrowExtLen)
	c.BL(&b.PlaneLineweight)
	c.CMC(&b.PlaneColor)
	c.BL(&b.BendLineweight)
	c.CMC(&b.BendColor)
	c.BD(&b.BendLength)
	c.BD(&b.EndLength)
	c.CMC(&b.LabelColor)
	c.BD(&b.LabelHeight)
	c.BL(&b.LabelAttach)
	c.BD(&b.LabelOffset)
	c.BL(&b.LabelAlign)
	c.TV(&b.LabelPattern)
	c.CMC(&b.HatchColor)
	c.CMC(&b.HatchBgColor)
	c.TV(&b.HatchPattern)
	c.BD(&b.HatchScale)
	c.BL(&b.HatchTransp)
	c.B(&b.ShowHatch)
	c.B(&b.Unknown290)
	c.BL(&b.IDPosition)
	c.BD(&b.IDOffset)
	c.BL(&b.ArrowPosition)
	c.BD(&b.EndOvershoot)
	n := c.BLCount(len(b.HatchAngles))
	b.HatchAngles = Slice(c, b.HatchAngles, n)
	for i := range b.HatchAngles {
		c.BD(&b.HatchAngles[i])
	}
}

func (b *SectionViewStyle) handles(c *codec, o *Object) {
	c.H(&b.IDStyle)
	c.H(&b.ArrowStart)
	c.H(&b.ArrowEnd)
	c.H(&b.PlaneLtype)
	c.H(&b.BendLtype)
	c.H(&b.LabelStyle)
}

// DetailViewStyle is ACDBDETAILVIEWSTYLE (AcDbDetailViewStyle).
type DetailViewStyle struct {
	ModelDocViewStyle
	Flags              int32   // 90
	IDStyle            Ref     // 340
	IDColor            CmColor // 62
	IDHeight           float64 // 40
	ExcludeChars       string  // 300
	ArrowExtLen        float64 // 40 (0 in all samples)
	IDPlacement        byte    // 280
	ArrowSymbol        Ref     // 340
	ArrowColor         CmColor // 62
	ArrowSize          float64 // 40
	BoundaryLtype      Ref     // 340
	BoundaryLineweight int32   // 90
	BoundaryColor      CmColor // 62
	LabelStyle         Ref     // 340
	LabelColor         CmColor // 62
	LabelHeight        float64 // 40
	LabelAttach        int32   // 90
	LabelOffset        float64 // 40
	LabelAlign         int32   // 90
	LabelPattern       string  // 300
	BorderLtype        Ref     // 340
	BorderLineweight   int32   // 90
	BorderColor        CmColor // 62
	ConnLtype          Ref     // 340
	ConnLineweight     int32   // 90
	ConnColor          CmColor // 62
	ShowArrows         byte    // 280
}

func (b *DetailViewStyle) data(c *codec, o *Object) {
	b.ModelDocViewStyle.code(c)
	c.BL(&b.Flags)
	c.CMC(&b.IDColor)
	c.BD(&b.IDHeight)
	c.TV(&b.ExcludeChars)
	c.BD(&b.ArrowExtLen)
	c.RC(&b.IDPlacement)
	c.CMC(&b.ArrowColor)
	c.BD(&b.ArrowSize)
	c.BL(&b.BoundaryLineweight)
	c.CMC(&b.BoundaryColor)
	c.CMC(&b.LabelColor)
	c.BD(&b.LabelHeight)
	c.BL(&b.LabelAttach)
	c.BD(&b.LabelOffset)
	c.BL(&b.LabelAlign)
	c.TV(&b.LabelPattern)
	c.BL(&b.BorderLineweight)
	c.CMC(&b.BorderColor)
	c.BL(&b.ConnLineweight)
	c.CMC(&b.ConnColor)
	c.RC(&b.ShowArrows)
}

// handles follow the DXF 340 order: identifier style, arrow symbol, boundary ltype,
// label style, then the two 71=4 line types (border, connection).
func (b *DetailViewStyle) handles(c *codec, o *Object) {
	c.H(&b.IDStyle)
	c.H(&b.ArrowSymbol)
	c.H(&b.BoundaryLtype)
	c.H(&b.LabelStyle)
	c.H(&b.BorderLtype)
	c.H(&b.ConnLtype)
}

// RenderSettings is the AcDbRenderSettings part.
type RenderSettings struct {
	ClassVersion  int32  // 90
	Name          string // 1
	FogEnabled    bool   // 290
	FogBackground bool   // 290
	BackFaces     bool   // 290
	EnvImage      bool   // 290
	EnvImageFile  string // 1
	Description   string // 1
	DisplayIndex  int32  // 90
}

func (r *RenderSettings) code(c *codec) {
	c.BL(&r.ClassVersion)
	c.TV(&r.Name)
	c.B(&r.FogEnabled)
	c.B(&r.FogBackground)
	c.B(&r.BackFaces)
	c.B(&r.EnvImage)
	c.TV(&r.EnvImageFile)
	c.TV(&r.Description)
	c.BL(&r.DisplayIndex)
}

// RapidRTRenderSettings is RAPIDRTRENDERSETTINGS (AcDbRapidRTRenderSettings).
type RapidRTRenderSettings struct {
	RenderSettings
	Version       int32   // 90
	RenderTarget  int16   // 70
	RenderLevel   int32   // 90
	RenderTime    int32   // 90
	LightingModel int16   // 70
	FilterType    int16   // 70
	FilterWidth   float64 // 40
	FilterHeight  float64 // 40
	Unknown290    bool    // 290
}

func (b *RapidRTRenderSettings) data(c *codec, o *Object) {
	b.RenderSettings.code(c)
	c.BL(&b.Version)
	c.BS(&b.RenderTarget)
	c.BL(&b.RenderLevel)
	c.BL(&b.RenderTime)
	c.BS(&b.LightingModel)
	c.BS(&b.FilterType)
	c.BD(&b.FilterWidth)
	c.BD(&b.FilterHeight)
	c.B(&b.Unknown290)
}

func (b *RapidRTRenderSettings) handles(c *codec, o *Object) {}

// MentalRayRenderSettings is MENTALRAYRENDERSETTINGS (AcDbMentalRayRenderSettings).
// Verified on AC1018 only; the fields keep DXF order and are named generically.
type MentalRayRenderSettings struct {
	RenderSettings
	Fields []MRField
}

// MRField is one value of the mental ray block; Kind is the DXF group code (90, 70, 40, 290, 1).
type MRField struct {
	Kind int
	I    int32
	F    float64
	B    bool
	S    string
}

// MentalRayKinds is the DXF group sequence of AcDbMentalRayRenderSettings (AC1018).
var MentalRayKinds = []int{90, 90, 90, 70, 40, 40, 40, 40, 40, 40, 70, 290, 290, 90, 90, 90, 290, 90, 290,
	40, 90, 90, 90, 90, 290, 90, 290, 290, 290, 40, 40, 40, 70, 70, 40, 70, 70, 290, 1, 90, 70, 90, 290, 40}

func (b *MentalRayRenderSettings) data(c *codec, o *Object) {
	b.RenderSettings.code(c)
	if !c.w {
		b.Fields = make([]MRField, len(MentalRayKinds))
	}
	for i, k := range MentalRayKinds {
		f := &b.Fields[i]
		f.Kind = k
		switch k {
		case 90:
			c.BL(&f.I)
		case 70:
			v := int16(f.I)
			c.BS(&v)
			f.I = int32(v)
		case 40:
			c.BD(&f.F)
		case 290:
			c.B(&f.B)
		case 1:
			c.TV(&f.S)
		}
	}
}

func (b *MentalRayRenderSettings) handles(c *codec, o *Object) {}

// PdfDefinition is PDFDEFINITION (AcDbUnderlayDefinition).
type PdfDefinition struct {
	File string // 1
	Page string // 2
}

func (b *PdfDefinition) data(c *codec, o *Object) {
	c.TV(&b.File)
	c.TV(&b.Page)
}

func (b *PdfDefinition) handles(c *codec, o *Object) {}
