package conv

import (
	"encoding/binary"
	"math"
	"reflect"

	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

func init() {
	register("VIEWPORT", 0x22, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.ViewportEnt) {
		m.Subclass("AcDbViewport")
		m.Point3(10, &b.Center)
		m.Float(40, &b.Width)
		m.Float(41, &b.Height)
		// 68 status and 69 id are not stored in DWG: id = order among the viewports of the same layout.
		id := int16(cViewportID(x, o))
		st := id
		if b.StatusFlags&0x20000 != 0 {
			st = 0
		}
		m.Int16(68, &st)
		m.Int16(69, &id)
		m.Point2(12, &b.ViewCenter)
		m.Point2(13, &b.SnapBase)
		m.Point2(14, &b.SnapSpacing)
		m.Point2(15, &b.GridSpacing)
		m.Point3(16, &b.ViewDir)
		m.Point3(17, &b.ViewTarget)
		m.Float(42, &b.LensLength)
		m.Float(43, &b.FrontClip)
		m.Float(44, &b.BackClip)
		m.Float(45, &b.ViewHeight)
		m.Angle(50, &b.SnapAngle)
		m.Angle(51, &b.TwistAngle)
		m.Int16(72, &b.CircleZoom)
		cRefList(m, x, 331, &b.FrozenLayers, 4)
		m.Int32(90, &b.StatusFlags)
		x.HandleRefOpt(m, 340, &b.ClipBoundary, 4)
		m.Str(1, &b.StyleSheet)
		m.Byte(281, &b.RenderMode)
		m.Bool(71, &b.UCSPerVport)
		m.Bool(74, &b.UCSAtOrigin)
		m.Point3(110, &b.UCSOrigin)
		m.Point3(111, &b.UCSXAxis)
		m.Point3(112, &b.UCSYAxis)
		x.HandleRefOpt(m, 345, &b.NamedUCS, 5)
		x.HandleRefOpt(m, 346, &b.BaseUCS, 5)
		m.Int16(79, &b.UCSOrthoType)
		m.Float(146, &b.UCSElevation)
		m.Int16(170, &b.ShadePlotMode)
		r2007 := x.d.Version.R2007
		if !r2007 {
			// R2004 has none of these; ODA writes the defaults.
			gm, lights, lt, amb := int16(5), true, byte(1), dwg.CmColor{Index: 256}
			vs := cDefaultVisualStyle(x)
			if !m.Reading() {
				m.Int16(61, &gm)
				x.HandleRefOpt(m, 348, &vs, 5)
				m.Bool(292, &lights)
				m.Byte(282, &lt)
				var z float64
				m.Float(141, &z)
				m.Float(142, &z)
				cCMC(m, 63, &amb)
			}
			return
		}
		m.Int16(61, &b.GridMajor)
		x.HandleRefOpt(m, 332, &b.Background, 5)
		x.HandleRefOpt(m, 333, &b.ShadePlot, 5)
		x.HandleRefOpt(m, 348, &b.VisualStyle, 5)
		m.Bool(292, &b.UseDefLights)
		m.Byte(282, &b.DefLightType)
		m.Float(141, &b.Brightness)
		m.Float(142, &b.Contrast)
		cCMC(m, 63, &b.AmbientColor)
		x.HandleRefOpt(m, 361, &b.Sun, 3)
	})

	register("OLE2FRAME", 0x4A, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Ole2Frame) {
		m.Subclass("AcDbOle2Frame")
		m.Int16(70, &b.Flags)
		// 10/11 (upper left, lower right) are not stored separately: they are the first and
		// third corner in the OLE data header (2-byte prefix, then 4 corners of 3 doubles).
		ul, lr := cOleCorner(b.Data, 0), cOleCorner(b.Data, 2)
		m.Point3(10, &ul)
		m.Point3(11, &lr)
		// 72 is the tile mode (0 model space, 1 paper space), derived from the entity mode.
		tile := int16(0)
		if o.Ent.EntMode == 1 {
			tile = 1
		}
		m.Int16(72, &tile)
		u := int16(b.Unknown)
		m.Int16(73, &u)
		b.Unknown = byte(u)
		n := int32(len(b.Data))
		m.Int32(90, &n)
		if !m.Reading() {
			for d := b.Data; len(d) > 0; d = d[min(32, len(d)):] { // ODA writes OLE data in 32-byte lines
				m.Writer().Binary(310, d[:min(32, len(d))])
			}
		} else {
			m.Binary(310, &b.Data)
		}
		end := "OLE"
		m.Str(1, &end)
	})
}

// cOleCorner reads corner i of the OLE2 data header (unverified layout beyond the samples).
func cOleCorner(d []byte, i int) [3]float64 {
	var p [3]float64
	off := 2 + i*24
	if len(d) < off+24 {
		return p
	}
	for k := range p {
		p[k] = math.Float64frombits(binary.LittleEndian.Uint64(d[off+8*k:]))
	}
	return p
}

// cRefList binds a run of handle groups with the same code.
func cRefList(m *dxf.Mapper, x *ctx, code int, refs *[]dwg.Ref, refCode byte) {
	if !m.Reading() {
		for i := range *refs {
			x.HandleRef(m, code, &(*refs)[i], refCode)
		}
		return
	}
	for m.Has(code) {
		var r dwg.Ref
		x.HandleRef(m, code, &r, refCode)
		*refs = append(*refs, r)
	}
}

// cCMC binds a CMC color as code (ACI), 420+code-62... (421/431 for 63).
func cCMC(m *dxf.Mapper, code int, c *dwg.CmColor) {
	m.Int16(code, &c.Index)
	rgbCode := 420 + (code - 62)
	if !m.Reading() {
		if uint32(c.RGB)>>24 == 0xC2 {
			rgb := c.RGB & 0xFFFFFF
			m.Int32(rgbCode, &rgb)
		}
		return
	}
	if m.Has(rgbCode) {
		var rgb int32
		m.Int32(rgbCode, &rgb)
		c.RGB = rgb | -0x3E000000
	}
}

// cViewportID numbers the viewports of a layout (same owner) in handle order, starting at 1.
func cViewportID(x *ctx, o *dwg.Object) int {
	id := 1
	for _, p := range x.d.Objects {
		if p.Type == 0x22 && p != o && p.Owner.H == o.Owner.H && p.Ent != nil && p.Ent.EntMode == o.Ent.EntMode && p.Handle < o.Handle {
			id++
		}
	}
	return id
}

// cDefaultVisualStyle finds the "2dWireframe" entry of the ACAD_VISUALSTYLE dictionary,
// which ODA references for R2004 viewports.
func cDefaultVisualStyle(x *ctx) dwg.Ref {
	for _, p := range x.d.Objects {
		root, ok := p.Body.(*dwg.Dictionary)
		if !ok {
			continue
		}
		for i, n := range root.Names {
			if n != "ACAD_VISUALSTYLE" {
				continue
			}
			if vs, ok := x.d.ByHandle[root.Items[i].H]; ok {
				if d, ok := vs.Body.(*dwg.Dictionary); ok {
					for j, m := range d.Names {
						if m == "2dWireframe" {
							return dwg.Ref{Code: 5, H: d.Items[j].H}
						}
					}
				}
			}
		}
	}
	return dwg.Ref{}
}

// cStringField reads an exported string field of a body by name.
func cStringField(b dwg.Body, name string) (string, bool) {
	f := reflect.ValueOf(b).Elem().FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.String {
		return "", false
	}
	return f.String(), true
}
