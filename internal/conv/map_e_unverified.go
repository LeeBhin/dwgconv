package conv

import (
	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// Mappings for types absent from the sample drawings, written from the AutoCAD DXF reference (unverified).

func init() {
	register("GROUP", 0x48, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Group) {
		m.Subclass("AcDbGroup")
		m.Str(300, &b.Name)
		m.Int16(70, &b.Unnamed)
		m.Int16(71, &b.Selectable)
		eRefList(m, x, 340, &b.Entities, 3)
	})
	register("IMAGEDEF_REACTOR", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.ImageDefReactor) {
		m.Subclass("AcDbRasterImageDefReactor")
		m.Int32(90, &b.ClassVersion)
	})
	register("RASTERVARIABLES", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.RasterVariables) {
		m.Subclass("AcDbRasterVariables")
		m.Int32(90, &b.ClassVersion)
		m.Int16(70, &b.DisplayFrame)
		m.Int16(71, &b.DisplayQual)
		m.Int16(72, &b.Units)
	})
	register("SPATIAL_FILTER", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.SpatialFilter) {
		m.Subclass("AcDbFilter")
		m.Subclass("AcDbSpatialFilter")
		n := m.Count(70, len(b.Points))
		if m.Reading() {
			b.Points = make([][2]float64, n)
		}
		for i := range b.Points {
			m.Point2(10, &b.Points[i])
		}
		m.Point3(210, &b.Extrusion)
		m.Point3(11, &b.ClipOrigin)
		m.Int16(71, &b.DisplayBound)
		m.Int16(72, &b.FrontClipOn)
		if b.FrontClipOn == 1 {
			m.Float(40, &b.FrontDist)
		}
		m.Int16(73, &b.BackClipOn)
		if b.BackClipOn == 1 {
			m.Float(41, &b.BackDist)
		}
		for i := range b.InvBlockTr {
			m.Float(40, &b.InvBlockTr[i])
		}
		for i := range b.ClipBoundTr {
			m.Float(40, &b.ClipBoundTr[i])
		}
	})
	register("FIELD", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Field) {
		m.Subclass("AcDbField")
		m.Str(1, &b.EvaluatorID)
		eLongStr(m, 2, 3, &b.FieldCode)
		n := m.Count(90, len(b.Children))
		if m.Reading() {
			b.Children = make([]dwg.Ref, n)
		}
		for i := range b.Children {
			x.HandleRef(m, 360, &b.Children[i], 3)
		}
		n = m.Count(97, len(b.FieldObjects))
		if m.Reading() {
			b.FieldObjects = make([]dwg.Ref, n)
		}
		for i := range b.FieldObjects {
			x.HandleRef(m, 331, &b.FieldObjects[i], 4)
		}
		if !x.d.Version.R2007 {
			m.Str(4, &b.Format)
		}
		m.Int32(91, &b.EvalOption)
		m.Int32(92, &b.FilingOption)
		m.Int32(94, &b.State)
		m.Int32(95, &b.EvalStatus)
		m.Int32(96, &b.EvalErrorCode)
		m.Str(300, &b.EvalErrorMsg)
		n = m.Count(93, len(b.ChildKeys))
		if m.Reading() {
			b.ChildKeys = make([]string, n)
			b.ChildValues = make([]dwg.Value, n)
		}
		for i := range b.ChildKeys {
			m.Str(6, &b.ChildKeys[i])
			eValue(m, x, &b.ChildValues[i])
		}
		key := "ACFD_FIELD_VALUE"
		m.Str(7, &key)
		eValue(m, x, &b.Value)
		eLongStr(m, 301, 9, &b.ValueString)
		m.Int32(98, &b.ValueStrLen)
	})
	register("GEODATA", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.GeoData) {
		m.Subclass("AcDbGeoData")
		m.Int32(90, &b.ObjectVersion)
		x.HandleRef(m, 330, &b.HostBlock, 4)
		m.Int16(70, &b.CoordType)
		m.Point3(10, &b.DesignPoint)
		m.Point3(11, &b.ReferencePoint)
		m.Float(40, &b.HorzUnitScale)
		m.Int32(91, &b.HorzUnits)
		m.Float(41, &b.VertUnitScale)
		m.Int32(92, &b.VertUnits)
		m.Point3(210, &b.UpDirection)
		north := [2]float64{b.NorthDirection[0], b.NorthDirection[1]}
		m.Point2(12, &north)
		m.Int32(95, &b.ScaleEstimation)
		m.Float(141, &b.UserScale)
		m.Bool(294, &b.SeaLevelCorr)
		m.Float(142, &b.SeaLevelElev)
		m.Float(143, &b.ProjRadius)
		eLongStr(m, 301, 303, &b.CoordSystemDef)
		m.Str(302, &b.GeoRSSTag)
		m.Str(305, &b.ObsFromTag)
		m.Str(306, &b.ObsToTag)
		m.Str(307, &b.ObsCoverTag)
		n := m.Count(93, len(b.MeshPoints))
		if m.Reading() {
			b.NorthDirection = [3]float64{north[0], north[1], 0}
			b.MeshPoints = make([]dwg.GeoMeshPoint, n)
		}
		for i := range b.MeshPoints {
			m.Point2(13, &b.MeshPoints[i].Source)
			m.Point2(14, &b.MeshPoints[i].Dest)
		}
		n = m.Count(96, len(b.MeshFaces))
		if m.Reading() {
			b.MeshFaces = make([][3]int32, n)
		}
		for i := range b.MeshFaces {
			m.Int32(97, &b.MeshFaces[i][0])
			m.Int32(98, &b.MeshFaces[i][1])
			m.Int32(99, &b.MeshFaces[i][2])
		}
	})
}

// eLongStr binds a string written in 250-character chunks: the first in code first, the rest in code rest.
func eLongStr(m *dxf.Mapper, first, rest int, v *string) {
	const chunk = 250
	if !m.Reading() {
		s := *v
		head := s[:min(len(s), chunk)]
		m.Str(first, &head)
		for s = s[len(head):]; len(s) > 0; {
			part := s[:min(len(s), chunk)]
			m.Str(rest, &part)
			s = s[len(part):]
		}
		return
	}
	m.Str(first, v)
	for m.Peek(rest) {
		var part string
		m.Str(rest, &part)
		*v += part
	}
}

// eValue binds a field value (§20.4.99) in the DXF form: 90 data type, then the typed value.
func eValue(m *dxf.Mapper, x *ctx, v *dwg.Value) {
	if x.d.Version.R2007 {
		m.Int32(93, &v.Flags)
	}
	m.Int32(90, &v.DataType)
	if v.Flags&1 == 0 {
		switch v.DataType {
		case 0, 1:
			m.Int32(91, &v.Long)
		case 2:
			m.Float(140, &v.Double)
		case 4:
			m.Str(1, &v.String)
		case 8, 512:
			n := m.Count(92, len(v.Bytes))
			m.Binary(310, &v.Bytes)
			if m.Reading() && len(v.Bytes) > n {
				v.Bytes = v.Bytes[:n]
			}
		case 16, 32:
			p := [3]float64{}
			copy(p[:], v.Point)
			m.Point3(11, &p)
			if m.Reading() {
				k := 2
				if v.DataType == 32 {
					k = 3
				}
				v.Point = p[:k]
			}
		case 64:
			x.HandleRef(m, 330, &v.ObjectID, 4)
		}
	}
	if x.d.Version.R2007 {
		m.Int32(94, &v.UnitType)
		m.Str(300, &v.FormatStr)
		m.Str(302, &v.ValueStr)
	}
}
