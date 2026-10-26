package dwg

// Group B: simple geometric entities (§20.4.18-20.4.43).

func init() {
	bodyFactory[0x11] = func() Body { return &Arc{} }
	bodyFactory[0x12] = func() Body { return &Circle{} }
	bodyFactory[0x1B] = func() Body { return &Point{} }
	bodyFactory[0x1C] = func() Body { return &Face3D{} }
	bodyFactory[0x1F] = func() Body { return &Solid{} }
	bodyFactory[0x20] = func() Body { return &Solid{} } // TRACE has the SOLID layout (§20.4.36)
	bodyFactory[0x23] = func() Body { return &Ellipse{} }
	bodyFactory[0x24] = func() Body { return &Spline{} }
	bodyFactory[0x28] = func() Body { return &Ray{} }
	bodyFactory[0x29] = func() Body { return &Ray{} } // XLINE has the RAY layout (§20.4.43)
	bodyFactory[0x2F] = func() Body { return &MLine{} }
}

// bNoHandles is embedded by bodies without type-specific handles.
type bNoHandles struct{}

func (bNoHandles) handles(c *codec, o *Object) {}

// Arc §20.4.18.
type Arc struct {
	bNoHandles
	Center     [3]float64
	Radius     float64
	Thickness  float64
	Extrusion  [3]float64
	StartAngle float64
	EndAngle   float64
}

func (b *Arc) data(c *codec, o *Object) {
	c.BD3(&b.Center)
	c.BD(&b.Radius)
	c.BT(&b.Thickness)
	c.BE(&b.Extrusion)
	c.BD(&b.StartAngle)
	c.BD(&b.EndAngle)
}

// Circle §20.4.20.
type Circle struct {
	bNoHandles
	Center    [3]float64
	Radius    float64
	Thickness float64
	Extrusion [3]float64
}

func (b *Circle) data(c *codec, o *Object) {
	c.BD3(&b.Center)
	c.BD(&b.Radius)
	c.BT(&b.Thickness)
	c.BE(&b.Extrusion)
}

// Point §20.4.31.
type Point struct {
	bNoHandles
	Position  [3]float64
	Thickness float64
	Extrusion [3]float64
	XAxisAng  float64
}

func (b *Point) data(c *codec, o *Object) {
	c.BD3(&b.Position)
	c.BT(&b.Thickness)
	c.BE(&b.Extrusion)
	c.BD(&b.XAxisAng)
}

// Face3D is 3DFACE §20.4.32 (R2000+ form).
type Face3D struct {
	bNoHandles
	Corners    [4][3]float64
	InvisFlags int16
}

func (b *Face3D) data(c *codec, o *Object) {
	noFlags := b.InvisFlags == 0
	zZero := b.Corners[0][2] == 0
	c.B(&noFlags)
	c.B(&zZero)
	c.RD(&b.Corners[0][0])
	c.RD(&b.Corners[0][1])
	if !zZero {
		c.RD(&b.Corners[0][2])
	}
	for i := 1; i < 4; i++ {
		for k := 0; k < 3; k++ {
			c.DD(&b.Corners[i][k], b.Corners[i-1][k])
		}
	}
	if !noFlags {
		c.BS(&b.InvisFlags)
	}
}

// Solid is SOLID §20.4.35 and TRACE §20.4.36.
type Solid struct {
	bNoHandles
	Thickness float64
	Elevation float64
	Corners   [4][2]float64
	Extrusion [3]float64
}

func (b *Solid) data(c *codec, o *Object) {
	c.BT(&b.Thickness)
	c.BD(&b.Elevation)
	for i := range b.Corners {
		c.RD2(&b.Corners[i])
	}
	c.BE(&b.Extrusion)
}

// Ellipse §20.4.39.
type Ellipse struct {
	bNoHandles
	Center     [3]float64
	MajorAxis  [3]float64
	Extrusion  [3]float64
	AxisRatio  float64
	StartParam float64
	EndParam   float64
}

func (b *Ellipse) data(c *codec, o *Object) {
	c.BD3(&b.Center)
	c.BD3(&b.MajorAxis)
	c.BD3(&b.Extrusion)
	c.BD(&b.AxisRatio)
	c.BD(&b.StartParam)
	c.BD(&b.EndParam)
}

// Ray is RAY §20.4.42 and XLINE §20.4.43.
type Ray struct {
	bNoHandles
	Point  [3]float64
	Vector [3]float64
}

