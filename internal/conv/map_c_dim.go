package conv

import (
	"strconv"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// dimension type numbers in DXF group 70 (low 3 bits) per DWG fixed type.
var cDimKind = map[int]int16{0x15: 0, 0x16: 1, 0x18: 2, 0x1A: 3, 0x19: 4, 0x17: 5, 0x14: 6}

func init() {
	cRegDim(0x14, "AcDbOrdinateDimension", func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DimOrdinate) {
		cDimCommon(m, x, o, &b.DimCommon, &b.Pt10, 6, func(f *int16) {
			if !m.Reading() {
				if b.Flags2&1 != 0 {
					*f |= 64
				}
			} else if *f&64 != 0 {
				b.Flags2 = 1
			}
		})
		m.Subclass("AcDbOrdinateDimension")
		m.Point3(13, &b.Pt13)
		m.Point3(14, &b.Pt14)
	})
	cRegDim(0x15, "", func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DimLinear) {
		cDimCommon(m, x, o, &b.DimCommon, &b.Pt10, 0, nil)
		m.Subclass("AcDbAlignedDimension")
		m.Point3(13, &b.Pt13)
		m.Point3(14, &b.Pt14)
		m.Angle(50, &b.DimRot)
		m.AngleDef(52, &b.ExtLineRot, 0)
		m.Subclass("AcDbRotatedDimension")
	})
	cRegDim(0x16, "", func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DimAligned) {
		cDimCommon(m, x, o, &b.DimCommon, &b.Pt10, 1, nil)
		m.Subclass("AcDbAlignedDimension")
		m.Point3(13, &b.Pt13)
		m.Point3(14, &b.Pt14)
		m.AngleDef(52, &b.ExtLineRot, 0)
	})
	cRegDim(0x17, "", func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DimAng3Pt) {
		cDimCommon(m, x, o, &b.DimCommon, &b.Pt10, 5, nil)
		m.Subclass("AcDb3PointAngularDimension")
		m.Point3(13, &b.Pt13)
		m.Point3(14, &b.Pt14)
		m.Point3(15, &b.Pt15)
	})
	cRegDim(0x18, "", func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DimAng2Ln) {
		cDimCommon(m, x, o, &b.DimCommon, &b.Pt10, 2, nil)
		m.Subclass("AcDb2LineAngularDimension")
		m.Point3(13, &b.Pt13)
		m.Point3(14, &b.Pt14)
		m.Point3(15, &b.Pt15)
		p16 := [3]float64{b.Pt16[0], b.Pt16[1], b.Elevation}
		m.Point3(16, &p16)
		if m.Reading() {
			b.Pt16 = [2]float64{p16[0], p16[1]}
		}
	})
	cRegDim(0x19, "", func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DimRadius) {
		cDimCommon(m, x, o, &b.DimCommon, &b.Pt10, 4, nil)
		m.Subclass("AcDbRadialDimension")
		m.Point3(15, &b.Pt15)
		m.Float(40, &b.LeaderLen)
	})
	cRegDim(0x1A, "", func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DimDiameter) {
		cDimCommon(m, x, o, &b.DimCommon, &b.Pt10, 3, nil)
		m.Subclass("AcDbDiametricDimension")
		m.Point3(15, &b.Pt15)
		m.Float(40, &b.LeaderLen)
	})
	register("ARC_DIMENSION", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DimArc) {
		cDimCommon(m, x, o, &b.DimCommon, &b.Pt10, 5, nil)
		m.Subclass("AcDbArcDimension")
		m.Point3(13, &b.Pt13)
		m.Point3(14, &b.Pt14)
		m.Point3(15, &b.Center)
		m.Bool(70, &b.IsPartial)
		m.Float(40, &b.StartAngle)
		m.Float(41, &b.EndAngle)
		m.Bool(71, &b.HasLeader)
		m.Point3(16, &b.Leader1)
		m.Point3(17, &b.Leader2)
	})
	register("LARGE_RADIAL_DIMENSION", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DimLargeRadial) {
		cDimCommon(m, x, o, &b.DimCommon, &b.Center, 9, nil)
		m.Subclass("AcDbRadialDimensionLarge")
		m.Point3(13, &b.Chord)
		m.Point3(14, &b.OverrideCenter)
		m.Point3(15, &b.Jog)
		m.Float(40, &b.Unknown40)
	})
}

// cRegDim registers one DIMENSION variant, selected on import by the 70 type bits.
func cRegDim[T dwg.Body](typ int, _ string, f func(m *dxf.Mapper, x *ctx, o *dwg.Object, b T)) {
	kind := cDimKind[typ]
	mp := register("DIMENSION", typ, f)
	mp.match = func(r *dxf.Record) bool {
		v, ok := r.Value(70)
		if !ok {
			return kind == 0
		}
		n, _ := strconv.Atoi(stringsTrim(v))
		return int16(n)&7 == kind
	}
}

func stringsTrim(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}

// cDimCommon binds the AcDbDimension subclass (common dimension data, §20.4.22).
// Group 70 is composed of the kind, Flags1 (bit0 = NOT 128, bit1 = 32) and extra bits (ordinate 64).
func cDimCommon(m *dxf.Mapper, x *ctx, o *dwg.Object, d *dwg.DimCommon, pt10 *[3]float64, kind int16, extra func(*int16)) {
	m.Subclass("AcDbDimension")
	if x.d.Version.R2010 {
		m.Byte(280, &d.Version)
	}
	x.NameRef(m, 2, &d.Block, "BLOCK_RECORD", 5)
	m.Point3(10, pt10)
	mid := [3]float64{d.TextMidpt[0], d.TextMidpt[1], d.Elevation}
	m.Point3(11, &mid)
	p12 := [3]float64{d.Pt12[0], d.Pt12[1], d.Elevation}
	if !m.Reading() {
		if d.Pt12 != [2]float64{} {
			m.Point3(12, &p12)
		}
	} else if m.Has(12) {
		m.Point3(12, &p12)
	}
	var f int16
	if !m.Reading() {
		f = kind
		if d.Flags1&1 == 0 {
			f |= 128
		}
		if d.Flags1&2 != 0 {
			f |= 32
		}
		if extra != nil {
			extra(&f)
		}
	}
	m.Int16(70, &f)
	if m.Reading() {
		d.Flags1 = 0x08 // set in every AutoCAD-written sample; meaning unknown
		if f&128 == 0 {
			d.Flags1 |= 1
		}
		if f&32 != 0 {
			d.Flags1 |= 2
		}
		if extra != nil {
			extra(&f)
		}
	}
	m.StrOpt(1, &d.UserText)
	m.AngleDef(53, &d.TextRot, 0)
	m.AngleDef(51, &d.HorizDir, 0)
	m.Int16Def(71, &d.Attachment, 0)
	m.Int16Def(72, &d.LSpaceStyle, 1)
	m.FloatDef(41, &d.LSpaceFactor, 1)
	m.Float(42, &d.ActualMeasure)
	if x.d.Version.R2007 {
		m.Bool(73, &d.Unknown73)
		m.Bool(74, &d.FlipArrow1)
		m.Bool(75, &d.FlipArrow2)
	}
	x.NameRef(m, 3, &d.DimStyle, "DIMSTYLE", 5)
	m.Extrusion(&d.Extrusion)
	if m.Reading() {
		d.TextMidpt = [2]float64{mid[0], mid[1]}
		d.Elevation = mid[2]
		d.Pt12 = [2]float64{p12[0], p12[1]}
		d.InsScale = [3]float64{1, 1, 1}
	}
}
