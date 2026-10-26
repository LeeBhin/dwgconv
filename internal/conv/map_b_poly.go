package conv

import (
	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// Group B: old-style POLYLINE/VERTEX variants and LWPOLYLINE.

func init() {
	flag70 := func(r *dxf.Record) int64 {
		v, _ := r.Value(70)
		g := dxf.Group{Code: 70, Value: v}
		return g.Int()
	}

	p2 := register("POLYLINE", 0x0F, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Polyline2D) {
		m.Subclass("AcDb2dPolyline")
		bFollows(m)
		elev := [3]float64{0, 0, b.Elevation}
		m.Point3(10, &elev)
		m.FloatDef(39, &b.Thickness, 0)
		m.Int16Def(70, &b.Flags, 0)
		m.FloatDef(40, &b.StartWidth, 0)
		m.FloatDef(41, &b.EndWidth, 0)
		m.Int16Def(75, &b.CurveType, 0)
		m.Extrusion(&b.Extrusion)
		if m.Reading() {
			b.Elevation = elev[2]
		}
	})
	p2.match = func(r *dxf.Record) bool { return flag70(r)&(8|16|64) == 0 }

	p3 := register("POLYLINE", 0x10, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Polyline3D) {
		m.Subclass("AcDb3dPolyline")
		bFollows(m)
		var zero [3]float64
		m.Point3(10, &zero)
		// DXF 70 = 8 | closed(1) | splined(4); 75 = 5 (quadratic) or 6 (cubic).
		f := int16(8)
		var curve int16
		if b.ClosedFlags&1 != 0 {
			f |= 1
		}
		switch {
		case b.SplineFlags&1 != 0:
			f |= 4
			curve = 5
		case b.SplineFlags&2 != 0:
			f |= 4
			curve = 6
		}
		m.Int16(70, &f)
		m.Int16Def(75, &curve, 0)
		if m.Reading() {
			b.ClosedFlags = byte(f & 1)
			switch curve {
			case 5:
				b.SplineFlags = 1
			case 6:
				b.SplineFlags = 2
			}
		}
	})
	p3.match = func(r *dxf.Record) bool { return flag70(r)&8 != 0 }

	pf := register("POLYLINE", 0x1D, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.PolylinePFace) {
		m.Subclass("AcDbPolyFaceMesh")
		bFollows(m)
		var zero [3]float64
		m.Point3(10, &zero)
		f := int16(64)
		m.Int16(70, &f)
		m.Int16(71, &b.NumVerts)
		m.Int16(72, &b.NumFaces)
	})
	pf.match = func(r *dxf.Record) bool { return flag70(r)&64 != 0 }

	pm := register("POLYLINE", 0x1E, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.PolylineMesh) {
		m.Subclass("AcDbPolygonMesh")
		bFollows(m)
		var zero [3]float64
		m.Point3(10, &zero)
		f := b.Flags | 16
		m.Int16(70, &f)
		m.Int16Def(71, &b.MCount, 0)
		m.Int16Def(72, &b.NCount, 0)
		m.Int16Def(73, &b.MDensity, 0)
		m.Int16Def(74, &b.NDensity, 0)
		m.Int16Def(75, &b.CurveType, 0)
		if m.Reading() {
			b.Flags = f &^ 16
		}
	})
	pm.match = func(r *dxf.Record) bool { return flag70(r)&16 != 0 }

	v2 := register("VERTEX", 0x0A, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Vertex2D) {
		m.Subclass("AcDbVertex")
		m.Subclass("AcDb2dVertex")
		m.Point3(10, &b.Point)
		// ODA omits widths equal to the owning polyline's default widths.
		var defS, defE float64
		if pl, ok := x.d.ByHandle[o.Owner.H]; ok {
			if p, ok := pl.Body.(*dwg.Polyline2D); ok {
				defS, defE = p.StartWidth, p.EndWidth
			}
		}
		m.FloatDef(40, &b.StartWidth, defS)
		m.FloatDef(41, &b.EndWidth, defE)
		m.FloatDef(42, &b.Bulge, 0)
		f := int16(b.Flags)
		m.Int16Def(70, &f, 0)
		m.AngleDef(50, &b.TangentDir, 0)
		m.Int32Def(91, &b.VertexID, 0)
		if m.Reading() {
			b.Flags = byte(f)
		}
	})
	v2.match = func(r *dxf.Record) bool { return flag70(r)&(32|64|128) == 0 }

	v3fields := func(sub string) func(m *dxf.Mapper, x *ctx, o *dwg.Object, b dwg.Body) {
		return func(m *dxf.Mapper, x *ctx, o *dwg.Object, bb dwg.Body) {
			b := bb.(*dwg.Vertex3D)
			m.Subclass("AcDbVertex")
			m.Subclass(sub)
			m.Point3(10, &b.Point)
			f := int16(b.Flags)
			m.Int16(70, &f)
			if m.Reading() {
				b.Flags = byte(f)
			}
		}
	}
	v3 := register("VERTEX", 0x0B, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Vertex3D) {
		v3fields("AcDb3dPolylineVertex")(m, x, o, b)
	})
	v3.match = func(r *dxf.Record) bool { return flag70(r)&32 != 0 }
	vm := bAlias(v3, "VERTEX", 0x0C)
	vm.fields = v3fields("AcDbPolygonMeshVertex")
	vm.match = func(r *dxf.Record) bool { return flag70(r)&(64|128) == 64 }
	vp := bAlias(v3, "VERTEX", 0x0D)
	vp.fields = v3fields("AcDbPolyFaceMeshVertex")
	vp.match = func(r *dxf.Record) bool { return flag70(r)&(64|128) == 192 }

	vf := register("VERTEX", 0x0E, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.VertexPFaceFace) {
		m.Subclass("AcDbFaceRecord")
		var zero [3]float64
		m.Point3(10, &zero)
		f := int16(128)
		m.Int16(70, &f)
		for i := range b.Index {
			m.Int16Def(71+i, &b.Index[i], 0)
		}
	})
	vf.match = func(r *dxf.Record) bool { return flag70(r)&(64|128) == 128 }

	register("LWPOLYLINE", 0x4D, lwpolylineFields)
}

