package dwg

// Symbol tables: control objects and table records (§20.4.51-20.4.70).

func init() {
	for _, t := range []int{0x30, 0x32, 0x34, 0x38, 0x3C, 0x3E, 0x40, 0x42, 0x44, 0x46} {
		t := t
		bodyFactory[t] = func() Body { return &TableControl{kind: t} }
	}
	bodyFactory[0x31] = func() Body { return &BlockHeader{} }
	bodyFactory[0x33] = func() Body { return &Layer{} }
	bodyFactory[0x35] = func() Body { return &Style{} }
	bodyFactory[0x39] = func() Body { return &Ltype{} }
	bodyFactory[0x3D] = func() Body { return &View{} }
	bodyFactory[0x3F] = func() Body { return &UCS{} }
	bodyFactory[0x41] = func() Body { return &VPort{} }
	bodyFactory[0x43] = func() Body { return &AppID{} }
	bodyFactory[0x45] = func() Body { return &DimStyle{} }
	bodyFactory[0x47] = func() Body { return &VXTableRecord{} }
}

// TableControl is a symbol table control object (§20.4.51, .53, .55, .57, .59, .61, .63, .65, .67, .69).
// Entries are the table records. The BLOCK control additionally owns *MODEL_SPACE and *PAPER_SPACE,
// the LTYPE control BYLAYER and BYBLOCK; those are kept in Special and are not counted in the entry count.
// The DIMSTYLE control has an extra RC-counted list of hard owner handles (Extra), not in the spec.
type TableControl struct {
	kind    int
	Entries []Ref
	Special []Ref
	Extra   []Ref
}

func (b *TableControl) specialCount() int {
	switch b.kind {
	case 0x30, 0x38:
		return 2
	}
	return 0
}

func (b *TableControl) data(c *codec, o *Object) {
	n := c.BLCount(len(b.Entries))
	b.Entries = Slice(c, b.Entries, n)
	if !c.w {
		b.Special = make([]Ref, b.specialCount())
	}
	if b.kind == 0x44 {
		m := byte(len(b.Extra))
		c.RC(&m)
		b.Extra = Slice(c, b.Extra, int(m))
	}
}

func (b *TableControl) handles(c *codec, o *Object) {
	for i := range b.Entries {
		c.H(&b.Entries[i])
	}
	for i := range b.Special {
		c.H(&b.Special[i])
	}
	for i := range b.Extra {
		c.H(&b.Extra[i])
	}
}

// dTableEntry holds the fields common to every table record.
type dTableEntry struct {
	Name      string // 2
	Flag64    bool   // 70 bit 64
	XrefIndex int16  // xrefindex+1
	Xdep      bool   // 70 bit 16
	XrefBlock Ref    // external reference block handle (hard pointer)
}

func (e *dTableEntry) dData(c *codec) {
	c.TV(&e.Name)
	c.B(&e.Flag64)
	if !c.R2007 { // absent in R2007+ (observed; the spec lists it as common)
		c.BS(&e.XrefIndex)
	}
	c.B(&e.Xdep)
}

// BlockHeader §20.4.52 (BLOCK_RECORD).
type BlockHeader struct {
	dTableEntry
	Anonymous    bool
	HasAttribs   bool
	IsXref       bool
	XrefOverlaid bool
	Loaded       bool
	BasePoint    [3]float64 // 10
	XrefPath     string     // 1
	InsertCounts []byte     // non-zero RCs, one per insert handle
	Description  string     // 4
	Preview      []byte     // 310
	InsertUnits  int16      // 70
	Explodable   bool       // 280
	Scaling      byte       // 281
	Block        Ref        // BLOCK entity
	Entities     []Ref      // owned entities
	EndBlk       Ref        // ENDBLK entity
	Inserts      []Ref
	Layout       Ref
}

func (b *BlockHeader) data(c *codec, o *Object) {
	b.dData(c)
	c.B(&b.Anonymous)
	c.B(&b.HasAttribs)
	c.B(&b.IsXref)
	c.B(&b.XrefOverlaid)
	c.B(&b.Loaded)
	if !b.IsXref && !b.XrefOverlaid { // xref blocks own no entities and have no count (observed)
		n := c.BLCount(len(b.Entities))
		b.Entities = Slice(c, b.Entities, n)
	}
	c.BD3(&b.BasePoint)
	c.TV(&b.XrefPath)
	if c.w {
		if len(b.InsertCounts) != len(b.Inserts) {
			b.InsertCounts = make([]byte, len(b.Inserts))
			for i := range b.InsertCounts {
				b.InsertCounts[i] = 1
			}
		}
		for _, v := range b.InsertCounts {
			c.wd.RC(v)
		}
		c.wd.RC(0)
	} else {
		b.InsertCounts = nil
		for {
			v := c.dat.RC()
			if v == 0 {
				break
			}
			b.InsertCounts = append(b.InsertCounts, v)
		}
		b.Inserts = make([]Ref, len(b.InsertCounts))
	}
	c.TV(&b.Description)
	np := c.BLCount(len(b.Preview))
	c.Raw(&b.Preview, np)
	if c.R2007 {
		c.BS(&b.InsertUnits)
		c.B(&b.Explodable)
		c.RC(&b.Scaling)
	}
}

