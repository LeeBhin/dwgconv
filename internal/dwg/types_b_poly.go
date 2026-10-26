package dwg

// Group B: polylines, vertices, LWPOLYLINE and HATCH (§20.4.11-17, 20.4.33-34, 20.4.75, 20.4.85).

func init() {
	bodyFactory[0x0A] = func() Body { return &Vertex2D{} }
	bodyFactory[0x0B] = func() Body { return &Vertex3D{} }
	bodyFactory[0x0C] = func() Body { return &Vertex3D{} } // VERTEX (MESH) §20.4.13
	bodyFactory[0x0D] = func() Body { return &Vertex3D{} } // VERTEX (PFACE) §20.4.14
	bodyFactory[0x0E] = func() Body { return &VertexPFaceFace{} }
	bodyFactory[0x0F] = func() Body { return &Polyline2D{} }
	bodyFactory[0x10] = func() Body { return &Polyline3D{} }
	bodyFactory[0x1D] = func() Body { return &PolylinePFace{} }
	bodyFactory[0x1E] = func() Body { return &PolylineMesh{} }
	bodyFactory[0x4D] = func() Body { return &LWPolyline{} }
	classBodyFactory["LWPOLYLINE"] = func() Body { return &LWPolyline{} }
	bodyFactory[0x4E] = func() Body { return &Hatch{} }
	classBodyFactory["HATCH"] = func() Body { return &Hatch{} }
}

// Vertex2D §20.4.11.
type Vertex2D struct {
	bNoHandles
	Flags      byte
	Point      [3]float64
	StartWidth float64
	EndWidth   float64
	Bulge      float64
	VertexID   int32 // R2010+
	TangentDir float64
}

func (b *Vertex2D) data(c *codec, o *Object) {
	c.RC(&b.Flags)
	c.BD3(&b.Point)
	// A negative start width means start == end == |start| (§20.4.11).
	sw := b.StartWidth
	if c.w && b.StartWidth == b.EndWidth && b.StartWidth != 0 {
		sw = -b.StartWidth
	}
	c.BD(&sw)
	if sw < 0 {
		b.StartWidth, b.EndWidth = -sw, -sw
	} else {
		b.StartWidth = sw
		c.BD(&b.EndWidth)
	}
	c.BD(&b.Bulge)
	if c.R2010 {
		c.BL(&b.VertexID)
	}
	c.BD(&b.TangentDir)
}

// Vertex3D is VERTEX (3D), (MESH) and (PFACE) §20.4.12-14.
type Vertex3D struct {
	bNoHandles
	Flags byte
	Point [3]float64
}

func (b *Vertex3D) data(c *codec, o *Object) {
	c.RC(&b.Flags)
	c.BD3(&b.Point)
}

// VertexPFaceFace §20.4.15.
type VertexPFaceFace struct {
	bNoHandles
	Index [4]int16
}

func (b *VertexPFaceFace) data(c *codec, o *Object) {
	for i := range b.Index {
		c.BS(&b.Index[i])
	}
}

// bOwned codes the R2004+ owned vertex list and SEQEND of the old-style polylines.
type bOwned struct {
	Vertices []Ref
	SeqEnd   Ref
}

func (b *bOwned) count(c *codec) {
	n := c.BLCount(len(b.Vertices))
	b.Vertices = Slice(c, b.Vertices, n)
}

func (b *bOwned) handles(c *codec, o *Object) {
	for i := range b.Vertices {
		c.H(&b.Vertices[i])
	}
	c.H(&b.SeqEnd)
}

// Polyline2D §20.4.16.
type Polyline2D struct {
	bOwned
	Flags      int16
	CurveType  int16
	StartWidth float64
	EndWidth   float64
	Thickness  float64
	Elevation  float64
	Extrusion  [3]float64
}

func (b *Polyline2D) data(c *codec, o *Object) {
	c.BS(&b.Flags)
	c.BS(&b.CurveType)
	c.BD(&b.StartWidth)
	c.BD(&b.EndWidth)
	c.BT(&b.Thickness)
	c.BD(&b.Elevation)
	c.BE(&b.Extrusion)
	b.count(c)
}

// Polyline3D §20.4.17. SplineFlags/ClosedFlags are the two raw RC flag bytes.
type Polyline3D struct {
	bOwned
	SplineFlags byte
	ClosedFlags byte
}

func (b *Polyline3D) data(c *codec, o *Object) {
	c.RC(&b.SplineFlags)
	c.RC(&b.ClosedFlags)
	b.count(c)
}

// PolylinePFace §20.4.33.
type PolylinePFace struct {
	bOwned
	NumVerts int16
	NumFaces int16
}

func (b *PolylinePFace) data(c *codec, o *Object) {
	c.BS(&b.NumVerts)
	c.BS(&b.NumFaces)
	b.count(c)
}

// PolylineMesh §20.4.34.
type PolylineMesh struct {
	bOwned
	Flags     int16
	CurveType int16
	MCount    int16
	NCount    int16
	MDensity  int16
	NDensity  int16
}

