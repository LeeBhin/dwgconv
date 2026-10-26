package dwg

func init() {
	bodyFactory[0x13] = func() Body { return &Line{} }
}

// Line §20.4.21.
type Line struct {
	ZZero      bool
	Start, End [3]float64
	Thickness  float64
	Extrusion  [3]float64
}

func (b *Line) data(c *codec, o *Object) {
	if c.w {
		b.ZZero = b.Start[2] == 0 && b.End[2] == 0
	}
	c.B(&b.ZZero)
	c.RD(&b.Start[0])
	c.DD(&b.End[0], b.Start[0])
	c.RD(&b.Start[1])
	c.DD(&b.End[1], b.Start[1])
	if !b.ZZero {
		c.RD(&b.Start[2])
		c.DD(&b.End[2], b.Start[2])
	}
	c.BT(&b.Thickness)
	c.BE(&b.Extrusion)
}

func (b *Line) handles(c *codec, o *Object) {}

func init() {
	bodyFactory[0x01] = func() Body { return &Text{} }
}

// TextCommon is the common TEXT entity data shared by TEXT, ATTRIB and ATTDEF (§20.4.3, R2000+ form).
type TextCommon struct {
	DataFlags  byte
	Elevation  float64
	Insertion  [2]float64
	Alignment  [2]float64
	Extrusion  [3]float64
	Thickness  float64
	Oblique    float64
	Rotation   float64
	Height     float64
	WidthFact  float64
	Value      string
	Generation int16
	HAlign     int16
	VAlign     int16
	Style      Ref
}

func (t *TextCommon) data(c *codec) {
	c.RC(&t.DataFlags)
	f := t.DataFlags
	if f&0x01 == 0 {
		c.RD(&t.Elevation)
	}
	c.RD2(&t.Insertion)
	if f&0x02 == 0 {
		c.DD2(&t.Alignment, t.Insertion)
	}
	c.BE(&t.Extrusion)
	c.BT(&t.Thickness)
	if f&0x04 == 0 {
		c.RD(&t.Oblique)
	}
	if f&0x08 == 0 {
		c.RD(&t.Rotation)
	}
	c.RD(&t.Height)
	if f&0x10 == 0 {
		c.RD(&t.WidthFact)
	} else if !c.w {
		t.WidthFact = 1
	}
	c.TV(&t.Value)
	if f&0x20 == 0 {
		c.BS(&t.Generation)
	}
	if f&0x40 == 0 {
		c.BS(&t.HAlign)
	}
	if f&0x80 == 0 {
		c.BS(&t.VAlign)
	}
}

// Text §20.4.3.
type Text struct{ TextCommon }

func (b *Text) data(c *codec, o *Object)    { b.TextCommon.data(c) }
func (b *Text) handles(c *codec, o *Object) { c.H(&b.Style) }
