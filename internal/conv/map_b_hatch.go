package conv

import (
	"fmt"
	"math"

	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

func init() {
	register("HATCH", 0x4E, hatchFields)
}

func hatchFields(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Hatch) {
	m.Subclass("AcDbHatch")
	elev := [3]float64{0, 0, b.Elevation}
	m.Point3(10, &elev)
	m.Point3(210, &b.Extrusion)
	m.Str(2, &b.Name)
	m.Bool(70, &b.SolidFill)
	// ODA writes 71=0 for an associative hatch without any boundary objects.
	assoc := b.Associative
	if !m.Reading() && assoc {
		assoc = false
		for _, p := range b.Paths {
			if len(p.Boundary) > 0 {
				assoc = true
			}
		}
	}
	m.Bool(71, &assoc)
	if m.Reading() {
		b.Associative = assoc
	}
	np := m.Count(91, len(b.Paths))
	if m.Reading() {
		b.Elevation = elev[2]
		b.Paths = make([]dwg.HatchPath, np)
	}
	anyPixel := false
	for i := range b.Paths {
		p := &b.Paths[i]
		m.Int32(92, &p.Flag)
		if p.Flag&4 != 0 {
			anyPixel = true
		}
		if p.Flag&2 != 0 {
			hasB := len(p.PolyBulge) > 0
			m.Bool(72, &hasB)
			m.Bool(73, &p.PolyClosed)
			n := m.Count(93, len(p.PolyPts))
			if m.Reading() {
				p.PolyBulges = hasB
				p.PolyPts = make([][2]float64, n)
				if hasB {
					p.PolyBulge = make([]float64, n)
				}
			}
			for k := range p.PolyPts {
				m.Point2(10, &p.PolyPts[k])
				if hasB {
					m.Float(42, &p.PolyBulge[k])
				}
			}
		} else {
			n := m.Count(93, len(p.Edges))
			if m.Reading() {
				p.Edges = make([]dwg.HatchEdge, n)
			}
			for k := range p.Edges {
				hatchEdge(m, &p.Edges[k])
			}
		}
		nb := m.Count(97, len(p.Boundary))
		if m.Reading() {
			p.Boundary = make([]dwg.Ref, nb)
		}
		for k := range p.Boundary {
			x.HandleRef(m, 330, &p.Boundary[k], 4)
		}
	}
	m.Int16(75, &b.Style)
	m.Int16(76, &b.PatternType)
	if !b.SolidFill {
		m.Angle(52, &b.Angle)
		m.Float(41, &b.ScaleSpacing)
		m.Bool(77, &b.Double)
		nl := m.Count(78, len(b.DefLines))
		if m.Reading() {
			b.DefLines = make([]dwg.HatchDefLine, nl)
		}
		for i := range b.DefLines {
			d := &b.DefLines[i]
			m.Angle(53, &d.Angle)
			m.Float(43, &d.Base[0])
			m.Float(44, &d.Base[1])
			m.Float(45, &d.Offset[0])
			m.Float(46, &d.Offset[1])
			nd := m.Count(79, len(d.Dashes))
			if m.Reading() {
				d.Dashes = make([]float64, nd)
			}
			for k := range d.Dashes {
				m.Float(49, &d.Dashes[k])
			}
		}
	}
	if anyPixel {
		m.Float(47, &b.PixelSize)
	}
	ns := m.Count(98, len(b.Seeds))
	if m.Reading() {
		b.Seeds = make([][2]float64, ns)
	}
	for i := range b.Seeds {
		m.Point2(10, &b.Seeds[i])
	}
	// ODA writes the gradient groups only for a gradient fill or a non-zero tint (observed);
	// stored gradient colors of a plain hatch are otherwise dropped.
	if (!m.Reading() && (b.IsGradient != 0 || b.GradTint != 0)) || (m.Reading() && m.Has(450)) {
		m.Int32(450, &b.IsGradient)
		m.Int32(451, &b.GradReserved)
		m.Float(460, &b.GradAngle) // radians in DXF as well
		m.Float(461, &b.GradShift)
		m.Int32(452, &b.GradSingle)
		m.Float(462, &b.GradTint)
		nc := m.Count(453, len(b.GradColors))
		if m.Reading() {
			b.GradColors = make([]dwg.HatchGradColor, nc)
		}
		for i := range b.GradColors {
			g := &b.GradColors[i]
			m.Float(463, &g.Value)
			// DWG keeps only the RGB (with a color method byte); the ACI is in the
			// object's GradientColorNACI xdata, which ODA copies to 63.
			aci := bGradientACI(x, o, i)
			m.Int16(63, &aci)
			rgb := g.RGB & 0xFFFFFF
			m.Int32(421, &rgb)
			if m.Reading() {
				g.RGB = int32(uint32(0xC2000000) | uint32(rgb))
			}
		}
		m.Str(470, &b.GradName)
	}
}

// bGradientACI returns the ACI stored in the "GradientColor<i+1>ACI" xdata of o (0 if absent).
func bGradientACI(x *ctx, o *dwg.Object, i int) int16 {
	want := fmt.Sprintf("GradientColor%dACI", i+1)
	for _, e := range o.EED {
		if x.names[e.App.H] != want {
			continue
		}
		gs, ok := eedToGroups(e.Data, x.d.Version.R2007, x.d.Text)
		if !ok {
			continue
		}
		for _, g := range gs {
			if g.Code == 1070 {
				return int16(g.Int())
			}
		}
	}
	return 0
}

// bEllipseGeo converts a parametric ellipse angle to the geometric angle, keeping whole turns.
func bEllipseGeo(t, ratio float64) float64 {
	if ratio == 1 || ratio == 0 {
		return t
	}
	return bNearTurn(math.Atan2(ratio*math.Sin(t), math.Cos(t)), t)
}

// bEllipseParam is the inverse of bEllipseGeo.
func bEllipseParam(g, ratio float64) float64 {
	if ratio == 1 || ratio == 0 {
		return g
	}
	return bNearTurn(math.Atan2(math.Sin(g)/ratio, math.Cos(g)), g)
}

// bNearTurn shifts a by whole turns so it is closest to ref.
func bNearTurn(a, ref float64) float64 {
	return a + 2*math.Pi*math.Round((ref-a)/(2*math.Pi))
}

func hatchEdge(m *dxf.Mapper, e *dwg.HatchEdge) {
	m.Byte(72, &e.Type)
	switch e.Type {
	case 1:
		m.Point2(10, &e.P0)
		m.Point2(11, &e.P1)
	case 2:
		m.Point2(10, &e.P0)
		m.Float(40, &e.Radius)
		m.Angle(50, &e.StartAngle)
		m.Angle(51, &e.EndAngle)
		m.Bool(73, &e.CCW)
	case 3:
		m.Point2(10, &e.P0)
		m.Point2(11, &e.P1)
		m.Float(40, &e.Radius)
		// DWG stores the parametric angle t, DXF the geometric angle atan2(r·sin t, cos t).
		sa, ea := bEllipseGeo(e.StartAngle, e.Radius), bEllipseGeo(e.EndAngle, e.Radius)
		m.Angle(50, &sa)
		m.Angle(51, &ea)
		m.Bool(73, &e.CCW)
		if m.Reading() {
			e.StartAngle, e.EndAngle = bEllipseParam(sa, e.Radius), bEllipseParam(ea, e.Radius)
		}
	case 4:
		m.Int32(94, &e.Degree)
		rational := len(e.Weights) > 0
		m.Bool(73, &rational)
		m.Bool(74, &e.Periodic)
		nk := m.Count(95, len(e.Knots))
		nc := m.Count(96, len(e.CtrlPts))
		if m.Reading() {
			e.Rational = rational
			e.Knots = make([]float64, nk)
			e.CtrlPts = make([][2]float64, nc)
			if rational {
				e.Weights = make([]float64, nc)
			}
		}
		for i := range e.Knots {
			m.Float(40, &e.Knots[i])
		}
		for i := range e.CtrlPts {
			m.Point2(10, &e.CtrlPts[i])
			if rational {
				m.Float(42, &e.Weights[i])
			}
		}
		nf := m.Count(97, len(e.FitPts))
		if m.Reading() {
			e.FitPts = make([][2]float64, nf)
		}
		for i := range e.FitPts {
			m.Point2(11, &e.FitPts[i])
		}
		if nf > 0 {
			m.Point2(12, &e.StartTan)
			m.Point2(13, &e.EndTan)
		}
	}
}
