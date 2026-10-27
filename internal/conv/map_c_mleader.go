package conv

import (
	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

// Unverified against ODA: no MULTILEADER or ACAD_TABLE in the samples. AutoCAD DXF reference layout.

func init() {
	register("MULTILEADER", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.MLeader) {
		m.Subclass("AcDbMLeader")
		v := int16(2)
		m.Int16(270, &v)
		cMLeaderContext(m, x, &b.Ctx)
		x.HandleRef(m, 340, &b.Style, 5)
		m.Int32(90, &b.Override)
		m.Int16(170, &b.LeaderType)
		cCMC(m, 91, &b.LeaderColor)
		x.HandleRefOpt(m, 341, &b.LeaderLtype, 5)
		m.Int32(171, &b.Lineweight)
		m.Bool(290, &b.Landing)
		m.Bool(291, &b.DogLeg)
		m.Float(41, &b.LandingDist)
		x.HandleRefOpt(m, 342, &b.ArrowHead, 5)
		m.Float(42, &b.ArrowSize)
		m.Int16(172, &b.ContentType)
		x.HandleRefOpt(m, 343, &b.TextStyle, 5)
		m.Int16(173, &b.LeftAttach)
		m.Int16(95, &b.RightAttach)
		m.Int16(174, &b.TextAngle)
		m.Int16(175, &b.Unknown175)
		cCMC(m, 92, &b.TextColor)
		m.Bool(292, &b.TextFrame)
		x.HandleRefOpt(m, 344, &b.BlockStyle, 5)
		cCMC(m, 93, &b.BlockColor)
		m.Point3(10, &b.BlockScale)
		m.Float(43, &b.BlockRotation)
		m.Int16(176, &b.AttachType)
		m.Bool(293, &b.Annotative)
		n := len(b.Arrows)
		if m.Reading() {
			n = 0
			for m.Has(94) {
				b.Arrows = append(b.Arrows, dwg.MLeaderArrow{})
				n++
				a := &b.Arrows[n-1]
				m.Bool(94, &a.IsDefault)
				x.HandleRef(m, 345, &a.Arrow, 5)
			}
		} else {
			for i := range b.Arrows {
				m.Bool(94, &b.Arrows[i].IsDefault)
				x.HandleRef(m, 345, &b.Arrows[i].Arrow, 5)
			}
		}
		if m.Reading() {
			for m.Has(330) {
				var l dwg.MLeaderLabel
				x.HandleRef(m, 330, &l.AttDef, 4)
				m.Int16(177, &l.UIIndex)
				m.Float(44, &l.Width)
				m.Str(302, &l.Text)
				b.Labels = append(b.Labels, l)
			}
		} else {
			for i := range b.Labels {
				l := &b.Labels[i]
				x.HandleRef(m, 330, &l.AttDef, 4)
				m.Int16(177, &l.UIIndex)
				m.Float(44, &l.Width)
				m.Str(302, &l.Text)
			}
		}
		m.Bool(294, &b.TextDirNeg)
		m.Int16(178, &b.IPEAlign)
		m.Int16(179, &b.Justification)
		m.Float(45, &b.ScaleFactor)
		if x.d.Version.R2010 {
			m.Int16(271, &b.AttachDir)
			m.Int16(273, &b.TopAttach)
			m.Int16(272, &b.BottomAttach)
		}
		if x.d.Version.R2013 {
			m.Bool(295, &b.ExtendToText)
		}
	})
}

// cMarker writes a control string group (CONTEXT_DATA{, LEADER{, }, ...) or skips past it.
func cMarker(m *dxf.Mapper, code int, s string) {
	v := s
	m.Str(code, &v)
}

func cMLeaderContext(m *dxf.Mapper, x *ctx, c *dwg.MLeaderContext) {
	cMarker(m, 300, "CONTEXT_DATA{")
	m.Float(40, &c.Scale)
	m.Point3(10, &c.ContentBase)
	m.Float(41, &c.TextHeight)
	m.Float(140, &c.ArrowSize)
	m.Float(145, &c.LandingGap)
	m.Int16(174, &c.LeftAttach)
	m.Int16(175, &c.RightAttach)
	m.Int16(176, &c.TextAlign)
	m.Int16(177, &c.AttachType)
	m.Bool(290, &c.HasText)
	if c.HasText {
		m.Str(304, &c.TextLabel)
		m.Point3(11, &c.TextNormal)
		x.HandleRef(m, 340, &c.TextStyle, 5)
		m.Point3(12, &c.TextLocation)
		m.Point3(13, &c.TextDirection)
		m.Float(42, &c.TextRotation)
		m.Float(43, &c.BoundWidth)
		m.Float(44, &c.BoundHeight)
		m.Float(45, &c.LineSpFactor)
		m.Int16(170, &c.LineSpStyle)
		cCMC(m, 90, &c.TextColor)
		m.Int16(171, &c.Alignment)
		m.Int16(172, &c.FlowDir)
		cCMC(m, 91, &c.BgColor)
		m.Float(141, &c.BgScale)
		m.Int32(92, &c.BgTransp)
		m.Bool(291, &c.BgFill)
		m.Bool(292, &c.BgMask)
		m.Int16(173, &c.ColumnType)
		m.Bool(293, &c.TextHeightAuto)
		m.Float(142, &c.ColumnWidth)
		m.Float(143, &c.ColumnGutter)
		m.Bool(294, &c.ColumnFlowRev)
		cFloatRun(m, 144, &c.ColumnSizes)
		m.Bool(295, &c.WordBreak)
	} else {
		m.Bool(296, &c.HasBlock)
		if c.HasBlock {
			x.HandleRef(m, 341, &c.Block, 4)
			m.Point3(14, &c.BlockNormal)
			m.Point3(15, &c.BlockLocation)
			m.Point3(16, &c.BlockScale)
			m.Float(46, &c.BlockRotation)
			cCMC(m, 93, &c.BlockColor)
			for i := range c.Matrix {
				m.Float(47, &c.Matrix[i])
			}
		}
	}
	m.Point3(110, &c.BasePoint)
	m.Point3(111, &c.BaseDir)
	m.Point3(112, &c.BaseVert)
	m.Bool(297, &c.NormalRev)
	if m.Reading() {
		c.Roots = make([]dwg.MLeaderRoot, cCountMarkers(m, 302, 301))
	}
	n := len(c.Roots)
	for i := 0; i < n; i++ {
		cMLeaderRoot(m, x, &c.Roots[i])
	}
	if x.d.Version.R2010 {
		m.Int16(272, &c.BottomAttach)
		m.Int16(273, &c.TopAttach)
	}
	cMarker(m, 301, "}")
}

