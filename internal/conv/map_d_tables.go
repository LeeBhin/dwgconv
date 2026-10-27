package conv

import (
	"sort"
	"strconv"
	"strings"

	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

// Symbol tables (group D): table records and their control objects (DXF "TABLE").

// dTableNames maps control object types to DXF table names.
var dTableNames = map[int]string{
	0x30: "BLOCK_RECORD", 0x32: "LAYER", 0x34: "STYLE", 0x38: "LTYPE", 0x3C: "VIEW",
	0x3E: "UCS", 0x40: "VPORT", 0x42: "APPID", 0x44: "DIMSTYLE", 0x46: "VX",
}

func init() {
	mp := register("TABLE", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.TableControl) {
		if m.Reading() {
			name, _ := m.Record().Value(2)
			for k, v := range dTableNames {
				if strings.EqualFold(v, name) {
					o.Type = k
					b.SetKind(k)
				}
			}
		}
		m.Subclass("AcDbSymbolTable")
		m.Count(70, dLiveEntries(x, b))
		if b.Kind() == 0x44 {
			// ODA writes no 71/340 list here, even when the control has extra handles.
			m.Subclass("AcDbDimStyleTable")
		}
	})
	mp.name = func(o *dwg.Object) string { return "TABLE" }
	mp.preHandle = func(m *dxf.Mapper, x *ctx, o *dwg.Object) {
		s := dTableNames[o.Body.(*dwg.TableControl).Kind()]
		m.Str(2, &s)
	}

	register("LAYER", 0x33, dLayer)
	register("LTYPE", 0x39, dLtype)
	register("STYLE", 0x35, dStyle)
	register("APPID", 0x43, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.AppID) {
		dRecord(m, &b.TableEntry, "AcDbRegAppTableRecord", 0)
		if m.Reading() {
			b.Unknown = 0
		}
	})
	register("BLOCK_RECORD", 0x31, dBlockRecord)
	register("VPORT", 0x41, dVPort)
	register("VIEW", 0x3D, dView)
	register("UCS", 0x3F, dUCS)
	dim := register("DIMSTYLE", 0x45, dDimStyle)
	dim.handleCode = 105
	dim.skipEED = dDimSkipEED
}

// dLiveEntries counts the entries that exist in the drawing (ODA's 70); the DWG list may
// keep handles of erased records.
func dLiveEntries(x *ctx, b *dwg.TableControl) int {
	n := 0
	for _, r := range b.Entries {
		if x.d.ByHandle[r.H] != nil {
			n++
		}
	}
	return n
}

// dRecord binds the table record header: subclasses, 2 name and 70 flags. extra holds the
// type-specific 70 bits (xdep 16 is added here); it returns the 70 value read.
func dRecord(m *dxf.Mapper, e *dwg.TableEntry, subclass string, extra int16) int16 {
	m.Subclass("AcDbSymbolTableRecord")
	m.Subclass(subclass)
	m.Str(2, &e.Name)
	f := extra
	if e.Xdep {
		f |= 16
	}
	m.Int16(70, &f)
	if m.Reading() {
		e.Xdep = f&16 != 0
	}
	return f
}

// CMC color methods (high byte of the R2004+ RGB value).
const (
	dByLayer   = 0xC0
	dByBlock   = 0xC1
	dTrueColor = 0xC2
	dACI       = 0xC3
	dNone      = 0xC8
)

// dACIGray are ACI colors with an exact RGB, used to pick the 62 that ODA writes for true colors.
var dACIRGB = map[int32]int16{
	0xFF0000: 1, 0xFFFF00: 2, 0x00FF00: 3, 0x00FFFF: 4, 0x0000FF: 5, 0xFF00FF: 6, 0xFFFFFF: 7,
	0x333333: 250, 0x5B5B5B: 251, 0x848484: 252, 0xADADAD: 253, 0xD6D6D6: 254,
	// nearest ACI as ODA picks it, observed in visual styles
	0x000000: 18, 0x808080: 8, 0x787878: 8,
}

