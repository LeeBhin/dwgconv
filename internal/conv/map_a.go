package conv

import (
	"bytes"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

func init() {
	register("ATTRIB", 0x02, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Attrib) {
		textFields(m, x, &b.TextCommon, "AcDbText")
		m.Subclass("AcDbAttribute")
		attribFields(m, x, o, &b.AttribCommon)
	})
	register("ATTDEF", 0x03, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Attdef) {
		textFields(m, x, &b.TextCommon, "AcDbText")
		m.Subclass("AcDbAttributeDefinition")
		m.Str(3, &b.Prompt)
		attribFields(m, x, o, &b.AttribCommon)
	})
	register("BLOCK", 0x04, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Block) {
		m.Subclass("AcDbBlockBegin")
		hdr := &dwg.BlockHeader{}
		if h, ok := x.d.ByHandle[o.Owner.H]; ok {
			if bh, ok := h.Body.(*dwg.BlockHeader); ok {
				hdr = bh
			}
		}
		name := b.Name
		if name == "" {
			name = hdr.Name
		}
		if m.Reading() {
			name = ""
		}
		m.Str(2, &name)
		flags := blockFlags(x, hdr)
		m.Int16(70, &flags)
		base := hdr.BasePoint
		m.Point3(10, &base)
		m.Str(3, &name)
		path := hdr.XrefPath
		m.Str(1, &path)
		if m.Reading() {
			b.Name = name
			// The BLOCK_RECORD owner carries these values in DWG.
			hdr.Name, hdr.BasePoint, hdr.XrefPath = name, base, path
			hdr.Anonymous = flags&1 != 0
			hdr.HasAttribs = flags&2 != 0
			hdr.IsXref = flags&4 != 0
			hdr.XrefOverlaid = flags&8 != 0
			hdr.Xdep = flags&16 != 0
			hdr.Loaded = flags&32 != 0
		}
	})
	register("ENDBLK", 0x05, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.EndBlk) {
		m.Subclass("AcDbBlockEnd")
	})
	register("SEQEND", 0x06, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.SeqEnd) {})
	register("INSERT", 0x07, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Insert) {
		m.Subclass("AcDbBlockReference")
		insertFields(m, x, &b.InsertCommon)
		m.Extrusion(&b.Extrusion)
	})
	mi := register("INSERT", 0x08, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.MInsert) {
		m.Subclass("AcDbMInsertBlock") // unverified: no MINSERT in the samples
		insertFields(m, x, &b.InsertCommon)
		m.Int16Def(70, &b.NumCols, 1)
		m.Int16Def(71, &b.NumRows, 1)
		m.FloatDef(44, &b.ColSpacing, 0)
		m.FloatDef(45, &b.RowSpacing, 0)
		m.Extrusion(&b.Extrusion)
	})
	mi.match = func(r *dxf.Record) bool {
		for _, g := range r.Groups {
			if g.Code == 100 && g.Value == "AcDbMInsertBlock" {
				return true
			}
		}
		return false
	}
	register("MTEXT", 0x2C, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.MText) {
		mtextFields(m, x, o, b)
	})
	register("SHAPE", 0x21, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Shape) {
		// unverified: no SHAPE in the samples. DXF 2 is the shape name; it needs the shape file
		// contents to resolve, so the number is not round-tripped here.
		m.Subclass("AcDbShape")
		m.FloatDef(39, &b.Thickness, 0)
		m.Point3(10, &b.Insertion)
		m.Float(40, &b.Scale)
		m.AngleDef(50, &b.Rotation, 0)
		m.FloatDef(41, &b.WidthFact, 1)
		m.AngleDef(51, &b.Oblique, 0)
		m.Extrusion(&b.Extrusion)
	})
	register("TOLERANCE", 0x2E, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Tolerance) {
		// unverified: no TOLERANCE in the samples
		m.Subclass("AcDbFcf")
		x.NameRef(m, 3, &b.DimStyle, "DIMSTYLE", 5)
		m.Point3(10, &b.Insertion)
		m.Str(1, &b.Value)
		m.Extrusion(&b.Extrusion)
		m.Point3(11, &b.XDir)
	})
	register("LEADER", 0x2D, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Leader) {
		leaderFields(m, x, b)
	})
}

