package conv

import (
	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

// Group B: simple geometric entities.

func init() {
	register("ARC", 0x11, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Arc) {
		m.Subclass("AcDbCircle")
		m.FloatDef(39, &b.Thickness, 0)
		m.Point3(10, &b.Center)
		m.Float(40, &b.Radius)
		m.Extrusion(&b.Extrusion)
		m.Subclass("AcDbArc")
		m.Angle(50, &b.StartAngle)
		m.Angle(51, &b.EndAngle)
	})
	register("CIRCLE", 0x12, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Circle) {
		m.Subclass("AcDbCircle")
		m.FloatDef(39, &b.Thickness, 0)
		m.Point3(10, &b.Center)
		m.Float(40, &b.Radius)
		m.Extrusion(&b.Extrusion)
	})
	register("POINT", 0x1B, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Point) {
		m.Subclass("AcDbPoint")
		m.Point3(10, &b.Position)
		m.FloatDef(39, &b.Thickness, 0)
		m.Extrusion(&b.Extrusion)
		m.AngleDef(50, &b.XAxisAng, 0)
	})
	solid := func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Solid) {
		m.Subclass("AcDbTrace")
		m.FloatDef(39, &b.Thickness, 0)
		for i := range b.Corners {
			p := [3]float64{b.Corners[i][0], b.Corners[i][1], b.Elevation}
			m.Point3(10+i, &p)
			if m.Reading() {
				b.Corners[i] = [2]float64{p[0], p[1]}
				b.Elevation = p[2]
			}
		}
		m.Extrusion(&b.Extrusion)
	}
	bAlias(register("SOLID", 0x1F, solid), "TRACE", 0x20)
	register("3DFACE", 0x1C, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Face3D) {
		m.Subclass("AcDbFace")
		for i := range b.Corners {
			m.Point3(10+i, &b.Corners[i])
		}
		m.Int16Def(70, &b.InvisFlags, 0)
	})
	register("ELLIPSE", 0x23, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Ellipse) {
		m.Subclass("AcDbEllipse")
		m.Point3(10, &b.Center)
		m.Point3(11, &b.MajorAxis)
		m.Point3(210, &b.Extrusion)
		m.Float(40, &b.AxisRatio)
		m.Float(41, &b.StartParam) // parameters stay in radians in DXF
		m.Float(42, &b.EndParam)
	})
	ray := func(sub string) func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Ray) {
		return func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Ray) {
			m.Subclass(sub)
			m.Point3(10, &b.Point)
			m.Point3(11, &b.Vector)
		}
	}
	xl := bAlias(register("RAY", 0x28, ray("AcDbRay")), "XLINE", 0x29)
	xl.fields = func(m *dxf.Mapper, x *ctx, o *dwg.Object, b dwg.Body) { ray("AcDbXline")(m, x, o, b.(*dwg.Ray)) }
	register("SPLINE", 0x24, splineFields)
	register("MLINE", 0x2F, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.MLine) {
		m.Subclass("AcDbMline")
		x.NameRef(m, 2, &b.Style, "MLINESTYLE", 5)
		x.HandleRef(m, 340, &b.Style, 5)
		m.Float(40, &b.Scale)
		m.Byte(70, &b.Just)
		m.Int16(71, &b.OpenClosed)
		nv := m.Count(72, len(b.Verts))
		m.Byte(73, &b.LinesInSty)
		m.Point3(10, &b.BasePoint)
		m.Point3(210, &b.Extrusion)
		if m.Reading() {
			b.Verts = make([]dwg.MLineVertex, nv)
		}
		for i := range b.Verts {
			v := &b.Verts[i]
			m.Point3(11, &v.Point)
			m.Point3(12, &v.VertDir)
			m.Point3(13, &v.MiterDir)
			if m.Reading() {
				v.Lines = make([]dwg.MLineSeg, b.LinesInSty)
			}
			for k := range v.Lines {
				l := &v.Lines[k]
				n := m.Count(74, len(l.SegParms))
				if m.Reading() {
					l.SegParms = make([]float64, n)
				}
				for j := range l.SegParms {
					m.Float(41, &l.SegParms[j])
				}
				n = m.Count(75, len(l.FillParms))
				if m.Reading() {
					l.FillParms = make([]float64, n)
				}
				for j := range l.FillParms {
					m.Float(42, &l.FillParms[j])
				}
			}
		}
	})
}