// dCmcACI returns the ACI written as 62 for a CMC color.
func dCmcACI(c *dwg.CmColor) int16 {
	switch uint32(c.RGB) >> 24 {
	case dByLayer:
		return 256
	case dByBlock:
		return 0
	case dACI:
		return int16(c.RGB & 0xFF)
	case dNone:
		return 257
	case dTrueColor:
		if i, ok := dACIRGB[c.RGB&0xFFFFFF]; ok {
			return i
		}
		return 7
	}
	return c.Index
}

// dCmcFromACI builds the CMC color for an ACI read from DXF.
func dCmcFromACI(i int16) dwg.CmColor {
	switch i {
	case 256:
		return dwg.CmColor{RGB: -0x40000000} // 0xC0000000
	case 0:
		return dwg.CmColor{RGB: -0x3F000000} // 0xC1000000
	case 257:
		return dwg.CmColor{RGB: -0x38000000} // 0xC8000000
	}
	return dwg.CmColor{RGB: -0x3D000000 | int32(uint16(i)&0xFF)}
}

// dColor binds a CMC color as 62 (negated when off) and 420/421 for true colors.
func dColor(m *dxf.Mapper, c *dwg.CmColor, code int, off bool) {
	if !m.Reading() {
		i := dCmcACI(c)
		if off {
			i = -i
		}
		m.Int16(code, &i)
		if uint32(c.RGB)>>24 == dTrueColor {
			rgb := c.RGB & 0xFFFFFF
			m.Int32(code+358, &rgb) // 62 -> 420, 63 -> 421
		}
		return
	}
	var i int16
	m.Int16(code, &i)
	neg := i < 0
	if neg {
		i = -i
	}
	*c = dCmcFromACI(i)
	if neg {
		c.Index = -1 // marker for the caller (layer off); cleared there
	}
	if m.Has(code + 358) {
		var rgb int32
		m.Int32(code+358, &rgb)
		c.RGB = rgb&0xFFFFFF | -0x3E000000 // 0xC2: true color
	}
}

func dLayer(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Layer) {
	// DWG flags: frozen 1, off 2, frozen in new viewports 4, locked 8, plot 16, lineweight << 5.
	var f70 int16
	if !m.Reading() {
		f70 = b.Flags&1 | (b.Flags&4)>>1 | (b.Flags&8)>>1
	}
	f70 = dRecord(m, &b.TableEntry, "AcDbLayerTableRecord", f70)
	off := b.Flags&2 != 0
	dColor(m, &b.Color, 62, off)
	if m.Reading() {
		off = b.Color.Index == -1
		b.Color.Index = 0
	}
	x.NameRef(m, 6, &b.Ltype, "LTYPE", 5)
	plot := b.Flags&16 != 0
	if !m.Reading() {
		if !plot {
			m.Bool(290, &plot)
		}
	} else {
		plot = true
		if m.Has(290) {
			m.Bool(290, &plot)
		}
	}
	lw := lineweightToDXF(byte(b.Flags>>5) & 0x1F)
	m.Int16(370, &lw)
	x.HandleRef(m, 390, &b.PlotStyle, 5)
	switch {
	case x.d.Version.R2007:
		x.HandleRef(m, 347, &b.Material, 5)
	case !m.Reading():
		g := dNamedObject(x, "Global", "MATERIAL") // R2004 has no layer material; ODA writes Global
		m.Handle(347, &g)
	}
	if x.d.Version.R2013 {
		x.HandleRef(m, 348, &b.Unknown, 5)
	} else if !m.Reading() {
		var z uint64
		m.Handle(348, &z)
	}
	if m.Reading() {
		fl := f70&1 | (f70&2)<<1 | (f70&4)<<1
		if off {
			fl |= 2
		}
		if plot {
			fl |= 16
		}
		fl |= int16(lineweightFromDXF(lw)) << 5
		b.Flags = fl
	}
}

