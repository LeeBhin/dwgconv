package dwg

func init() {
	bodyFactory[0x02] = func() Body { return &Attrib{} }
	bodyFactory[0x03] = func() Body { return &Attdef{} }
	bodyFactory[0x04] = func() Body { return &Block{} }
	bodyFactory[0x05] = func() Body { return &EndBlk{} }
	bodyFactory[0x06] = func() Body { return &SeqEnd{} }
	bodyFactory[0x07] = func() Body { return &Insert{} }
	bodyFactory[0x08] = func() Body { return &MInsert{} }
}

// AttribCommon holds the ATTRIB-specific fields after the common TEXT data (§20.4.4).
type AttribCommon struct {
	TextCommon
	Version     byte // R2010+
	AttType     byte // R2018+: 1 single line, 2 multi line (ATTRIB), 4 multi line (ATTDEF)
	Tag         string
	FieldLength int16
	Flags       byte
	LockPos     bool // R2007+
	MText       *aEmbeddedMText
}

func (a *AttribCommon) data(c *codec, o *Object) {
	a.TextCommon.data(c)
	if c.R2010 {
		c.RC(&a.Version)
	}
	if c.R2018 {
		c.RC(&a.AttType)
	}
	if c.R2018 && a.AttType > 1 {
		if !c.w {
			a.MText = &aEmbeddedMText{}
		}
		a.MText.data(c, o)
		c.TV(&a.Tag)
		var unk int16
		c.BS(&unk)
		c.RC(&a.Flags)
		c.B(&a.LockPos)
		return
	}
	c.TV(&a.Tag)
	c.BS(&a.FieldLength)
	c.RC(&a.Flags)
	if c.R2007 {
		c.B(&a.LockPos)
	}
}

func (a *AttribCommon) handles(c *codec, o *Object) {
	c.H(&a.Style)
	if a.MText != nil {
		a.MText.handles(c, o)
	}
}

// aEmbeddedMText is the MTEXT embedded in a multi-line R2018 ATTRIB/ATTDEF (§20.4.4).
// unverified: no multi-line attributes in the samples.
type aEmbeddedMText struct {
	MText
	AnnotSize int16
	AnnotData []byte
	RegApp    Ref
	AnnotUnk  int16
}

func (m *aEmbeddedMText) data(c *codec, o *Object) {
	m.MText.data(c, o)
	n := c.BSCount(len(m.AnnotData))
	if n > 0 {
		c.Raw(&m.AnnotData, n)
		c.BS(&m.AnnotUnk)
	}
}

func (m *aEmbeddedMText) handles(c *codec, o *Object) {
	m.MText.handles(c, o)
	if len(m.AnnotData) > 0 {
		c.H(&m.RegApp)
	}
}

// Attrib §20.4.4.
type Attrib struct{ AttribCommon }

func (b *Attrib) data(c *codec, o *Object)    { b.AttribCommon.data(c, o) }
func (b *Attrib) handles(c *codec, o *Object) { b.AttribCommon.handles(c, o) }

// Attdef §20.4.5.
type Attdef struct {
	AttribCommon
	PromptVersion byte // R2010+
	Prompt        string
}

func (b *Attdef) data(c *codec, o *Object) {
	b.AttribCommon.data(c, o)
	if c.R2010 {
		c.RC(&b.PromptVersion)
	}
	c.TV(&b.Prompt)
}
func (b *Attdef) handles(c *codec, o *Object) { b.AttribCommon.handles(c, o) }

// Block §20.4.6.
type Block struct{ Name string }

func (b *Block) data(c *codec, o *Object)    { c.TV(&b.Name) }
func (b *Block) handles(c *codec, o *Object) {}

// EndBlk §20.4.7.
type EndBlk struct{}

func (b *EndBlk) data(c *codec, o *Object)    {}
func (b *EndBlk) handles(c *codec, o *Object) {}

// SeqEnd §20.4.8.
type SeqEnd struct{}

func (b *SeqEnd) data(c *codec, o *Object)    {}
func (b *SeqEnd) handles(c *codec, o *Object) {}

// InsertCommon is shared by INSERT and MINSERT (§20.4.9-10).
type InsertCommon struct {
	Insertion  [3]float64
	Scale      [3]float64
	Rotation   float64
	Extrusion  [3]float64
	HasAttribs bool
	BlockHdr   Ref
	Attribs    []Ref `conv:"derived"` // R2004+: owned ATTRIBs (linked by the record sequence in DXF)
	SeqEnd     Ref   `conv:"derived"`
}

func (b *InsertCommon) dataHead(c *codec) {
	c.BD3(&b.Insertion)
	var flags byte
	if c.w {
		s := b.Scale
		switch {
		case s == [3]float64{1, 1, 1}:
			flags = 3
		case s[0] == 1:
			flags = 1
		case s[1] == s[0] && s[2] == s[0]:
			flags = 2
		}
	}
	c.BB(&flags)
	switch flags {
	case 3:
		b.Scale = [3]float64{1, 1, 1}
	case 1:
		b.Scale[0] = 1
		c.DD(&b.Scale[1], 1)
		c.DD(&b.Scale[2], 1)
	case 2:
		c.RD(&b.Scale[0])
		b.Scale[1], b.Scale[2] = b.Scale[0], b.Scale[0]
	default:
		c.RD(&b.Scale[0])
		c.DD(&b.Scale[1], b.Scale[0])
		c.DD(&b.Scale[2], b.Scale[0])
	}
	c.BD(&b.Rotation)
	c.BD3(&b.Extrusion)
	if c.w {
		// A SEQEND without ATTRIBs (count 0) occurs in real files; keep it.
		b.HasAttribs = len(b.Attribs) > 0 || !b.SeqEnd.IsNull()
	}
	c.B(&b.HasAttribs)
	if b.HasAttribs {
		n := c.BLCount(len(b.Attribs))
		b.Attribs = Slice(c, b.Attribs, n)
	}
}

func (b *InsertCommon) handles(c *codec) {
	c.H(&b.BlockHdr)
	if b.HasAttribs {
		for i := range b.Attribs {
			c.H(&b.Attribs[i])
		}
		c.H(&b.SeqEnd)
	}
}

// Insert §20.4.9.
type Insert struct{ InsertCommon }

func (b *Insert) data(c *codec, o *Object)    { b.dataHead(c) }
func (b *Insert) handles(c *codec, o *Object) { b.InsertCommon.handles(c) }

// MInsert §20.4.10.
type MInsert struct {
	InsertCommon
	NumCols, NumRows       int16
	ColSpacing, RowSpacing float64
}

func (b *MInsert) data(c *codec, o *Object) {
	b.dataHead(c)
	c.BS(&b.NumCols)
	c.BS(&b.NumRows)
	c.BD(&b.ColSpacing)
	c.BD(&b.RowSpacing)
}
func (b *MInsert) handles(c *codec, o *Object) { b.InsertCommon.handles(c) }
