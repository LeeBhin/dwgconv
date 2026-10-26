package dwg

func init() {
	classBodyFactory["MULTILEADER"] = func() Body { return &MLeader{} }
}

// MLeaderBreak is a break start/end point pair.
type MLeaderBreak struct{ Start, End [3]float64 }

// MLeaderLineBreak is the break info of one segment of a leader line.
type MLeaderLineBreak struct {
	SegmentIndex int32
	Pairs        []MLeaderBreak
}

// MLeaderLine is LEADER_LINE{ (§20.4.86).
type MLeaderLine struct {
	Points     [][3]float64 // 10
	Breaks     []MLeaderLineBreak
	Index      int32   // 91
	LeaderType int16   // 170, R2010+
	Color      CmColor // 92
	Ltype      Ref     // 340
	Lineweight int32   // 171
	ArrowSize  float64 // 40
	Arrow      Ref     // 341
	Override   int32   // 93
}

// MLeaderRoot is LEADER{ (§20.4.86).
type MLeaderRoot struct {
	ContentValid bool       // 290
	Unknown291   bool       // 291
	Connection   [3]float64 // 10
	Direction    [3]float64 // 11
	Breaks       []MLeaderBreak
	Index        int32   // 90
	LandingDist  float64 // 40
	Lines        []MLeaderLine
	AttachDir    int16 // 271, R2010+
}

// MLeaderContext is the embedded MLeaderAnnotContext (§20.4.86), without the
// AcDbObjectContextData common part (unverified assumption).
type MLeaderContext struct {
	Roots          []MLeaderRoot
	Scale          float64    // 40
	ContentBase    [3]float64 // 10
	TextHeight     float64    // 41
	ArrowSize      float64    // 140
	LandingGap     float64    // 145
	LeftAttach     int16      // 174
	RightAttach    int16      // 175
	TextAlign      int16      // 176
	AttachType     int16      // 177
	HasText        bool       // 290
	TextLabel      string     // 304
	TextNormal     [3]float64 // 11
	TextStyle      Ref        // 340
	TextLocation   [3]float64 // 12
	TextDirection  [3]float64 // 13
	TextRotation   float64    // 42
	BoundWidth     float64    // 43
	BoundHeight    float64    // 44
	LineSpFactor   float64    // 45
	LineSpStyle    int16      // 170
	TextColor      CmColor    // 90
	Alignment      int16      // 171
	FlowDir        int16      // 172
	BgColor        CmColor    // 91
	BgScale        float64    // 141
	BgTransp       int32      // 92
	BgFill         bool       // 291
	BgMask         bool       // 292
	ColumnType     int16      // 173
	TextHeightAuto bool       // 293
	ColumnWidth    float64    // 142
	ColumnGutter   float64    // 143
	ColumnFlowRev  bool       // 294
	ColumnSizes    []float64  // 144
	WordBreak      bool       // 295
	Unknown        bool
	HasBlock       bool        // 296
	Block          Ref         // 341
	BlockNormal    [3]float64  // 14
	BlockLocation  [3]float64  // 15
	BlockScale     [3]float64  // 16
	BlockRotation  float64     // 46
	BlockColor     CmColor     // 93
	Matrix         [16]float64 // 47
	BasePoint      [3]float64  // 110
	BaseDir        [3]float64  // 111
	BaseVert       [3]float64  // 112
	NormalRev      bool        // 297
	TopAttach      int16       // 273, R2010+
	BottomAttach   int16       // 272, R2010+
}