func dLtype(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Ltype) {
	dRecord(m, &b.TableEntry, "AcDbLinetypeTableRecord", 0)
	m.Str(3, &b.Description)
	al := int16(b.Alignment)
	m.Int16(72, &al)
	b.Alignment = byte(al)
	n := m.Count(73, len(b.Dashes))
	m.Float(40, &b.PatternLength)
	wide := x.d.Version.R2007
	var texts []string
	if m.Reading() {
		b.Dashes = make([]dwg.LtypeDash, n)
	}
	for i := range b.Dashes {
		d := &b.Dashes[i]
		m.Float(49, &d.Length)
		m.Int16(74, &d.ShapeFlag)
		if d.ShapeFlag == 0 {
			if m.Reading() {
				d.Scale = 1 // simple dashes store scale 1
			}
			continue
		}
		// text dashes keep an offset into the string area in DWG; DXF writes 75 = 0 and the text in 9
		code := d.ShapeCode
		if d.ShapeFlag&2 != 0 {
			code = 0
		}
		m.Int16(75, &code)
		x.HandleRef(m, 340, &d.Style, 5)
		m.Float(46, &d.Scale)
		m.Angle(50, &d.Rotation)
		m.Float(44, &d.Offset[0])
		m.Float(45, &d.Offset[1])
		if d.ShapeFlag&2 == 0 {
			d.ShapeCode = code
			continue
		}
		var t string
		if !m.Reading() {
			t = dLtypeText(b.StringArea, int(d.ShapeCode), wide)
		}
		m.Str(9, &t)
		texts = append(texts, t)
	}
	if m.Reading() {
		dLtypeBuildArea(b, texts, wide)
	}
}

// dLtypeBuildArea lays the dash texts out in the string area (one after another, 0-terminated)
// and points the text dashes' ShapeCode at them. R2004 always has the 256-byte area;
// R2007+ has a 512-byte UTF-16 area only when a dash has text.
func dLtypeBuildArea(b *dwg.Ltype, texts []string, wide bool) {
	size := 256
	if wide {
		if len(texts) == 0 {
			b.StringArea = nil
			return
		}
		size = 512
	}
	area := make([]byte, size)
	off, k := 0, 0
	for i := range b.Dashes {
		d := &b.Dashes[i]
		if d.ShapeFlag&2 == 0 || k >= len(texts) {
			continue
		}
		d.ShapeCode = int16(off)
		if wide {
			for _, r := range texts[k] {
				if off+2 <= size {
					area[off], area[off+1] = byte(r), byte(r>>8)
				}
				off += 2
			}
			off += 2
		} else {
			off += copy(area[off:], texts[k]) + 1
		}
		k++
	}
	b.StringArea = area
}

// dLtypeText reads the 0-terminated string at offset in the linetype string area
// (UTF-16 in R2007+, 8-bit before).
func dLtypeText(area []byte, off int, wide bool) string {
	if off < 0 || off >= len(area) {
		return ""
	}
	if !wide {
		e := off
		for e < len(area) && area[e] != 0 {
			e++
		}
		return string(area[off:e])
	}
	var r []rune
	for i := off; i+1 < len(area); i += 2 {
		c := rune(area[i]) | rune(area[i+1])<<8
		if c == 0 {
			break
		}
		r = append(r, c)
	}
	return string(r)
}

func dStyle(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Style) {
	var f int16
	if b.IsShapeFile {
		f |= 1
	}
	if b.Vertical {
		f |= 4
	}
	f = dRecord(m, &b.TableEntry, "AcDbTextStyleTableRecord", f)
	if m.Reading() {
		b.IsShapeFile = f&1 != 0
		b.Vertical = f&4 != 0
	}
	m.Float(40, &b.FixedHeight)
	m.Float(41, &b.WidthFactor)
	m.Angle(50, &b.Oblique)
	g := int16(b.Generation)
	m.Int16(71, &g)
	b.Generation = byte(g)
	m.Float(42, &b.LastHeight)
	m.Str(3, &b.FontName)
	m.Str(4, &b.BigFont)
}