func (b *Ray) data(c *codec, o *Object) {
	c.BD3(&b.Point)
	c.BD3(&b.Vector)
}

// Spline §20.4.40.
type Spline struct {
	bNoHandles
	Scenario   int32
	Flags1     int32 // R2013+
	KnotParam  int32 // R2013+
	Degree     int32
	FitTol     float64
	BegTangent [3]float64
	EndTangent [3]float64
	Rational   bool
	Closed     bool
	Periodic   bool
	KnotTol    float64
	CtrlTol    float64
	Weighted   bool
	Knots      []float64
	CtrlPts    [][3]float64
	Weights    []float64 // present when Weighted
	FitPts     [][3]float64
}

func (b *Spline) data(c *codec, o *Object) {
	c.BL(&b.Scenario)
	if c.R2013 {
		c.BL(&b.Flags1)
		c.BL(&b.KnotParam)
	}
	c.BL(&b.Degree)
	var nFit, nKnots, nCtrl int
	// R2013+: a spline defined by fit points (Flags1&1) uses the fit layout even though
	// the stored scenario is 1; the control point block is then absent (observed, spec differs).
	hasFit := b.Scenario == 2 || (c.R2013 && b.Flags1&1 != 0)
	if hasFit {
		c.BD(&b.FitTol)
		c.BD3(&b.BegTangent)
		c.BD3(&b.EndTangent)
		nFit = c.BLCount(len(b.FitPts))
	}
	if !hasFit {
		c.B(&b.Rational)
		c.B(&b.Closed)
		c.B(&b.Periodic)
		c.BD(&b.KnotTol)
		c.BD(&b.CtrlTol)
		nKnots = c.BLCount(len(b.Knots))
		nCtrl = c.BLCount(len(b.CtrlPts))
		if c.w {
			b.Weighted = len(b.Weights) > 0
		}
		c.B(&b.Weighted)
	}
	b.Knots = Slice(c, b.Knots, nKnots)
	for i := range b.Knots {
		c.BD(&b.Knots[i])
	}
	b.CtrlPts = Slice(c, b.CtrlPts, nCtrl)
	if b.Weighted {
		b.Weights = Slice(c, b.Weights, nCtrl)
	}
	for i := range b.CtrlPts {
		c.BD3(&b.CtrlPts[i])
		if b.Weighted {
			c.BD(&b.Weights[i])
		}
	}
	b.FitPts = Slice(c, b.FitPts, nFit)
	for i := range b.FitPts {
		c.BD3(&b.FitPts[i])
	}
}

// MLine §20.4.50.
type MLine struct {
	Scale      float64
	Just       byte
	BasePoint  [3]float64
	Extrusion  [3]float64
	OpenClosed int16
	LinesInSty byte
	Verts      []MLineVertex
	Style      Ref
}

// MLineVertex is one MLINE vertex with per-style-line parameters.
type MLineVertex struct {
	Point    [3]float64
	VertDir  [3]float64
	MiterDir [3]float64
	Lines    []MLineSeg
}

// MLineSeg holds the segment and area fill parameters of one style line.
type MLineSeg struct {
	SegParms  []float64
	FillParms []float64
}

func (b *MLine) data(c *codec, o *Object) {
	c.BD(&b.Scale)
	c.RC(&b.Just)
	c.BD3(&b.BasePoint)
	c.BD3(&b.Extrusion)
	c.BS(&b.OpenClosed)
	c.RC(&b.LinesInSty)
	n := c.BSCount(len(b.Verts))
	b.Verts = Slice(c, b.Verts, n)
	for i := range b.Verts {
		v := &b.Verts[i]
		c.BD3(&v.Point)
		c.BD3(&v.VertDir)
		c.BD3(&v.MiterDir)
		v.Lines = Slice(c, v.Lines, int(b.LinesInSty))
		for k := range v.Lines {
			l := &v.Lines[k]
			l.SegParms = Slice(c, l.SegParms, c.BSCount(len(l.SegParms)))
			for j := range l.SegParms {
				c.BD(&l.SegParms[j])
			}
			l.FillParms = Slice(c, l.FillParms, c.BSCount(len(l.FillParms)))
			for j := range l.FillParms {
				c.BD(&l.FillParms[j])
			}
		}
	}
}

func (b *MLine) handles(c *codec, o *Object) { c.H(&b.Style) }