func (x *MLeaderContext) code(c *codec) {
	n := c.BLCount(len(x.Roots))
	x.Roots = Slice(c, x.Roots, n)
	for i := range x.Roots {
		r := &x.Roots[i]
		c.B(&r.ContentValid)
		c.B(&r.Unknown291)
		c.BD3(&r.Connection)
		c.BD3(&r.Direction)
		nb := c.BLCount(len(r.Breaks))
		r.Breaks = Slice(c, r.Breaks, nb)
		for j := range r.Breaks {
			c.BD3(&r.Breaks[j].Start)
			c.BD3(&r.Breaks[j].End)
		}
		c.BL(&r.Index)
		c.BD(&r.LandingDist)
		nl := c.BLCount(len(r.Lines))
		r.Lines = Slice(c, r.Lines, nl)
		for j := range r.Lines {
			l := &r.Lines[j]
			np := c.BLCount(len(l.Points))
			l.Points = Slice(c, l.Points, np)
			for k := range l.Points {
				c.BD3(&l.Points[k])
			}
			nbi := c.BLCount(len(l.Breaks))
			l.Breaks = Slice(c, l.Breaks, nbi)
			for k := range l.Breaks {
				lb := &l.Breaks[k]
				c.BL(&lb.SegmentIndex)
				npair := c.BLCount(len(lb.Pairs))
				lb.Pairs = Slice(c, lb.Pairs, npair)
				for m := range lb.Pairs {
					c.BD3(&lb.Pairs[m].Start)
					c.BD3(&lb.Pairs[m].End)
				}
			}
			c.BL(&l.Index)
			if c.R2010 {
				c.BS(&l.LeaderType)
				c.CMC(&l.Color)
				c.BL(&l.Lineweight)
				c.BD(&l.ArrowSize)
				c.BL(&l.Override)
			}
		}
		if c.R2010 {
			c.BS(&r.AttachDir)
		}
	}
	c.BD(&x.Scale)
	c.BD3(&x.ContentBase)
	c.BD(&x.TextHeight)
	c.BD(&x.ArrowSize)
	c.BD(&x.LandingGap)
	c.BS(&x.LeftAttach)
	c.BS(&x.RightAttach)
	c.BS(&x.TextAlign)
	c.BS(&x.AttachType)
	c.B(&x.HasText)
	if x.HasText {
		c.TV(&x.TextLabel)
		c.BD3(&x.TextNormal)
		c.BD3(&x.TextLocation)
		c.BD3(&x.TextDirection)
		c.BD(&x.TextRotation)
		c.BD(&x.BoundWidth)
		c.BD(&x.BoundHeight)
		c.BD(&x.LineSpFactor)
		c.BS(&x.LineSpStyle)
		c.CMC(&x.TextColor)
		c.BS(&x.Alignment)
		c.BS(&x.FlowDir)
		c.CMC(&x.BgColor)
		c.BD(&x.BgScale)
		c.BL(&x.BgTransp)
		c.B(&x.BgFill)
		c.B(&x.BgMask)
		c.BS(&x.ColumnType)
		c.B(&x.TextHeightAuto)
		c.BD(&x.ColumnWidth)
		c.BD(&x.ColumnGutter)
		c.B(&x.ColumnFlowRev)
		ns := c.BLCount(len(x.ColumnSizes))
		x.ColumnSizes = Slice(c, x.ColumnSizes, ns)
		for i := range x.ColumnSizes {
			c.BD(&x.ColumnSizes[i])
		}
		c.B(&x.WordBreak)
		c.B(&x.Unknown)
	} else {
		c.B(&x.HasBlock)
		if x.HasBlock {
			c.BD3(&x.BlockNormal)
			c.BD3(&x.BlockLocation)
			c.BD3(&x.BlockScale)
			c.BD(&x.BlockRotation)
			c.CMC(&x.BlockColor)
			for i := range x.Matrix {
				c.BD(&x.Matrix[i])
			}
		}
	}
	c.BD3(&x.BasePoint)
	c.BD3(&x.BaseDir)
	c.BD3(&x.BaseVert)
	c.B(&x.NormalRev)
	if c.R2010 {
		c.BS(&x.TopAttach)
		c.BS(&x.BottomAttach)
	}
}

// MLeaderArrow is an arrow head override.
type MLeaderArrow struct {
	IsDefault bool // 94
	Arrow     Ref  // 345
}

// MLeaderLabel is a block attribute label.
type MLeaderLabel struct {
	AttDef  Ref     // 330
	Text    string  // 302
	UIIndex int16   // 177
	Width   float64 // 44
}