func dBlockRecord(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.BlockHeader) {
	m.Subclass("AcDbSymbolTableRecord")
	m.Subclass("AcDbBlockTableRecord")
	name := b.Name
	if !m.Reading() {
		name = dAnonName(x, o, name)
	}
	m.Str(2, &name)
	if m.Reading() {
		b.Name = name
	}
	x.HandleRef(m, 340, &b.Layout, 5)
	if !m.Reading() {
		if len(b.Preview) > 0 {
			m.Binary(310, &b.Preview)
		}
	} else if m.Has(310) {
		m.Binary(310, &b.Preview)
	}
	dBlkRefs(m, x, b)
	units := b.InsertUnits
	expl := b.Explodable
	scaling := int16(b.Scaling)
	if !x.d.Version.R2007 && !m.Reading() {
		// R2004 has no such fields; ODA takes the units from the "DesignCenter Data" EED.
		units, expl, scaling = dDesignCenterUnits(x, o), true, 0
	}
	m.Int16(70, &units)
	m.Bool(280, &expl)
	m.Int16(281, &scaling)
	if m.Reading() && x.d.Version.R2007 {
		b.InsertUnits, b.Explodable, b.Scaling = units, expl, byte(scaling)
	}
}

// dBlkRefs binds the {BLKREFS 331 list}: existing references only, sorted, as ODA writes them.
func dBlkRefs(m *dxf.Mapper, x *ctx, b *dwg.BlockHeader) {
	if m.Reading() {
		if !m.Peek(102) || !strings.EqualFold(m.Rest()[0].Value, "{BLKREFS") {
			return
		}
		m.Seek(m.Pos() + 1)
		for m.Peek(331) {
			b.Inserts = append(b.Inserts, dwg.Ref{Code: 4, H: m.Rest()[0].Handle()})
			m.Seek(m.Pos() + 1)
		}
		m.Seek(m.Pos() + 1)
		return
	}
	var hs []uint64
	for _, r := range b.Inserts {
		if x.d.ByHandle[r.H] != nil {
			hs = append(hs, r.H)
		}
	}
	if len(hs) == 0 {
		return
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i] < hs[j] })
	w := m.Writer()
	w.Str(102, "{BLKREFS")
	for _, h := range hs {
		w.Handle(331, h)
	}
	w.Str(102, "}")
}

// dDesignCenterUnits reads the insert units from the R2004 "ACAD" EED
// {1000 "DesignCenter Data", 1002 "{", 1070 version, 1070 units, 1002 "}"}.
func dDesignCenterUnits(x *ctx, o *dwg.Object) int16 {
	for _, e := range o.EED {
		if !strings.EqualFold(x.names[e.App.H], "ACAD") {
			continue
		}
		gs, ok := eedToGroups(e.Data, false, x.d.Text)
		if !ok || len(gs) < 4 || gs[0].Code != 1000 || gs[0].Value != "DesignCenter Data" {
			continue
		}
		if gs[3].Code == 1070 {
			return int16(gs[3].Int())
		}
	}
	return 0
}

// dViewMode converts the stored 4-bit view mode (see dwg.View.ViewMode) to/from DXF 71.
func dViewModeToDXF(v byte, ucsFollow bool) int16 {
	var f int16
	if v&8 != 0 {
		f |= 1
	}
	if v&4 != 0 {
		f |= 2
	}
	if v&2 != 0 {
		f |= 4
	}
	if ucsFollow {
		f |= 8
	}
	if v&1 == 0 {
		f |= 16
	}
	return f
}

func dViewModeFromDXF(f int16) byte {
	var v byte
	if f&1 != 0 {
		v |= 8
	}
	if f&2 != 0 {
		v |= 4
	}
	if f&4 != 0 {
		v |= 2
	}
	if f&16 == 0 {
		v |= 1
	}
	return v
}

