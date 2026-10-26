package conv

import (
	"strings"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

func init() {
	register("LAYOUT", 0x52, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Layout) {
		ePlotSettings(m, x, &b.PlotSettings)
		m.Subclass("AcDbLayout")
		m.Str(1, &b.Name)
		m.Int16(70, &b.Flags)
		m.Int32(71, &b.TabOrder)
		m.Point2(10, &b.LimMin)
		m.Point2(11, &b.LimMax)
		m.Point3(12, &b.InsBase)
		m.Point3(14, &b.ExtMin)
		m.Point3(15, &b.ExtMax)
		m.Float(146, &b.Elevation)
		m.Point3(13, &b.UCSOrigin)
		m.Point3(16, &b.UCSXAxis)
		m.Point3(17, &b.UCSYAxis)
		m.Int16(76, &b.OrthoViewType)
		x.HandleRef(m, 330, &b.BlockRecord, 4)
		x.HandleRefOpt(m, 331, &b.LastViewport, 4)
		x.HandleRefOpt(m, 345, &b.NamedUCS, 5)
		x.HandleRefOpt(m, 346, &b.BaseUCS, 5)
	})
	register("PLOTSETTINGS", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.PlotSettingsObj) {
		ePlotSettings(m, x, &b.PlotSettings)
	})
}

// ePlotSettings binds the AcDbPlotSettings part (§20.4.84). The plot view (a VIEW handle in DWG) is written by name.
func ePlotSettings(m *dxf.Mapper, x *ctx, p *dwg.PlotSettings) {
	m.Subclass("AcDbPlotSettings")
	m.Str(1, &p.PageSetupName)
	m.Str(2, &p.PrinterConfig)
	m.Str(4, &p.PaperSize)
	view := x.names[p.PlotView.H]
	m.Str(6, &view)
	if m.Reading() && view != "" {
		if h, ok := x.byName["VIEW"][strings.ToUpper(view)]; ok {
			p.PlotView = dwg.Ref{Code: 5, H: h}
		}
	}
	m.Float(40, &p.LeftMargin)
	m.Float(41, &p.BottomMargin)
	m.Float(42, &p.RightMargin)
	m.Float(43, &p.TopMargin)
	m.Float(44, &p.PaperWidth)
	m.Float(45, &p.PaperHeight)
	m.Float(46, &p.PlotOrigin[0])
	m.Float(47, &p.PlotOrigin[1])
	m.Float(48, &p.WindowMin[0])
	m.Float(49, &p.WindowMin[1])
	m.Float(140, &p.WindowMax[0])
	m.Float(141, &p.WindowMax[1])
	m.Float(142, &p.RealWorldUnits)
	m.Float(143, &p.DrawingUnits)
	m.Int16(70, &p.PlotLayoutFlags)
	m.Int16(72, &p.PaperUnits)
	m.Int16(73, &p.PlotRotation)
	m.Int16(74, &p.PlotType)
	m.Str(7, &p.CurrentStyleSheet)
	m.Int16(75, &p.ScaleType)
	m.Int16(76, &p.ShadePlotMode)
	m.Int16(77, &p.ShadePlotResLvl)
	m.Int16(78, &p.ShadePlotDPI)
	m.Float(147, &p.ScaleFactor)
	m.Float(148, &p.PaperImageOrigin[0])
	m.Float(149, &p.PaperImageOrigin[1])
}