// attribFields binds the AcDbAttribute / AcDbAttributeDefinition tail shared by ATTRIB and ATTDEF.
func attribFields(m *dxf.Mapper, x *ctx, o *dwg.Object, a *dwg.AttribCommon) {
	m.Str(2, &a.Tag)
	m.Byte(70, &a.Flags)
	m.Int16Def(73, &a.FieldLength, 0)
	m.Int16Def(74, &a.VAlign, 0)
	lock := a.LockPos
	if !x.d.Version.R2007 && !m.Reading() {
		lock = attrLockFromEED(x, o)
	}
	m.Bool(280, &lock)
	typ := int16(a.AttType)
	if typ == 0 {
		typ = 1
	}
	m.Int16(71, &typ)
	var unk int16
	m.Int16(72, &unk)
	al := [3]float64{a.Alignment[0], a.Alignment[1], a.Elevation}
	m.Point3(11, &al)
	if !m.Reading() {
		return
	}
	if x.d.Version.R2018 {
		a.AttType = byte(typ)
	}
	if x.d.Version.R2007 {
		a.LockPos = lock
	} else {
		attrLockToEED(x, o, lock)
	}
	a.Alignment = [2]float64{al[0], al[1]}
	a.DataFlags = textDataFlags(&a.TextCommon)
}

// R2004 files keep the lock position in "AcDbAttr" xdata (1070 0, 1070 lock); ODA folds it into 280.
func attrLockFromEED(x *ctx, o *dwg.Object) bool {
	app := x.byName["APPID"]["ACDBATTR"]
	for _, e := range o.EED {
		if app != 0 && e.App.H == app && len(e.Data) >= 6 && e.Data[3] == 0x46 {
			return e.Data[4] != 0 || e.Data[5] != 0
		}
	}
	return false
}

func attrLockToEED(x *ctx, o *dwg.Object, lock bool) {
	app := x.byName["APPID"]["ACDBATTR"]
	if app == 0 || !lock {
		return
	}
	data := []byte{0x46, 0, 0, 0x46, 1, 0}
	for _, e := range o.EED {
		if e.App.H == app && bytes.Equal(e.Data, data) {
			return
		}
	}
	o.EED = append(o.EED, dwg.EED{App: dwg.Ref{Code: 5, H: app}, Data: data})
}

// blockFlags builds BLOCK group 70. Bit 2 follows the block's actual non-constant ATTDEFs
// (as ODA writes it), not the stored BLOCK_RECORD flag.
func blockFlags(x *ctx, h *dwg.BlockHeader) int16 {
	attribs := false
	for _, e := range h.Entities {
		if eo := x.d.ByHandle[e.H]; eo != nil {
			if ad, ok := eo.Body.(*dwg.Attdef); ok && ad.Flags&2 == 0 {
				attribs = true
				break
			}
		}
	}
	var f int16
	for i, b := range []bool{h.Anonymous, attribs, h.IsXref, h.XrefOverlaid, h.Xdep, h.Loaded} {
		if b {
			f |= 1 << i
		}
	}
	return f
}

func insertFields(m *dxf.Mapper, x *ctx, b *dwg.InsertCommon) {
	if !m.Reading() {
		if b.HasAttribs {
			m.Bool(66, &b.HasAttribs)
		}
	} else if m.Has(66) {
		m.Bool(66, &b.HasAttribs)
	}
	x.NameRef(m, 2, &b.BlockHdr, "BLOCK_RECORD", 5)
	m.Point3(10, &b.Insertion)
	m.FloatDef(41, &b.Scale[0], 1)
	m.FloatDef(42, &b.Scale[1], 1)
	m.FloatDef(43, &b.Scale[2], 1)
	m.AngleDef(50, &b.Rotation, 0)
}

// mtextChunk is the length of the 3-group pieces ODA splits long MTEXT strings into.
const mtextChunk = 250