func (b *PolylineMesh) data(c *codec, o *Object) {
	c.BS(&b.Flags)
	c.BS(&b.CurveType)
	c.BS(&b.MCount)
	c.BS(&b.NCount)
	c.BS(&b.MDensity)
	c.BS(&b.NDensity)
	b.count(c)
}

// LWPolyline §20.4.85. Flag bits 4/8/2/1/16/1024/32 mark optional data; on write
// they are derived from the fields (other bits, e.g. closed 512 and plinegen 256, are kept).
type LWPolyline struct {
	bNoHandles
	Flag       int16 `conv:"derived"` // closed/plinegen are carried by DXF 70, the rest is derived
	ConstWidth float64
	Elevation  float64
	Thickness  float64
	Normal     [3]float64
	Points     [][2]float64
	Bulges     []float64
	VertexIDs  []int32 // R2010+
	Widths     [][2]float64
}

func (b *LWPolyline) data(c *codec, o *Object) {
	if c.w {
		// Optional-data bits are set from the fields; bits already set are kept so
		// that an unchanged object re-encodes identically.
		set := func(cond bool, bit int16) {
			if cond {
				b.Flag |= bit
			}
		}
		set(b.ConstWidth != 0, 4)
		set(b.Elevation != 0, 8)
		set(b.Thickness != 0, 2)
		set(b.Normal != [3]float64{0, 0, 1} && b.Normal != [3]float64{}, 1)
		set(len(b.Bulges) > 0, 16)
		set(len(b.VertexIDs) > 0, 1024)
		set(len(b.Widths) > 0, 32)
	}
	c.BS(&b.Flag)
	f := b.Flag
	if f&4 != 0 {
		c.BD(&b.ConstWidth)
	}
	if f&8 != 0 {
		c.BD(&b.Elevation)
	}
	if f&2 != 0 {
		c.BD(&b.Thickness)
	}
	if f&1 != 0 {
		c.BD3(&b.Normal)
	} else if !c.w {
		b.Normal = [3]float64{0, 0, 1}
	}
	nPts := c.BLCount(len(b.Points))
	nBulge, nID, nWidth := 0, 0, 0
	if f&16 != 0 {
		nBulge = c.BLCount(len(b.Bulges))
	}
	if c.R2010 && f&1024 != 0 {
		nID = c.BLCount(len(b.VertexIDs))
	}
	if f&32 != 0 {
		nWidth = c.BLCount(len(b.Widths))
	}
	b.Points = Slice(c, b.Points, nPts)
	for i := range b.Points {
		if i == 0 {
			c.RD2(&b.Points[0])
		} else {
			c.DD2(&b.Points[i], b.Points[i-1])
		}
	}
	b.Bulges = Slice(c, b.Bulges, nBulge)
	for i := range b.Bulges {
		c.BD(&b.Bulges[i])
	}
	b.VertexIDs = Slice(c, b.VertexIDs, nID)
	for i := range b.VertexIDs {
		c.BL(&b.VertexIDs[i])
	}
	b.Widths = Slice(c, b.Widths, nWidth)
	for i := range b.Widths {
		c.BD2(&b.Widths[i])
	}
}

// Hatch §20.4.75.
type Hatch struct {
	IsGradient   int32
	GradReserved int32
	GradAngle    float64
	GradShift    float64
	GradSingle   int32
	GradTint     float64
	GradColors   []HatchGradColor
	GradName     string
	Elevation    float64
	Extrusion    [3]float64
	Name         string
	SolidFill    bool
	Associative  bool
	Paths        []HatchPath
	Style        int16
	PatternType  int16
	Angle        float64
	ScaleSpacing float64
	Double       bool
	DefLines     []HatchDefLine
	PixelSize    float64
	Seeds        [][2]float64
}

// HatchGradColor is one gradient color entry.
type HatchGradColor struct {
	Value  float64
	Unk    int16
	RGB    int32
	ColorB byte
}

// HatchPath is one boundary path. Polyline paths (Flag&2) use Poly*, others Edges.
type HatchPath struct {
	Flag       int32
	Edges      []HatchEdge
	PolyBulges bool
	PolyClosed bool
	PolyPts    [][2]float64
	PolyBulge  []float64
	Boundary   []Ref
}

// HatchEdge is one edge of a non-polyline path. Type 1 line, 2 arc, 3 ellipse arc, 4 spline.
type HatchEdge struct {
	Type       byte
	P0, P1     [2]float64 // line ends; arc/ellipse center (P0) and major axis end (P1)
	Radius     float64    // arc radius, ellipse minor/major ratio
	StartAngle float64
	EndAngle   float64
	CCW        bool
	Degree     int32
	Rational   bool
	Periodic   bool
	Knots      []float64
	CtrlPts    [][2]float64
	Weights    []float64
	FitPts     [][2]float64
	StartTan   [2]float64
	EndTan     [2]float64
}

// HatchDefLine is one pattern definition line.
type HatchDefLine struct {
	Angle  float64
	Base   [2]float64
	Offset [2]float64
	Dashes []float64
}