// bFollows writes the obsolete "entities follow" flag (66) of POLYLINE.
func bFollows(m *dxf.Mapper) {
	one := int16(1)
	m.Int16Def(66, &one, 0)
}

func lwpolylineFields(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.LWPolyline) {
	m.Subclass("AcDbPolyline")
	n := m.Count(90, len(b.Points))
	// DWG flag 512 = closed (DXF 1), 256 = plinegen (DXF 128); the other DWG bits only
	// mark optional data and are recomputed by the encoder.
	var f int16
	if b.Flag&512 != 0 {
		f |= 1
	}
	if b.Flag&256 != 0 {
		f |= 128
	}
	m.Int16(70, &f)
	hasW := len(b.Widths) > 0
	if m.Reading() {
		hasW = m.Has(40)
	}
	if !hasW {
		m.Float(43, &b.ConstWidth) // ODA always writes 43 when there are no per-vertex widths
	}
	m.FloatDef(38, &b.Elevation, 0)
	m.FloatDef(39, &b.Thickness, 0)
	hasB, hasID := len(b.Bulges) > 0, len(b.VertexIDs) > 0
	if m.Reading() {
		b.Flag = 0
		if f&1 != 0 {
			b.Flag |= 512
		}
		if f&128 != 0 {
			b.Flag |= 256
		}
		hasB, hasID = m.Has(42), m.Has(91)
		b.Points = make([][2]float64, n)
		if hasW {
			b.Widths = make([][2]float64, n)
		}
		if hasB {
			b.Bulges = make([]float64, n)
		}
		if hasID {
			b.VertexIDs = make([]int32, n)
		}
	}
	for i := range b.Points {
		m.Point2(10, &b.Points[i])
		if hasW && (!m.Reading() || m.Peek(40)) {
			m.Float(40, &b.Widths[i][0])
			m.Float(41, &b.Widths[i][1])
		}
		if hasB && (m.Reading() && m.Peek(42) || !m.Reading() && b.Bulges[i] != 0) {
			m.Float(42, &b.Bulges[i])
		}
		if hasID && (!m.Reading() || m.Peek(91)) {
			m.Int32(91, &b.VertexIDs[i])
		}
	}
	m.Extrusion(&b.Normal)
}