func mtextFields(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.MText) {
	r2007 := x.d.Version.R2007
	m.Subclass("AcDbMText")
	m.Point3(10, &b.Insertion)
	m.Float(40, &b.TextHeight)
	m.Float(41, &b.RectWidth)
	// Before R2007 the defined height lives in ACAD xdata; ODA writes it as 46.
	rh := b.RectHeight
	if !r2007 && !m.Reading() {
		if g := acadBlock(x, o, "ACAD_MTEXT_DEFINED_HEIGHT"); len(g) >= 2 {
			rh = g[1].Float()
		}
	}
	m.Float(46, &rh)
	m.Int16(71, &b.Attachment)
	m.Int16(72, &b.DrawingDir)
	mtextValue(m, &b.Value)
	x.NameRefDef(m, 7, &b.Style, "STYLE", 5, "Standard")
	m.Extrusion(&b.Extrusion)
	m.Point3Def(11, &b.XAxis, [3]float64{1, 0, 0})
	m.Int16(73, &b.LineSpStyle)
	m.Float(44, &b.LineSpFactor)
	if !m.Reading() {
		if b.BgFlags != 0 {
			bgFields(m, b)
		}
	} else if m.Has(90) {
		bgFields(m, b)
	}
	cols := mtextColumns(x, o, b)
	if !m.Reading() {
		if cols != nil {
			embeddedColumns(m, b, cols)
		}
	} else if m.Has(101) {
		cols = &mtextCols{}
		embeddedColumns(m, b, cols)
	}
	if !m.Reading() {
		return
	}
	if r2007 {
		b.RectHeight = rh
	} else if rh != 0 {
		addACADBlock(x, o, "ACAD_MTEXT_DEFINED_HEIGHT", []dxf.Group{g70(46), g40(rh)})
	}
	if cols != nil {
		if x.d.Version.R2018 {
			b.ColType, b.ColCount, b.ColWidth, b.ColGutter = cols.typ, cols.count, cols.width, cols.gutter
			b.ColAutoHeight, b.ColFlowRev, b.ColHeights = cols.auto, cols.flow, cols.heights
		} else {
			gs := []dxf.Group{g70(75), g70(int(cols.typ)), g70(79), g70(b2i(cols.auto)), g70(76), g70(int(cols.count)),
				g70(78), g70(b2i(cols.flow)), g70(48), g40(cols.width), g70(49), g40(cols.gutter)}
			if len(cols.heights) > 0 {
				gs = append(gs, g70(50), g70(len(cols.heights)))
				for _, h := range cols.heights {
					gs = append(gs, g40(h))
				}
			}
			addACADBlock(x, o, "ACAD_MTEXT_COLUMN_INFO", gs)
		}
	}
	b.NotAnnotative, b.AVersion, b.ADefault = true, 4, true
	b.AAttachment = int32(b.Attachment)
	b.AXAxis, b.AInsertion = b.XAxis, b.Insertion
	b.ARectWidth, b.ARectHeight = b.RectWidth, b.RectHeight
}

type mtextCols struct {
	typ        int16
	count      int32
	width      float64
	gutter     float64
	auto, flow bool
	heights    []float64
}

// mtextColumns returns the column data: native fields (R2018) or ACAD_MTEXT_COLUMN_INFO xdata (older).
func mtextColumns(x *ctx, o *dwg.Object, b *dwg.MText) *mtextCols {
	if x.d.Version.R2018 {
		if b.ColType == 0 {
			return nil
		}
		return &mtextCols{b.ColType, b.ColCount, b.ColWidth, b.ColGutter, b.ColAutoHeight, b.ColFlowRev, b.ColHeights}
	}
	g := acadBlock(x, o, "ACAD_MTEXT_COLUMN_INFO")
	if g == nil {
		return nil
	}
	c := &mtextCols{}
	for i := 0; i+1 < len(g); i += 2 {
		switch g[i].Int() {
		case 75:
			c.typ = int16(g[i+1].Int())
		case 79:
			c.auto = g[i+1].Int() != 0
		case 76:
			c.count = int32(g[i+1].Int())
		case 78:
			c.flow = g[i+1].Int() != 0
		case 48:
			c.width = g[i+1].Float()
		case 49:
			c.gutter = g[i+1].Float()
		case 50:
			n := int(g[i+1].Int())
			for k := 0; k < n && i+2+k < len(g); k++ {
				c.heights = append(c.heights, g[i+2+k].Float())
			}
			i += n
		}
	}
	return c
}

// embeddedColumns binds the "101 Embedded Object" block ODA writes for MTEXT columns.
func embeddedColumns(m *dxf.Mapper, b *dwg.MText, c *mtextCols) {
	s := "Embedded Object"
	m.Str(101, &s)
	one := int16(1)
	m.Int16(70, &one)
	xa, ins := b.XAxis, b.Insertion
	m.Point3(10, &xa)
	m.Point3(11, &ins)
	rw, rh, ew, eh := b.RectWidth, b.RectHeight, b.ExtWidth, b.ExtHeight
	m.Float(40, &rw)
	m.Float(41, &rh)
	m.Float(42, &ew)
	m.Float(43, &eh)
	m.Int16(71, &c.typ)
	m.Int32(72, &c.count)
	m.Float(44, &c.width)
	m.Float(45, &c.gutter)
	m.Bool(73, &c.auto)
	m.Bool(74, &c.flow)
	if !m.Reading() {
		for i := range c.heights {
			m.Float(46, &c.heights[i])
		}
		return
	}
	for m.Has(46) {
		var h float64
		m.Float(46, &h)
		c.heights = append(c.heights, h)
	}
	b.ExtWidth, b.ExtHeight = ew, eh
}