func (b *BlockHeader) handles(c *codec, o *Object) {
	c.H(&b.XrefBlock) // NULL hard pointer; the spec marks it R13-R2000 only but R2004+ files have it too
	c.H(&b.Block)
	for i := range b.Entities {
		c.H(&b.Entities[i])
	}
	c.H(&b.EndBlk)
	for i := range b.Inserts {
		c.H(&b.Inserts[i])
	}
	c.H(&b.Layout)
}

// Layer §20.4.54.
type Layer struct {
	dTableEntry
	Flags     int16 // frozen 1, on 2 (inverted), frz new vp 4, locked 8, plot 16, lineweight 0x03E0
	Color     CmColor
	PlotStyle Ref // 390
	Material  Ref // 347
	Ltype     Ref // 6
	Unknown   Ref // R2013+, normally NULL
}

func (b *Layer) data(c *codec, o *Object) {
	b.dData(c)
	c.BS(&b.Flags)
	c.CMC(&b.Color)
}

func (b *Layer) handles(c *codec, o *Object) {
	c.H(&b.XrefBlock)
	c.H(&b.PlotStyle)
	if c.R2007 {
		c.H(&b.Material)
	}
	c.H(&b.Ltype)
	if c.R2013 {
		c.H(&b.Unknown)
	}
}

// Style §20.4.56 (text style, "shapefile").
type Style struct {
	dTableEntry
	Vertical    bool
	IsShapeFile bool
	FixedHeight float64 // 40
	WidthFactor float64 // 41
	Oblique     float64 // 50
	Generation  byte    // 71
	LastHeight  float64 // 42
	FontName    string  // 3
	BigFont     string  // 4
}

func (b *Style) data(c *codec, o *Object) {
	b.dData(c)
	c.B(&b.Vertical)
	c.B(&b.IsShapeFile)
	c.BD(&b.FixedHeight)
	c.BD(&b.WidthFactor)
	c.BD(&b.Oblique)
	c.RC(&b.Generation)
	c.BD(&b.LastHeight)
	c.TV(&b.FontName)
	c.TV(&b.BigFont)
}

func (b *Style) handles(c *codec, o *Object) { c.H(&b.XrefBlock) }

// LtypeDash is one dash of a linetype pattern.
type LtypeDash struct {
	Length    float64    // 49
	ShapeCode int16      // 75
	Offset    [2]float64 // 44, 45
	Scale     float64    // 46
	Rotation  float64    // 50
	ShapeFlag int16      // 74
	Style     Ref        // 340
}

// Ltype §20.4.58.
type Ltype struct {
	dTableEntry
	Description   string  // 3
	PatternLength float64 // 40
	Alignment     byte    // 72, always 'A'
	Dashes        []LtypeDash
	StringArea    []byte // 256 bytes (R2004), 512 bytes when present (R2007+)
}

func (b *Ltype) hasStringArea(c *codec) bool {
	if !c.R2007 {
		return true
	}
	for _, d := range b.Dashes {
		if d.ShapeFlag&2 != 0 {
			return true
		}
	}
	return false
}

func (b *Ltype) data(c *codec, o *Object) {
	b.dData(c)
	c.TV(&b.Description)
	c.BD(&b.PatternLength)
	c.RC(&b.Alignment)
	n := byte(len(b.Dashes))
	c.RC(&n)
	b.Dashes = Slice(c, b.Dashes, int(n))
	for i := range b.Dashes {
		d := &b.Dashes[i]
		c.BD(&d.Length)
		c.BS(&d.ShapeCode)
		c.RD(&d.Offset[0])
		c.RD(&d.Offset[1])
		c.BD(&d.Scale)
		c.BD(&d.Rotation)
		c.BS(&d.ShapeFlag)
	}
	if b.hasStringArea(c) {
		size := 256
		if c.R2007 {
			size = 512
		}
		if c.w && len(b.StringArea) != size {
			a := make([]byte, size)
			copy(a, b.StringArea)
			b.StringArea = a
		}
		c.Raw(&b.StringArea, size)
	}
}

