package conv

import (
	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

func init() {
	register("ACDBSECTIONVIEWSTYLE", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.SectionViewStyle) {
		cModelDoc(m, x, &b.ModelDocViewStyle, "AcDbSectionViewStyle")
		cMarker71(m, 0)
		m.Int32(90, &b.Flags)
		cMarker71(m, 1)
		x.HandleRef(m, 340, &b.IDStyle, 5)
		eCMC(m, 62, &b.IDColor)
		m.Float(40, &b.IDHeight)
		x.HandleRef(m, 340, &b.ArrowStart, 5)
		x.HandleRef(m, 340, &b.ArrowEnd, 5)
		eCMC(m, 62, &b.ArrowColor)
		m.Float(40, &b.ArrowSize)
		m.Str(300, &b.ExcludeChars)
		m.Float(40, &b.ArrowExtLen)
		m.Int32(90, &b.IDPosition)
		m.Float(40, &b.IDOffset)
		m.Int32(90, &b.ArrowPosition)
		cMarker71(m, 2)
		x.HandleRef(m, 340, &b.PlaneLtype, 5)
		m.Int32(90, &b.PlaneLineweight)
		eCMC(m, 62, &b.PlaneColor)
		x.HandleRef(m, 340, &b.BendLtype, 5)
		m.Int32(90, &b.BendLineweight)
		eCMC(m, 62, &b.BendColor)
		m.Float(40, &b.BendLength)
		m.Float(40, &b.EndOvershoot)
		m.Float(40, &b.EndLength)
		cMarker71(m, 3)
		x.HandleRef(m, 340, &b.LabelStyle, 5)
		eCMC(m, 62, &b.LabelColor)
		m.Float(40, &b.LabelHeight)
		m.Int32(90, &b.LabelAttach)
		m.Float(40, &b.LabelOffset)
		m.Int32(90, &b.LabelAlign)
		m.Str(300, &b.LabelPattern)
		cMarker71(m, 4)
		eCMC(m, 62, &b.HatchColor)
		eCMC(m, 62, &b.HatchBgColor)
		m.Str(300, &b.HatchPattern)
		m.Float(40, &b.HatchScale)
		m.Int32(90, &b.HatchTransp)
		m.Bool(290, &b.ShowHatch)
		m.Bool(290, &b.Unknown290)
		n := m.Count(90, len(b.HatchAngles))
		if m.Reading() {
			b.HatchAngles = make([]float64, n)
		}
		for i := range b.HatchAngles {
			m.Float(40, &b.HatchAngles[i])
		}
	})
	register("ACDBDETAILVIEWSTYLE", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.DetailViewStyle) {
		cModelDoc(m, x, &b.ModelDocViewStyle, "AcDbDetailViewStyle")
		cMarker71(m, 0)
		m.Int32(90, &b.Flags)
		cMarker71(m, 1)
		x.HandleRef(m, 340, &b.IDStyle, 5)
		eCMC(m, 62, &b.IDColor)
		m.Float(40, &b.IDHeight)
		x.HandleRef(m, 340, &b.ArrowSymbol, 5)
		eCMC(m, 62, &b.ArrowColor)
		m.Float(40, &b.ArrowSize)
		m.Str(300, &b.ExcludeChars)
		m.Float(40, &b.ArrowExtLen)
		m.Byte(280, &b.IDPlacement)
		cMarker71(m, 2)
		x.HandleRef(m, 340, &b.BoundaryLtype, 5)
		m.Int32(90, &b.BoundaryLineweight)
		eCMC(m, 62, &b.BoundaryColor)
		cMarker71(m, 3)
		x.HandleRef(m, 340, &b.LabelStyle, 5)
		eCMC(m, 62, &b.LabelColor)
		m.Float(40, &b.LabelHeight)
		m.Int32(90, &b.LabelAttach)
		m.Float(40, &b.LabelOffset)
		m.Int32(90, &b.LabelAlign)
		m.Str(300, &b.LabelPattern)
		cMarker71(m, 4)
		x.HandleRef(m, 340, &b.BorderLtype, 5)
		m.Int32(90, &b.BorderLineweight)
		eCMC(m, 62, &b.BorderColor)
		x.HandleRef(m, 340, &b.ConnLtype, 5)
		m.Int32(90, &b.ConnLineweight)
		eCMC(m, 62, &b.ConnColor)
		m.Byte(280, &b.ShowArrows)
	})
	register("RAPIDRTRENDERSETTINGS", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.RapidRTRenderSettings) {
		cRenderSettings(m, &b.RenderSettings)
		m.Subclass("AcDbRapidRTRenderSettings")
		m.Int32(90, &b.Version)
		m.Int16(70, &b.RenderTarget)
		m.Int32(90, &b.RenderLevel)
		m.Int32(90, &b.RenderTime)
		m.Int16(70, &b.LightingModel)
		m.Int16(70, &b.FilterType)
		m.Float(40, &b.FilterWidth)
		m.Float(40, &b.FilterHeight)
		m.Bool(290, &b.Unknown290)
	})
	register("MENTALRAYRENDERSETTINGS", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.MentalRayRenderSettings) {
		cRenderSettings(m, &b.RenderSettings)
		m.Subclass("AcDbMentalRayRenderSettings")
		if m.Reading() {
			b.Fields = make([]dwg.MRField, len(dwg.MentalRayKinds))
			for i, k := range dwg.MentalRayKinds {
				b.Fields[i].Kind = k
			}
		}
		for i := range b.Fields {
			f := &b.Fields[i]
			switch f.Kind {
			case 90:
				m.Int32(90, &f.I)
			case 70:
				v := int16(f.I)
				m.Int16(70, &v)
				f.I = int32(v)
			case 40:
				m.Float(40, &f.F)
			case 290:
				m.Bool(290, &f.B)
			case 1:
				m.Str(1, &f.S)
			}
		}
	})
	register("PDFDEFINITION", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.PdfDefinition) {
		m.Subclass("AcDbUnderlayDefinition")
		m.Str(1, &b.File)
		m.Str(2, &b.Page)
	})
}

// cModelDoc binds AcDbModelDocViewStyle and the leading 70 of the derived subclass.
// Before R2018 the object has a single string; ODA writes it as both 3 and 300.
func cModelDoc(m *dxf.Mapper, x *ctx, b *dwg.ModelDocViewStyle, subclass string) {
	r2018 := x.d.Version.R2018
	m.Subclass("AcDbModelDocViewStyle")
	m.Int16(70, &b.Flags)
	m.Str(3, &b.Name)
	m.Bool(290, &b.Modified)
	desc := b.Description
	if !r2018 {
		desc = b.Name
	}
	m.Str(300, &desc)
	m.Int32(90, &b.Flags90)
	m.Subclass(subclass)
	v := b.Version70
	m.Int16(70, &v)
	if m.Reading() && r2018 {
		b.Description, b.Version70 = desc, v
	}
}

// cMarker71 writes or skips a section marker (71 = 0..4) of the view style records.
func cMarker71(m *dxf.Mapper, n int16) { m.Int16(71, &n) }

func cRenderSettings(m *dxf.Mapper, r *dwg.RenderSettings) {
	m.Subclass("AcDbRenderSettings")
	m.Int32(90, &r.ClassVersion)
	m.Str(1, &r.Name)
	m.Bool(290, &r.FogEnabled)
	m.Bool(290, &r.FogBackground)
	m.Bool(290, &r.BackFaces)
	m.Bool(290, &r.EnvImage)
	m.Str(1, &r.EnvImageFile)
	m.Str(1, &r.Description)
	m.Int32(90, &r.DisplayIndex)
}