func dVPort(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.VPort) {
	dRecord(m, &b.TableEntry, "AcDbViewportTableRecord", 0)
	m.Point2(10, &b.LowerLeft)
	m.Point2(11, &b.UpperRight)
	m.Point2(12, &b.Center)
	m.Point2(13, &b.SnapBase)
	m.Point2(14, &b.SnapSpacing)
	m.Point2(15, &b.GridSpacing)
	m.Point3(16, &b.Direction)
	m.Point3(17, &b.Target)
	m.Float(40, &b.Height)
	ar := 0.0
	if b.Height != 0 {
		ar = b.AspectTimesH / b.Height
	}
	m.Float(41, &ar)
	m.Float(42, &b.LensLength)
	m.Float(43, &b.FrontClip)
	m.Float(44, &b.BackClip)
	m.Angle(50, &b.SnapRot)
	m.Angle(51, &b.Twist)
	vm := dViewModeToDXF(b.ViewMode, b.UCSFollow)
	m.Int16(71, &vm)
	m.Int16(72, &b.CircleZoom)
	m.Bool(73, &b.FastZoom)
	icon := int16(b.UCSIcon>>1 | b.UCSIcon&1<<1) // stored BB: first bit is 74's bit 0
	m.Int16(74, &icon)
	m.Bool(75, &b.SnapOn)
	m.Bool(76, &b.GridOn)
	m.Bool(77, &b.SnapStyle)
	m.Int16(78, &b.SnapIsoPair)
	rm := int16(b.RenderMode)
	m.Int16(281, &rm)
	m.Bool(65, &b.UCSPerVP)
	m.Point3(110, &b.UCSOrigin)
	m.Point3(111, &b.UCSXAxis)
	m.Point3(112, &b.UCSYAxis)
	m.Int16(79, &b.OrthoType)
	m.Float(146, &b.UCSElev)
	x.HandleRefOpt(m, 345, &b.NamedUCS, 5)
	x.HandleRefOpt(m, 346, &b.BaseUCS, 5)
	dVPortLighting(m, x, b)
	if m.Reading() {
		b.AspectTimesH = ar * b.Height
		b.ViewMode = dViewModeFromDXF(vm)
		b.UCSFollow = vm&8 != 0
		b.UCSIcon = byte(icon>>1&1 | icon&1<<1)
		b.RenderMode = byte(rm)
	}
}

// dVPortLighting binds 348/60/61 and the default lighting groups. R2004 files have no such
// fields; ODA still writes defaults for them, which are reproduced on export and ignored on import.
func dVPortLighting(m *dxf.Mapper, x *ctx, b *dwg.VPort) {
	if !x.d.Version.R2007 {
		if m.Reading() {
			return
		}
		vs := dVisualStyle(x)
		m.Handle(348, &vs)
		gf, gm := int16(3), int16(5)
		m.Int16(60, &gf)
		m.Int16(61, &gm)
		l := dwg.Lighting{UseDefaultLights: true, LightingType: 1, Ambient: dwg.CmColor{Index: 256}}
		dLightingFields(m, &l)
		return
	}
	x.HandleRef(m, 348, &b.VisualStyle, 5)
	m.Int16(60, &b.GridFlags)
	m.Int16(61, &b.GridMajor)
	x.HandleRefOpt(m, 332, &b.Background, 5)
	x.HandleRefOpt(m, 361, &b.Sun, 3)
	dLightingFields(m, &b.Lighting)
}

func dLightingFields(m *dxf.Mapper, l *dwg.Lighting) {
	m.Bool(292, &l.UseDefaultLights)
	lt := int16(l.LightingType)
	m.Int16(282, &lt)
	l.LightingType = byte(lt)
	m.Float(141, &l.Brightness)
	m.Float(142, &l.Contrast)
	dColor(m, &l.Ambient, 63, false)
}

// dNamedObject finds an object of the given type through a dictionary entry name
// (R2004 drawings lack the R2007+ handles ODA writes, e.g. the "2dWireframe" visual style
// for 348 and the "Global" material for 347).
func dNamedObject(x *ctx, name, typ string) uint64 {
	for _, o := range x.d.Objects {
		dict, ok := o.Body.(*dwg.Dictionary)
		if !ok {
			continue
		}
		for i, n := range dict.Names {
			if !strings.EqualFold(n, name) || i >= len(dict.Items) {
				continue
			}
			if t := x.d.ByHandle[dict.Items[i].H]; t != nil && t.TypeName(x.d.Classes.Classes) == typ {
				return t.Handle
			}
		}
	}
	return 0
}