func cMLeaderRoot(m *dxf.Mapper, x *ctx, r *dwg.MLeaderRoot) {
	cMarker(m, 302, "LEADER{")
	m.Bool(290, &r.ContentValid)
	m.Bool(291, &r.Unknown291)
	m.Point3(10, &r.Connection)
	m.Point3(11, &r.Direction)
	cBreaks(m, 12, 13, &r.Breaks)
	m.Int32(90, &r.Index)
	m.Float(40, &r.LandingDist)
	if m.Reading() {
		r.Lines = make([]dwg.MLeaderLine, cCountMarkers(m, 304, 303))
	}
	n := len(r.Lines)
	for i := 0; i < n; i++ {
		l := &r.Lines[i]
		cMarker(m, 304, "LEADER_LINE{")
		cPointRun(m, 10, &l.Points)
		if m.Reading() {
			for m.Has(90) && !m.Peek(91) {
				var lb dwg.MLeaderLineBreak
				m.Int32(90, &lb.SegmentIndex)
				cBreaks(m, 11, 12, &lb.Pairs)
				l.Breaks = append(l.Breaks, lb)
			}
		} else {
			for k := range l.Breaks {
				m.Int32(90, &l.Breaks[k].SegmentIndex)
				cBreaks(m, 11, 12, &l.Breaks[k].Pairs)
			}
		}
		m.Int32(91, &l.Index)
		if x.d.Version.R2010 {
			m.Int16(170, &l.LeaderType)
			cCMC(m, 92, &l.Color)
			cHandleNext(m, x, 340, &l.Ltype, 5)
			m.Int32(171, &l.Lineweight)
			m.Float(40, &l.ArrowSize)
			cHandleNext(m, x, 341, &l.Arrow, 5)
			m.Int32(93, &l.Override)
		}
		cMarker(m, 305, "}")
	}
	if x.d.Version.R2010 {
		m.Int16(271, &r.AttachDir)
	}
	cMarker(m, 303, "}")
}

func cBreaks(m *dxf.Mapper, start, end int, bs *[]dwg.MLeaderBreak) {
	if m.Reading() {
		for m.Peek(start) {
			var b dwg.MLeaderBreak
			m.Point3(start, &b.Start)
			m.Point3(end, &b.End)
			*bs = append(*bs, b)
		}
		return
	}
	for i := range *bs {
		m.Point3(start, &(*bs)[i].Start)
		m.Point3(end, &(*bs)[i].End)
	}
}

func cPointRun(m *dxf.Mapper, code int, ps *[][3]float64) {
	if m.Reading() {
		for m.Peek(code) {
			var p [3]float64
			m.Point3(code, &p)
			*ps = append(*ps, p)
		}
		return
	}
	for i := range *ps {
		m.Point3(code, &(*ps)[i])
	}
}

func cFloatRun(m *dxf.Mapper, code int, fs *[]float64) {
	if m.Reading() {
		for m.Peek(code) {
			var f float64
			m.Float(code, &f)
			*fs = append(*fs, f)
		}
		return
	}
	for i := range *fs {
		m.Float(code, &(*fs)[i])
	}
}

// cCountMarkers counts the open markers (code) ahead of the cursor, up to the closing group stop
// at the same nesting level.
func cCountMarkers(m *dxf.Mapper, code, stop int) int {
	n := 0
	for _, g := range m.Rest() {
		if g.Code == stop {
			break
		}
		if g.Code == code {
			n++
		}
	}
	return n
}

// cHandleNext is an optional handle inside a repeated structure: on read it binds only when
// it is the very next group, so a missing one does not consume a later group with the same code.
func cHandleNext(m *dxf.Mapper, x *ctx, code int, r *dwg.Ref, refCode byte) {
	if m.Reading() && !m.Peek(code) {
		return
	}
	x.HandleRefOpt(m, code, r, refCode)
}