func (b *Ltype) handles(c *codec, o *Object) {
	c.H(&b.XrefBlock)
	for i := range b.Dashes {
		c.H(&b.Dashes[i].Style)
	}
}

// dViewMode codes the 4-bit view mode of VIEW and VPORT as stored: the first stream bit is the MSB
// (value bit 3). Stream order is 71's bit 0, bit 1, bit 2, then the inverse of 71's bit 4,
// so value = b0<<3 | b1<<2 | b2<<1 | !b4.
func dViewMode(c *codec, v *byte) {
	if c.w {
		c.wd.Bits(uint64(*v), 4)
	} else {
		*v = byte(c.dat.Bits(4))
	}
}

// dLighting holds the R2007+ default lighting fields of VIEW and VPORT.
type dLighting struct {
	UseDefaultLights bool    // 292
	LightingType     byte    // 282
	Brightness       float64 // 141
	Contrast         float64 // 142
	Ambient          CmColor // 63
}

func (l *dLighting) dLightData(c *codec) {
	c.B(&l.UseDefaultLights)
	c.RC(&l.LightingType)
	c.BD(&l.Brightness)
	c.BD(&l.Contrast)
	c.CMC(&l.Ambient)
}

// View §20.4.60.
type View struct {
	dTableEntry
	Height, Width float64    // 40, 41
	Center        [2]float64 // 10
	Target        [3]float64 // 12
	Direction     [3]float64 // 11
	Twist         float64    // 50
	LensLength    float64    // 42
	FrontClip     float64    // 43
	BackClip      float64    // 44
	ViewMode      byte       // 71 (4 bits)
	RenderMode    byte       // 281
	dLighting
	PaperSpace     bool       // 70 bit 1
	AssociatedUCS  bool       // 72
	UCSOrigin      [3]float64 // 110
	UCSXAxis       [3]float64 // 111
	UCSYAxis       [3]float64 // 112
	UCSElevation   float64    // 146
	OrthoType      int16      // 79
	CameraPlotable bool       // 73
	Background     Ref        // 332
	VisualStyle    Ref        // 348
	Sun            Ref        // 361
	BaseUCS        Ref        // 346
	NamedUCS       Ref        // 345
	LiveSection    Ref        // 334
}

func (b *View) data(c *codec, o *Object) {
	b.dData(c)
	c.BD(&b.Height)
	c.BD(&b.Width)
	c.RD2(&b.Center)
	c.BD3(&b.Target)
	c.BD3(&b.Direction)
	c.BD(&b.Twist)
	c.BD(&b.LensLength)
	c.BD(&b.FrontClip)
	c.BD(&b.BackClip)
	dViewMode(c, &b.ViewMode)
	c.RC(&b.RenderMode)
	if c.R2007 {
		b.dLighting.dLightData(c)
	}
	c.B(&b.PaperSpace)
	c.B(&b.AssociatedUCS)
	if b.AssociatedUCS {
		c.BD3(&b.UCSOrigin)
		c.BD3(&b.UCSXAxis)
		c.BD3(&b.UCSYAxis)
		c.BD(&b.UCSElevation)
		c.BS(&b.OrthoType)
	}
	if c.R2007 {
		c.B(&b.CameraPlotable)
	}
}

func (b *View) handles(c *codec, o *Object) {
	c.H(&b.XrefBlock)
	if c.R2007 {
		c.H(&b.Background)
		c.H(&b.VisualStyle)
		c.H(&b.Sun)
	}
	if b.AssociatedUCS {
		c.H(&b.BaseUCS)
		c.H(&b.NamedUCS)
	}
	if c.R2007 {
		c.H(&b.LiveSection)
	}
}

// UCS §20.4.62.
type UCS struct {
	dTableEntry
	Origin    [3]float64 // 10
	XAxis     [3]float64 // 11
	YAxis     [3]float64 // 12
	Elevation float64    // 146
	OrthoView int16      // 79
	OrthoType int16      // 71
	BaseUCS   Ref        // 346
	NamedUCS  Ref
}

func (b *UCS) data(c *codec, o *Object) {
	b.dData(c)
	c.BD3(&b.Origin)
	c.BD3(&b.XAxis)
	c.BD3(&b.YAxis)
	c.BD(&b.Elevation)
	c.BS(&b.OrthoView)
	c.BS(&b.OrthoType)
}