// MLeader is MULTILEADER (§20.4.48). Unverified: no samples contain these.
type MLeader struct {
	Version       int16 // 270, R2010+
	Ctx           MLeaderContext
	Style         Ref        // 340
	Override      int32      // 90
	LeaderType    int16      // 170
	LeaderColor   CmColor    // 91
	LeaderLtype   Ref        // 341
	Lineweight    int32      // 171
	Landing       bool       // 290
	DogLeg        bool       // 291
	LandingDist   float64    // 41
	ArrowHead     Ref        // 342
	ArrowSize     float64    // 42
	ContentType   int16      // 172
	TextStyle     Ref        // 343
	LeftAttach    int16      // 173
	RightAttach   int16      // 95
	TextAngle     int16      // 174
	Unknown175    int16      // 175, until R2007
	TextColor     CmColor    // 92
	TextFrame     bool       // 292
	BlockStyle    Ref        // 344
	BlockColor    CmColor    // 93
	BlockScale    [3]float64 // 10
	BlockRotation float64    // 43
	AttachType    int16      // 176
	Annotative    bool       // 293
	Arrows        []MLeaderArrow
	Labels        []MLeaderLabel
	TextDirNeg    bool    // 294
	IPEAlign      int16   // 178
	Justification int16   // 179
	ScaleFactor   float64 // 45
	AttachDir     int16   // 271, R2010+
	TopAttach     int16   // 273, R2010+
	BottomAttach  int16   // 272, R2010+
	ExtendToText  bool    // 295, R2013+
}

func (b *MLeader) data(c *codec, o *Object) {
	if c.R2010 {
		c.BS(&b.Version)
	}
	b.Ctx.code(c)
	c.BL(&b.Override)
	c.BS(&b.LeaderType)
	c.CMC(&b.LeaderColor)
	c.BL(&b.Lineweight)
	c.B(&b.Landing)
	c.B(&b.DogLeg)
	c.BD(&b.LandingDist)
	c.BD(&b.ArrowSize)
	c.BS(&b.ContentType)
	c.BS(&b.LeftAttach)
	c.BS(&b.RightAttach)
	c.BS(&b.TextAngle)
	if !c.R2010 {
		c.BS(&b.Unknown175)
	}
	c.CMC(&b.TextColor)
	c.B(&b.TextFrame)
	c.CMC(&b.BlockColor)
	c.BD3(&b.BlockScale)
	c.BD(&b.BlockRotation)
	c.BS(&b.AttachType)
	c.B(&b.Annotative)
	na := c.BLCount(len(b.Arrows))
	b.Arrows = Slice(c, b.Arrows, na)
	for i := range b.Arrows {
		c.B(&b.Arrows[i].IsDefault)
	}
	nl := c.BLCount(len(b.Labels))
	b.Labels = Slice(c, b.Labels, nl)
	for i := range b.Labels {
		l := &b.Labels[i]
		c.TV(&l.Text)
		c.BS(&l.UIIndex)
		c.BD(&l.Width)
	}
	c.B(&b.TextDirNeg)
	c.BS(&b.IPEAlign)
	c.BS(&b.Justification)
	c.BD(&b.ScaleFactor)
	if c.R2010 {
		c.BS(&b.AttachDir)
		c.BS(&b.TopAttach)
		c.BS(&b.BottomAttach)
	}
	if c.R2013 {
		c.B(&b.ExtendToText)
	}
}

// handles codes every handle of the entity in stream order, following the
// structure already established by data().
func (b *MLeader) handles(c *codec, o *Object) {
	x := &b.Ctx
	for i := range x.Roots {
		for j := range x.Roots[i].Lines {
			if c.R2010 {
				c.H(&x.Roots[i].Lines[j].Ltype)
				c.H(&x.Roots[i].Lines[j].Arrow)
			}
		}
	}
	if x.HasText {
		c.H(&x.TextStyle)
	} else if x.HasBlock {
		c.H(&x.Block)
	}
	c.H(&b.Style)
	c.H(&b.LeaderLtype)
	c.H(&b.ArrowHead)
	c.H(&b.TextStyle)
	c.H(&b.BlockStyle)
	for i := range b.Arrows {
		c.H(&b.Arrows[i].Arrow)
	}
	for i := range b.Labels {
		c.H(&b.Labels[i].AttDef)
	}
}
