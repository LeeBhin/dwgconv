package dwg

import "errors"

func init() {
	classBodyFactory["ACAD_TABLE"] = func() Body { return &Table{} }
}

// cInsertScale codes the R2000+ insert scale (data flags BB + scale data, §20.4.9).
// When writing, the flags are derived from the scale.
func cInsertScale(c *codec, s *[3]float64) {
	var f byte
	if c.w {
		switch {
		case *s == [3]float64{1, 1, 1}:
			f = 3
		case s[0] == 1:
			f = 1
		case s[1] == s[0] && s[2] == s[0]:
			f = 2
		}
	}
	c.BB(&f)
	switch f {
	case 3:
		*s = [3]float64{1, 1, 1}
	case 1:
		s[0] = 1
		c.DD(&s[1], 1)
		c.DD(&s[2], 1)
	case 2:
		c.RD(&s[0])
		s[1], s[2] = s[0], s[0]
	default:
		c.RD(&s[0])
		c.DD(&s[1], s[0])
		c.DD(&s[2], s[0])
	}
}

// TableAttr is an attribute definition value of a block cell.
type TableAttr struct {
	Index  int16
	Text   string // 300
	AttDef Ref    // 331
}

// TableCell is one cell of the pre-R2010 table (§20.4.96.1).
type TableCell struct {
	Type         int16   // 171: 1 text, 2 block
	EdgeFlags    byte    // 172
	Merged       bool    // 173
	Autofit      bool    // 174
	MergedWidth  int32   // 175
	MergedHeight int32   // 176
	Rotation     float64 // 145
	Text         string  // 1
	BlockScale   float64 // 144
	HasAttrs     bool
	Attrs        []TableAttr
	HasOverride  bool
	FlagOverride int32      // 177
	VirtualEdge  byte       // 178
	Alignment    int16      // 170
	BgFillNone   bool       // 283
	BgColor      CmColor    // 63
	ContentColor CmColor    // 64
	TextStyle    Ref        // 7
	TextHeight   float64    // 140
	EdgeColor    [4]CmColor // top, right, bottom, left: 69, 65, 66, 68
	EdgeLW       [4]int16   // 279, 275, 276, 278
	EdgeVis      [4]int16   // 289, 285, 286, 288
	Value        Ref        // 344 text cell / 340 block cell
}

// Table is ACAD_TABLE in the pre-R2010 layout (§20.4.96.1). The R2010+ layout embeds
// TABLECONTENT and is not decoded (objects stay raw). Unverified: no samples contain tables.
type Table struct {
	InsPt       [3]float64 // 10
	Scale       [3]float64 // 41-43
	Rotation    float64    // 50
	Extrusion   [3]float64 // 210
	HasAttribs  bool       // 66
	Attribs     []Ref
	SeqEnd      Ref
	Flags       int16      // 90
	HorDir      [3]float64 // 11
	ColWidths   []float64  // 142
	RowHeights  []float64  // 141
	Cells       []TableCell
	BlockHeader Ref // 2
	TableStyle  Ref // 342

	HasTableOverrides bool
	TableOverride     int32 // 93
	TitleSuppressed   bool  // 280
	FlowDir           int16 // 70
	HMargin, VMargin  float64
	RowColor          [3]CmColor // title, header, data: 64
	RowFillNone       [3]bool    // 283
	RowFillColor      [3]CmColor // 63
	RowAlign          [3]int16   // 170
	RowTextStyle      [3]Ref     // 7
	RowHeight         [3]float64 // 140

	HasBorderColor   bool
	BorderColorFlags int32 // 94
	BorderColors     [18]CmColor
	HasBorderLW      bool
	BorderLWFlags    int32 // 95
	BorderLW         [18]int16
	HasBorderVis     bool
	BorderVisFlags   int32 // 96
	BorderVis        [18]int16
}

var errTableR2010 = errors.New("ACAD_TABLE: R2010+ layout (TABLECONTENT) not decoded")

// cell override field bits (§20.4.96.1)
var (
	edgeColorBit = [4]int32{0x40, 0x80, 0x100, 0x200}
	edgeLWBit    = [4]int32{0x400, 0x800, 0x1000, 0x2000}
	edgeVisBit   = [4]int32{0x4000, 0x8000, 0x10000, 0x20000}
)