func (b *Hatch) data(c *codec, o *Object) {
	c.BL(&b.IsGradient)
	c.BL(&b.GradReserved)
	c.BD(&b.GradAngle)
	c.BD(&b.GradShift)
	c.BL(&b.GradSingle)
	c.BD(&b.GradTint)
	b.GradColors = Slice(c, b.GradColors, c.BLCount(len(b.GradColors)))
	for i := range b.GradColors {
		g := &b.GradColors[i]
		c.BD(&g.Value)
		c.BS(&g.Unk)
		c.BL(&g.RGB)
		c.RC(&g.ColorB)
	}
	c.TV(&b.GradName)
	c.BD(&b.Elevation)
	c.BD3(&b.Extrusion)
	c.TV(&b.Name)
	c.B(&b.SolidFill)
	c.B(&b.Associative)
	b.Paths = Slice(c, b.Paths, c.BLCount(len(b.Paths)))
	anyPixel := false
	for i := range b.Paths {
		p := &b.Paths[i]
		c.BL(&p.Flag)
		if p.Flag&4 != 0 {
			anyPixel = true
		}
		if p.Flag&2 == 0 {
			p.Edges = Slice(c, p.Edges, c.BLCount(len(p.Edges)))
			for k := range p.Edges {
				p.Edges[k].code(c)
			}
		} else {
			if c.w {
				p.PolyBulges = len(p.PolyBulge) > 0
			}
			c.B(&p.PolyBulges)
			c.B(&p.PolyClosed)
			n := c.BLCount(len(p.PolyPts))
			p.PolyPts = Slice(c, p.PolyPts, n)
			if p.PolyBulges {
				p.PolyBulge = Slice(c, p.PolyBulge, n)
			}
			for k := range p.PolyPts {
				c.RD2(&p.PolyPts[k])
				if p.PolyBulges {
					c.BD(&p.PolyBulge[k])
				}
			}
		}
		p.Boundary = Slice(c, p.Boundary, c.BLCount(len(p.Boundary)))
	}
	c.BS(&b.Style)
	c.BS(&b.PatternType)
	if !b.SolidFill {
		c.BD(&b.Angle)
		c.BD(&b.ScaleSpacing)
		c.B(&b.Double)
		b.DefLines = Slice(c, b.DefLines, c.BSCount(len(b.DefLines)))
		for i := range b.DefLines {
			d := &b.DefLines[i]
			c.BD(&d.Angle)
			c.BD2(&d.Base)
			c.BD2(&d.Offset)
			d.Dashes = Slice(c, d.Dashes, c.BSCount(len(d.Dashes)))
			for k := range d.Dashes {
				c.BD(&d.Dashes[k])
			}
		}
	}
	if anyPixel {
		c.BD(&b.PixelSize)
	}
	b.Seeds = Slice(c, b.Seeds, c.BLCount(len(b.Seeds)))
	for i := range b.Seeds {
		c.RD2(&b.Seeds[i])
	}
}

func (e *HatchEdge) code(c *codec) {
	c.RC(&e.Type)
	switch e.Type {
	case 1:
		c.RD2(&e.P0)
		c.RD2(&e.P1)
	case 2:
		c.RD2(&e.P0)
		c.BD(&e.Radius)
		c.BD(&e.StartAngle)
		c.BD(&e.EndAngle)
		c.B(&e.CCW)
	case 3:
		c.RD2(&e.P0)
		c.RD2(&e.P1)
		c.BD(&e.Radius)
		c.BD(&e.StartAngle)
		c.BD(&e.EndAngle)
		c.B(&e.CCW)
	case 4:
		c.BL(&e.Degree)
		if c.w {
			e.Rational = len(e.Weights) > 0
		}
		c.B(&e.Rational)
		c.B(&e.Periodic)
		nk := c.BLCount(len(e.Knots))
		nc := c.BLCount(len(e.CtrlPts))
		e.Knots = Slice(c, e.Knots, nk)
		for i := range e.Knots {
			c.BD(&e.Knots[i])
		}
		e.CtrlPts = Slice(c, e.CtrlPts, nc)
		if e.Rational {
			e.Weights = Slice(c, e.Weights, nc)
		}
		for i := range e.CtrlPts {
			c.RD2(&e.CtrlPts[i])
			if e.Rational {
				c.BD(&e.Weights[i])
			}
		}
		if c.R2010 {
			nf := c.BLCount(len(e.FitPts))
			e.FitPts = Slice(c, e.FitPts, nf)
			for i := range e.FitPts {
				c.RD2(&e.FitPts[i])
			}
			if nf > 0 {
				c.RD2(&e.StartTan)
				c.RD2(&e.EndTan)
			}
		}
	default:
		panic(errHatchEdge)
	}
}

func (b *Hatch) handles(c *codec, o *Object) {
	for i := range b.Paths {
		p := &b.Paths[i]
		for k := range p.Boundary {
			c.H(&p.Boundary[k])
		}
	}
}

var errHatchEdge = bError("hatch: unknown edge type")

type bError string

func (e bError) Error() string { return string(e) }