// bAlias registers a second DXF name for a Go type whose DWG type number differs (SOLID/TRACE,
// RAY/XLINE). Export picks the name by DWG type; import uses the alias mapping's type number.
// The returned alias may override fields; export then dispatches to it as well.
func bAlias(base *mapping, dxfName string, dwgType int) *mapping {
	alias := *base
	alias.dxfName, alias.dwgType = dxfName, dwgType
	byDXFName[dxfName] = append(byDXFName[dxfName], &alias)
	baseFields := base.fields
	base.name = func(o *dwg.Object) string {
		if o.Type == dwgType {
			return dxfName
		}
		return base.dxfName
	}
	base.fields = func(m *dxf.Mapper, x *ctx, o *dwg.Object, b dwg.Body) {
		if o.Type == dwgType {
			alias.fields(m, x, o, b)
			return
		}
		baseFields(m, x, o, b)
	}
	return &alias
}

func splineFields(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Spline) {
	m.Subclass("AcDbSpline")
	// fit layout: R2004 scenario 2, or R2013+ Flags1&1 (see the dwg type).
	fit := b.Scenario == 2 || (x.d.Version.R2013 && b.Flags1&1 != 0)
	// DXF 70: 1 closed, 2 periodic, 4 rational, 8 planar, 16 linear; ODA adds 1024|32 for
	// splines defined by fit points and sets periodic on closed splines.
	var flags int16
	if !m.Reading() {
		if b.Closed {
			flags |= 1 | 2
		}
		if b.Periodic {
			flags |= 2
		}
		if b.Rational {
			flags |= 4
		}
		if splinePlanar(b) {
			flags |= 8
			n := [3]float64{0, 0, 1}
			m.Point3(210, &n)
		}
		if b.Degree == 1 {
			flags |= 16
		}
		if fit {
			flags |= 1024 | 32
		}
	} else {
		var n [3]float64
		m.Point3Def(210, &n, [3]float64{})
	}
	m.Int16(70, &flags)
	m.Int32(71, &b.Degree)
	nk := m.Count(72, len(b.Knots))
	nc := m.Count(73, len(b.CtrlPts))
	nf := m.Count(74, len(b.FitPts))
	if !fit || m.Reading() {
		m.Float(42, &b.KnotTol)
		m.Float(43, &b.CtrlTol)
	}
	if m.Reading() {
		fit = flags&1024 != 0 || (nk == 0 && nf > 0)
	}
	if fit {
		m.Float(44, &b.FitTol)
		m.Point3Def(12, &b.BegTangent, [3]float64{})
		m.Point3Def(13, &b.EndTangent, [3]float64{})
	}
	if m.Reading() {
		b.Closed, b.Periodic, b.Rational = flags&1 != 0, flags&2 != 0, flags&4 != 0
		// ODA reports closed splines as periodic; R2013+ files store periodic=false for them.
		if b.Closed && x.d.Version.R2013 {
			b.Periodic = false
		}
		b.Knots = make([]float64, nk)
		b.CtrlPts = make([][3]float64, nc)
		b.FitPts = make([][3]float64, nf)
	}
	for i := range b.Knots {
		m.Float(40, &b.Knots[i])
	}
	if m.Reading() && m.Has(41) {
		b.Weights = make([]float64, nc)
	}
	for i := range b.CtrlPts {
		m.Point3(10, &b.CtrlPts[i])
		if len(b.Weights) > 0 {
			m.Float(41, &b.Weights[i])
		}
	}
	for i := range b.FitPts {
		m.Point3(11, &b.FitPts[i])
	}
	if m.Reading() {
		// The DWG fit layout keeps only the fit data; ODA's computed control points are dropped.
		b.Weighted = len(b.Weights) > 0
		b.Scenario, b.Flags1, b.KnotParam = 1, 0, 0
		if x.d.Version.R2013 {
			b.KnotParam = 15
		}
		if fit {
			b.Knots, b.CtrlPts, b.Weights, b.Weighted = nil, nil, nil, false
			b.KnotTol, b.CtrlTol = 0, 0
			b.Rational, b.Closed, b.Periodic = false, false, false
			b.Scenario, b.KnotParam = 2, 0
			if x.d.Version.R2013 {
				b.Scenario, b.Flags1 = 1, 9
			}
		}
	}
}

// splinePlanar reports whether all points of the spline lie in a plane z = const.
func splinePlanar(b *dwg.Spline) bool {
	pts := append(append([][3]float64{}, b.CtrlPts...), b.FitPts...)
	for _, p := range pts {
		if p[2] != pts[0][2] {
			return false
		}
	}
	return true
}