// acadBlock returns the groups between "<name>_BEGIN" and "<name>_END" in the object's ACAD xdata.
func acadBlock(x *ctx, o *dwg.Object, name string) []dxf.Group {
	app := x.byName["APPID"]["ACAD"]
	for _, e := range o.EED {
		if e.App.H != app {
			continue
		}
		gs, ok := eedToGroups(e.Data, x.d.Version.R2007, x.d.Text)
		if !ok {
			continue
		}
		for i, g := range gs {
			if g.Code != 1000 || g.Value != name+"_BEGIN" {
				continue
			}
			for j := i + 1; j < len(gs); j++ {
				if gs[j].Code == 1000 && gs[j].Value == name+"_END" {
					return gs[i+1 : j]
				}
			}
		}
	}
	return nil
}

// addACADBlock appends a BEGIN/END block to the object's ACAD xdata (import of ODA-folded values).
func addACADBlock(x *ctx, o *dwg.Object, name string, inner []dxf.Group) {
	app := x.byName["APPID"]["ACAD"]
	if app == 0 {
		return
	}
	gs := append([]dxf.Group{{Code: 1000, Value: name + "_BEGIN"}}, inner...)
	gs = append(gs, dxf.Group{Code: 1000, Value: name + "_END"})
	data := groupsToEED(gs, x.d.Version.R2007, x.d.Text)
	for i, e := range o.EED {
		if e.App.H == app {
			o.EED[i].Data = append(append([]byte(nil), e.Data...), data...)
			return
		}
	}
	o.EED = append(o.EED, dwg.EED{App: dwg.Ref{Code: 5, H: app}, Data: data})
}

func g70(v int) dxf.Group     { return dxf.Group{Code: 1070, Value: strconv.Itoa(v)} }
func g40(v float64) dxf.Group { return dxf.Group{Code: 1040, Value: dxf.FormatFloat(v)} }

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func bgFields(m *dxf.Mapper, b *dwg.MText) {
	m.Int32(90, &b.BgFlags)
	if b.BgFlags&1 != 0 {
		m.Int16(63, &b.BgColor.Index)
		m.Float(45, &b.BgScale)
		m.Int32(441, &b.BgTransp)
	}
}

// mtextValue splits the text into 250-character 3 groups ending with a 1 group.
func mtextValue(m *dxf.Mapper, v *string) {
	if !m.Reading() {
		s := *v
		for utf8.RuneCountInString(s) > mtextChunk {
			i := 0
			for n := 0; n < mtextChunk; n++ {
				_, sz := utf8.DecodeRuneInString(s[i:])
				i += sz
			}
			piece := s[:i]
			m.Str(3, &piece)
			s = s[i:]
		}
		m.Str(1, &s)
		return
	}
	var sb strings.Builder
	for m.Peek(3) {
		var p string
		m.Str(3, &p)
		sb.WriteString(p)
	}
	var last string
	m.Str(1, &last)
	sb.WriteString(last)
	*v = sb.String()
}

func leaderFields(m *dxf.Mapper, x *ctx, b *dwg.Leader) {
	m.Subclass("AcDbLeader")
	x.NameRef(m, 3, &b.DimStyle, "DIMSTYLE", 5)
	m.Bool(71, &b.ArrowheadOn)
	m.Int16Def(72, &b.PathType, 0)
	// ODA omits 73 for annotation type 2 (all sample leaders; the DXF reference default is 3).
	m.Int16Def(73, &b.AnnotType, 2)
	hook := int16(0)
	if b.HookXDir {
		hook = 1
	}
	m.Int16Def(74, &hook, 0)
	m.FloatDef(40, &b.BoxHeight, 0)
	m.FloatDef(41, &b.BoxWidth, 0)
	n := m.Count(76, len(b.Points))
	if m.Reading() {
		b.Points = make([][3]float64, n)
	}
	for i := range b.Points {
		m.Point3(10, &b.Points[i])
	}
	x.HandleRefOpt(m, 340, &b.Annotation, 5)
	m.Extrusion(&b.Extrusion)
	m.Point3(211, &b.XDir)
	m.Point3(212, &b.OffsetBlock)
	m.Point3Def(213, &b.EndPtProj, [3]float64{})
	if m.Reading() {
		b.HookXDir = hook != 0
		if len(b.Points) > 0 {
			b.Origin = b.Points[0] // not in DXF; the first point is the documented default
		}
	}
}