func dVisualStyle(x *ctx) uint64 { return dNamedObject(x, "2dWireframe", "VISUALSTYLE") }

func dView(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.View) {
	var f int16
	if b.PaperSpace {
		f |= 1
	}
	f = dRecord(m, &b.TableEntry, "AcDbViewTableRecord", f)
	if m.Reading() {
		b.PaperSpace = f&1 != 0
	}
	m.Float(40, &b.Height)
	m.Point2(10, &b.Center)
	m.Float(41, &b.Width)
	m.Point3(11, &b.Direction)
	m.Point3(12, &b.Target)
	m.Float(42, &b.LensLength)
	m.Float(43, &b.FrontClip)
	m.Float(44, &b.BackClip)
	m.Angle(50, &b.Twist)
	vm := dViewModeToDXF(b.ViewMode, false)
	m.Int16(71, &vm)
	rm := int16(b.RenderMode)
	m.Int16(281, &rm)
	m.Bool(72, &b.AssociatedUCS)
	if x.d.Version.R2007 {
		m.Bool(73, &b.CameraPlotable)
		x.HandleRefOpt(m, 332, &b.Background, 5)
		x.HandleRefOpt(m, 334, &b.LiveSection, 4)
		x.HandleRefOpt(m, 348, &b.VisualStyle, 5)
		x.HandleRefOpt(m, 361, &b.Sun, 3)
	} else if !m.Reading() {
		// R2004 has none of these; ODA writes the defaults
		f := false
		m.Bool(73, &f)
		vs := dVisualStyle(x)
		m.HandleOpt(348, &vs)
	}
	if b.AssociatedUCS {
		m.Point3(110, &b.UCSOrigin)
		m.Point3(111, &b.UCSXAxis)
		m.Point3(112, &b.UCSYAxis)
		m.Int16(79, &b.OrthoType)
		m.Float(146, &b.UCSElevation)
		x.HandleRefOpt(m, 345, &b.NamedUCS, 5)
		x.HandleRefOpt(m, 346, &b.BaseUCS, 5)
	}
	if m.Reading() {
		b.ViewMode = dViewModeFromDXF(vm)
		b.RenderMode = byte(rm)
	}
}

func dUCS(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.UCS) {
	dRecord(m, &b.TableEntry, "AcDbUCSTableRecord", 0)
	m.Point3(10, &b.Origin)
	m.Point3(11, &b.XAxis)
	m.Point3(12, &b.YAxis)
	m.Int16(79, &b.OrthoView)
	m.Float(146, &b.Elevation)
	x.HandleRefOpt(m, 346, &b.BaseUCS, 5)
	m.Int16Def(71, &b.OrthoType, 0)
}

// dAnonName returns the DXF name ODA writes for a block record: a "*" name without a number
// (e.g. "*X", "*D", an extra "*Paper_Space") gets the record's position among the live
// BLOCK_CONTROL entries appended ("*X469", "*Paper_Space0"). The real model and paper space
// records are not entries, so they keep their names.
func dAnonName(x *ctx, o *dwg.Object, name string) string {
	if len(name) < 2 || name[0] != '*' {
		return name
	}
	for _, r := range name[1:] {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r == '_') {
			return name
		}
	}
	ctl := x.d.ByHandle[o.Owner.H]
	if ctl == nil {
		return name
	}
	tc, ok := ctl.Body.(*dwg.TableControl)
	if !ok {
		return name
	}
	i := 0
	for _, e := range tc.Entries {
		if e.H == o.Handle {
			return name + strconv.Itoa(i)
		}
		if x.d.ByHandle[e.H] != nil {
			i++
		}
	}
	return name
}