func (b *UCS) handles(c *codec, o *Object) {
	c.H(&b.XrefBlock)
	c.H(&b.BaseUCS)
	c.H(&b.NamedUCS)
}

// VPort §20.4.64.
type VPort struct {
	dTableEntry
	Height       float64    // 40
	AspectTimesH float64    // 41 (aspect ratio * view height)
	Center       [2]float64 // 12
	Target       [3]float64 // 17
	Direction    [3]float64 // 16
	Twist        float64    // 51
	LensLength   float64    // 42
	FrontClip    float64    // 43
	BackClip     float64    // 44
	ViewMode     byte       // 71 (4 bits)
	RenderMode   byte       // 281
	dLighting
	LowerLeft   [2]float64 // 10
	UpperRight  [2]float64 // 11
	UCSFollow   bool       // 71 bit 8
	CircleZoom  int16      // 72
	FastZoom    bool       // 73
	UCSIcon     byte       // 74 (2 bits)
	GridOn      bool       // 76
	GridSpacing [2]float64 // 15
	SnapOn      bool       // 75
	SnapStyle   bool       // 77
	SnapIsoPair int16      // 78
	SnapRot     float64    // 50
	SnapBase    [2]float64 // 13
	SnapSpacing [2]float64 // 14
	Unknown     bool
	UCSPerVP    bool       // 71
	UCSOrigin   [3]float64 // 110
	UCSXAxis    [3]float64 // 111
	UCSYAxis    [3]float64 // 112
	UCSElev     float64    // 146
	OrthoType   int16      // 79
	GridFlags   int16      // 60
	GridMajor   int16      // 61
	Background  Ref        // 332
	VisualStyle Ref        // 348
	Sun         Ref        // 361
	NamedUCS    Ref        // 345
	BaseUCS     Ref        // 346
}

func (b *VPort) data(c *codec, o *Object) {
	b.dData(c)
	c.BD(&b.Height)
	c.BD(&b.AspectTimesH)
	c.RD2(&b.Center)
	c.BD3(&b.Target)
	c.BD3(&b.Direction)
	c.BD(&b.Twist)
	c.BD(&b.LensLength)
	c.BD(&b.FrontClip)
	c.BD(&b.BackClip)
	dViewMode(c, &b.ViewMode)
	c.RC(&b.RenderMode)
	if c.R2007 {
		b.dLighting.dLightData(c)
	}
	c.RD2(&b.LowerLeft)
	c.RD2(&b.UpperRight)
	c.B(&b.UCSFollow)
	c.BS(&b.CircleZoom)
	c.B(&b.FastZoom)
	c.BB(&b.UCSIcon)
	c.B(&b.GridOn)
	c.RD2(&b.GridSpacing)
	c.B(&b.SnapOn)
	c.B(&b.SnapStyle)
	c.BS(&b.SnapIsoPair)
	c.BD(&b.SnapRot)
	c.RD2(&b.SnapBase)
	c.RD2(&b.SnapSpacing)
	c.B(&b.Unknown)
	c.B(&b.UCSPerVP)
	c.BD3(&b.UCSOrigin)
	c.BD3(&b.UCSXAxis)
	c.BD3(&b.UCSYAxis)
	c.BD(&b.UCSElev)
	c.BS(&b.OrthoType)
	if c.R2007 {
		c.BS(&b.GridFlags)
		c.BS(&b.GridMajor)
	}
}

func (b *VPort) handles(c *codec, o *Object) {
	c.H(&b.XrefBlock)
	if c.R2007 {
		c.H(&b.Background)
		c.H(&b.VisualStyle)
		c.H(&b.Sun)
	}
	c.H(&b.NamedUCS)
	c.H(&b.BaseUCS)
}

// AppID §20.4.66.
type AppID struct {
	dTableEntry
	Unknown byte // 71
}

func (b *AppID) data(c *codec, o *Object) {
	b.dData(c)
	c.RC(&b.Unknown)
}

func (b *AppID) handles(c *codec, o *Object) { c.H(&b.XrefBlock) }

// VXTableRecord §20.4.70 (viewport entity header).
type VXTableRecord struct {
	dTableEntry
	Flag1    bool // 70 bit 1
	Viewport Ref  // the viewport entity (soft owner)
	Prev     Ref  // previous header in chain (hard pointer)
}

func (b *VXTableRecord) data(c *codec, o *Object) {
	b.dData(c)
	c.B(&b.Flag1)
}

func (b *VXTableRecord) handles(c *codec, o *Object) {
	c.H(&b.XrefBlock)
	c.H(&b.Viewport)
	c.H(&b.Prev)
}
