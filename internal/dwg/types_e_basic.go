package dwg

// Group E: non-graphic objects (§20.4).

func init() {
	bodyFactory[0x2A] = func() Body { return &Dictionary{} }
	classBodyFactory["ACDBDICTIONARYWDFLT"] = func() Body { return &DictionaryWithDefault{} }
	bodyFactory[0x4F] = func() Body { return &XRecord{} }
	classBodyFactory["XRECORD"] = func() Body { return &XRecord{} }
	classBodyFactory["DICTIONARYVAR"] = func() Body { return &DictionaryVar{} }
	classBodyFactory["IDBUFFER"] = func() Body { return &IDBuffer{} }
	classBodyFactory["SCALE"] = func() Body { return &Scale{} }
	classBodyFactory["SORTENTSTABLE"] = func() Body { return &SortEntsTable{} }
	classBodyFactory["LAYER_INDEX"] = func() Body { return &LayerIndex{} }
	bodyFactory[0x50] = func() Body { return &Placeholder{} }
	classBodyFactory["ACDBPLACEHOLDER"] = func() Body { return &Placeholder{} }
}

// eRestBits codes all remaining data-stream bits verbatim (fields whose layout is not documented).
func eRestBits(c *codec, b *[]byte, n *int) {
	if c.w {
		c.wd.AppendBits(*b, 0, *n)
		return
	}
	*n = c.dat.Remaining()
	w := bitsWriterFrom(c, *n)
	*b = w
}

// bitsWriterFrom reads n bits from the data stream into an MSB-first byte slice.
func bitsWriterFrom(c *codec, n int) []byte {
	out := make([]byte, (n+7)/8)
	for i := 0; i < n; i++ {
		if c.dat.B() == 1 {
			out[i>>3] |= 1 << (7 - uint(i&7))
		}
	}
	return out
}

// eRestHandles codes handles until the handle stream is exhausted (only padding left).
func eRestHandles(c *codec, hs *[]Ref) {
	if c.w {
		for i := range *hs {
			c.H(&(*hs)[i])
		}
		return
	}
	for c.hdl.Remaining() >= 8 {
		var r Ref
		c.H(&r)
		*hs = append(*hs, r)
	}
}

// Dictionary §20.4.44.
type Dictionary struct {
	Cloning   int16
	HardOwner byte
	Names     []string
	Items     []Ref
}

func (b *Dictionary) data(c *codec, o *Object) {
	n := c.BLCount(len(b.Names))
	c.BS(&b.Cloning)
	c.RC(&b.HardOwner)
	b.Names = Slice(c, b.Names, n)
	for i := range b.Names {
		c.TV(&b.Names[i])
	}
}

func (b *Dictionary) handles(c *codec, o *Object) {
	b.Items = Slice(c, b.Items, len(b.Names))
	for i := range b.Items {
		c.H(&b.Items[i])
	}
}

// DictionaryWithDefault §20.4.45.
type DictionaryWithDefault struct {
	Dictionary
	Default Ref
}

func (b *DictionaryWithDefault) handles(c *codec, o *Object) {
	b.Dictionary.handles(c, o)
	c.H(&b.Default)
}

// XRecord §20.4.104. Data holds the raw group-code/value byte stream.
type XRecord struct {
	Data    []byte
	Cloning int16
	ObjIDs  []Ref
}

func (b *XRecord) data(c *codec, o *Object) {
	n := c.BLCount(len(b.Data))
	c.Raw(&b.Data, n)
	c.BS(&b.Cloning)
}

func (b *XRecord) handles(c *codec, o *Object) { eRestHandles(c, &b.ObjIDs) }

// DictionaryVar §20.4.74.
type DictionaryVar struct {
	Schema byte
	Value  string
}

func (b *DictionaryVar) data(c *codec, o *Object) {
	c.RC(&b.Schema)
	c.TV(&b.Value)
}
func (b *DictionaryVar) handles(c *codec, o *Object) {}

// IDBuffer §20.4.79.
type IDBuffer struct {
	Unknown byte
	ObjIDs  []Ref
}

func (b *IDBuffer) data(c *codec, o *Object) {
	c.RC(&b.Unknown)
	n := c.BLCount(len(b.ObjIDs))
	b.ObjIDs = Slice(c, b.ObjIDs, n)
}

func (b *IDBuffer) handles(c *codec, o *Object) {
	for i := range b.ObjIDs {
		c.H(&b.ObjIDs[i])
	}
}

// Scale §20.4.92.
type Scale struct {
	Unknown      int16
	Name         string
	PaperUnits   float64
	DrawingUnits float64
	HasUnitScale bool
}

func (b *Scale) data(c *codec, o *Object) {
	c.BS(&b.Unknown)
	c.TV(&b.Name)
	c.BD(&b.PaperUnits)
	c.BD(&b.DrawingUnits)
	c.B(&b.HasUnitScale)
}
func (b *Scale) handles(c *codec, o *Object) {}

// SortEntsTable §20.4.93. Sort handles live in the data stream.
type SortEntsTable struct {
	SortHandles []Ref
	BlockOwner  Ref
	Entities    []Ref
}

func (b *SortEntsTable) data(c *codec, o *Object) {
	n := c.BLCount(len(b.SortHandles))
	b.SortHandles = Slice(c, b.SortHandles, n)
	for i := range b.SortHandles {
		c.HData(&b.SortHandles[i])
	}
}

func (b *SortEntsTable) handles(c *codec, o *Object) {
	c.H(&b.BlockOwner)
	b.Entities = Slice(c, b.Entities, len(b.SortHandles))
	for i := range b.Entities {
		c.H(&b.Entities[i])
	}
}

// LayerIndex §20.4.83.
type LayerIndex struct {
	Timestamp1, Timestamp2 int32
	IndexLongs             []int32
	IndexNames             []string
	Entries                []Ref
}

func (b *LayerIndex) data(c *codec, o *Object) {
	c.BL(&b.Timestamp1)
	c.BL(&b.Timestamp2)
	n := c.BLCount(len(b.IndexLongs))
	b.IndexLongs = Slice(c, b.IndexLongs, n)
	b.IndexNames = Slice(c, b.IndexNames, n)
	for i := 0; i < n; i++ {
		c.BL(&b.IndexLongs[i])
		c.TV(&b.IndexNames[i])
	}
}

func (b *LayerIndex) handles(c *codec, o *Object) {
	b.Entries = Slice(c, b.Entries, len(b.IndexLongs))
	for i := range b.Entries {
		c.H(&b.Entries[i])
	}
}

// Placeholder (ACDBPLACEHOLDER) has no own fields.
type Placeholder struct{}

func (b *Placeholder) data(c *codec, o *Object)    {}
func (b *Placeholder) handles(c *codec, o *Object) {}