func (b *Table) data(c *codec, o *Object) {
	if c.R2010 {
		panic(errTableR2010)
	}
	c.BD3(&b.InsPt)
	cInsertScale(c, &b.Scale)
	c.BD(&b.Rotation)
	c.BD3(&b.Extrusion)
	if c.w {
		b.HasAttribs = len(b.Attribs) > 0
	}
	c.B(&b.HasAttribs)
	if b.HasAttribs {
		n := c.BLCount(len(b.Attribs))
		b.Attribs = Slice(c, b.Attribs, n)
	}
	c.BS(&b.Flags)
	c.BD3(&b.HorDir)
	nc := c.BLCount(len(b.ColWidths))
	nr := c.BLCount(len(b.RowHeights))
	b.ColWidths = Slice(c, b.ColWidths, nc)
	b.RowHeights = Slice(c, b.RowHeights, nr)
	for i := range b.ColWidths {
		c.BD(&b.ColWidths[i])
	}
	for i := range b.RowHeights {
		c.BD(&b.RowHeights[i])
	}
	b.Cells = Slice(c, b.Cells, nc*nr)
	for i := range b.Cells {
		b.Cells[i].data(c)
	}
	c.B(&b.HasTableOverrides)
	if b.HasTableOverrides {
		f := &b.TableOverride
		c.BL(f)
		if *f&0x01 != 0 {
			c.B(&b.TitleSuppressed)
		}
		if *f&0x04 != 0 {
			c.BS(&b.FlowDir)
		}
		if *f&0x08 != 0 {
			c.BD(&b.HMargin)
		}
		if *f&0x10 != 0 {
			c.BD(&b.VMargin)
		}
		for i := 0; i < 3; i++ {
			if *f&(0x20<<i) != 0 {
				c.CMC(&b.RowColor[i])
			}
		}
		for i := 0; i < 3; i++ {
			if *f&(0x100<<i) != 0 {
				c.B(&b.RowFillNone[i])
			}
		}
		for i := 0; i < 3; i++ {
			if *f&(0x800<<i) != 0 {
				c.CMC(&b.RowFillColor[i])
			}
		}
		for i := 0; i < 3; i++ {
			if *f&(0x4000<<i) != 0 {
				c.BS(&b.RowAlign[i])
			}
		}
		for i := 0; i < 3; i++ {
			if *f&(0x100000<<i) != 0 {
				c.BD(&b.RowHeight[i])
			}
		}
	}
	c.B(&b.HasBorderColor)
	if b.HasBorderColor {
		c.BL(&b.BorderColorFlags)
		for i := range b.BorderColors {
			if b.BorderColorFlags&(1<<i) != 0 {
				c.CMC(&b.BorderColors[i])
			}
		}
	}
	c.B(&b.HasBorderLW)
	if b.HasBorderLW {
		c.BL(&b.BorderLWFlags)
		for i := range b.BorderLW {
			if b.BorderLWFlags&(1<<i) != 0 {
				c.BS(&b.BorderLW[i])
			}
		}
	}
	c.B(&b.HasBorderVis)
	if b.HasBorderVis {
		c.BL(&b.BorderVisFlags)
		for i := range b.BorderVis {
			if b.BorderVisFlags&(1<<i) != 0 {
				c.BS(&b.BorderVis[i])
			}
		}
	}
}

func (t *TableCell) data(c *codec) {
	c.BS(&t.Type)
	c.RC(&t.EdgeFlags)
	c.B(&t.Merged)
	c.B(&t.Autofit)
	c.BL(&t.MergedWidth)
	c.BL(&t.MergedHeight)
	c.BD(&t.Rotation)
	switch t.Type {
	case 1:
		// The spec makes the text conditional on the 344 handle being null; that handle
		// lives in the handle stream, so the text is assumed present (unverified).
		c.TV(&t.Text)
	case 2:
		c.BD(&t.BlockScale)
		if c.w {
			t.HasAttrs = len(t.Attrs) > 0
		}
		c.B(&t.HasAttrs)
		if t.HasAttrs {
			n := c.BSCount(len(t.Attrs))
			t.Attrs = Slice(c, t.Attrs, n)
			for i := range t.Attrs {
				c.BS(&t.Attrs[i].Index)
				c.TV(&t.Attrs[i].Text)
			}
		}
	}
	c.B(&t.HasOverride)
	if !t.HasOverride {
		return
	}
	f := &t.FlagOverride
	c.BL(f)
	c.RC(&t.VirtualEdge)
	if *f&0x01 != 0 {
		c.RS(&t.Alignment)
	}
	if *f&0x02 != 0 {
		c.B(&t.BgFillNone)
	}
	if *f&0x04 != 0 {
		c.CMC(&t.BgColor)
	}
	if *f&0x08 != 0 {
		c.CMC(&t.ContentColor)
	}
	if *f&0x20 != 0 {
		c.BD(&t.TextHeight)
	}
	for e := 0; e < 4; e++ {
		if *f&edgeColorBit[e] != 0 {
			c.CMC(&t.EdgeColor[e])
		}
		if *f&edgeLWBit[e] != 0 {
			c.BS(&t.EdgeLW[e])
		}
		if *f&edgeVisBit[e] != 0 {
			c.BS(&t.EdgeVis[e])
		}
	}
}

func (b *Table) handles(c *codec, o *Object) {
	c.H(&b.BlockHeader)
	for i := range b.Attribs {
		c.H(&b.Attribs[i])
	}
	if b.HasAttribs {
		c.H(&b.SeqEnd)
	}
	c.H(&b.TableStyle)
	for i := range b.Cells {
		t := &b.Cells[i]
		c.H(&t.Value)
		if t.Type == 2 {
			for j := range t.Attrs {
				c.H(&t.Attrs[j].AttDef)
			}
		}
		if t.HasOverride && t.FlagOverride&0x10 != 0 {
			c.H(&t.TextStyle)
		}
	}
	if b.HasTableOverrides {
		for i := 0; i < 3; i++ {
			if b.TableOverride&(0x20000<<i) != 0 {
				c.H(&b.RowTextStyle[i])
			}
		}
	}
}
