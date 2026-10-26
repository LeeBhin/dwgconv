package dwg

func init() {
	classBodyFactory["TABLESTYLE"] = func() Body { return &TableStyle{} }
	classBodyFactory["CELLSTYLEMAP"] = func() Body { return &CellStyleMap{} }
}

// ContentFormat §20.4.101.3.
type ContentFormat struct {
	PropOverride  int32
	PropFlags     int32
	ValueType     int32
	ValueUnit     int32
	ValueFormat   string
	Rotation      float64
	BlockScale    float64
	CellAlignment int32
	ContentColor  CmColor
	TextStyle     Ref
	TextHeight    float64
}

// code runs the data pass (hp=false) or the handle pass (hp=true).
func (f *ContentFormat) code(c *codec, hp bool) {
	if hp {
		c.H(&f.TextStyle)
		return
	}
	c.BL(&f.PropOverride)
	c.BL(&f.PropFlags)
	c.BL(&f.ValueType)
	c.BL(&f.ValueUnit)
	c.TV(&f.ValueFormat)
	c.BD(&f.Rotation)
	c.BD(&f.BlockScale)
	c.BL(&f.CellAlignment)
	c.CMC(&f.ContentColor)
	c.BD(&f.TextHeight)
}

// CellBorder is one border of a cell style.
type CellBorder struct {
	EdgeFlags   int32
	Override    int32
	Type        int32
	Color       CmColor
	Lineweight  int32
	Ltype       Ref
	Invisible   int32
	DoubleSpace float64
}

// CellStyle §20.4.101.4 (followed by its id, class and name as stored in TABLESTYLE / CELLSTYLEMAP).
type CellStyle struct {
	Type           int32
	DataFlags      int16
	PropOverride   int32
	MergeFlags     int32
	BgColor        CmColor
	LayoutFlags    int32
	Content        ContentFormat
	MarginOverride int16
	Margins        [6]float64 // vertical, horizontal, bottom, right, h spacing, v spacing
	Borders        []CellBorder
	ID             int32
	Class          int32
	Name           string
}

func (s *CellStyle) code(c *codec, hp bool) {
	if hp {
		if s.DataFlags&1 != 0 {
			s.Content.code(c, true)
			for i := range s.Borders {
				if s.Borders[i].EdgeFlags != 0 {
					c.H(&s.Borders[i].Ltype)
				}
			}
		}
		return
	}
	c.BL(&s.Type)
	c.BS(&s.DataFlags)
	if s.DataFlags&1 != 0 {
		c.BL(&s.PropOverride)
		c.BL(&s.MergeFlags)
		c.CMC(&s.BgColor)
		c.BL(&s.LayoutFlags)
		s.Content.code(c, false)
		c.BS(&s.MarginOverride)
		if s.MarginOverride&1 != 0 {
			for i := range s.Margins {
				c.BD(&s.Margins[i])
			}
		}
		n := c.BLCount(len(s.Borders))
		s.Borders = Slice(c, s.Borders, n)
		for i := range s.Borders {
			b := &s.Borders[i]
			c.BL(&b.EdgeFlags)
			if b.EdgeFlags != 0 {
				c.BL(&b.Override)
				c.BL(&b.Type)
				c.CMC(&b.Color)
				c.BL(&b.Lineweight)
				c.BL(&b.Invisible)
				c.BD(&b.DoubleSpace)
			}
		}
	}
	c.BL(&s.ID)
	c.BL(&s.Class)
	c.TV(&s.Name)
}

// TableStyleRow is a row style of the pre-R2010 TABLESTYLE format (§20.4.101.1).
type TableStyleRow struct {
	TextStyle  Ref
	TextHeight float64
	TextAlign  int16
	TextColor  CmColor
	FillColor  CmColor
	BgEnabled  bool
	Borders    [6]struct {
		Lineweight int16
		Visible    bool
		Color      CmColor
	}
	DataType  int32
	UnitType  int32
	FormatStr string
}

// TableStyle §20.4.101. R2010+ files use the R24 layout, earlier files the classic one.
type TableStyle struct {
	Description string
	// classic (until R21)
	FlowDirection  int16
	BitFlags       int16
	HorzMargin     float64
	VertMargin     float64
	SuppressTitle  bool
	SuppressHeader bool
	Rows           [3]TableStyleRow // data, title, header
	// R24
	Unknown1     byte
	Unknown2     int32
	Unknown3     int32
	UnknownOwner Ref
	TableCell    CellStyle
	CellStyles   []CellStyle
	// LeadIDs: in TABLESTYLE each custom cell style is preceded by a BL repeating its ID
	// (not in the spec; observed in R2013/R2018 files).
	LeadIDs []int32
}

func (b *TableStyle) data(c *codec, o *Object) {
	if c.R2010 {
		c.RC(&b.Unknown1)
		c.TV(&b.Description)
		c.BL(&b.Unknown2)
		c.BL(&b.Unknown3)
		b.TableCell.code(c, false)
		n := c.BLCount(len(b.CellStyles))
		b.CellStyles = Slice(c, b.CellStyles, n)
		if c.w && len(b.LeadIDs) != n {
			b.LeadIDs = make([]int32, n)
			for i := range b.CellStyles {
				b.LeadIDs[i] = b.CellStyles[i].ID
			}
		}
		b.LeadIDs = Slice(c, b.LeadIDs, n)
		for i := range b.CellStyles {
			c.BL(&b.LeadIDs[i])
			b.CellStyles[i].code(c, false)
		}
		return
	}
	c.TV(&b.Description)
	c.BS(&b.FlowDirection)
	c.BS(&b.BitFlags)
	c.BD(&b.HorzMargin)
	c.BD(&b.VertMargin)
	c.B(&b.SuppressTitle)
	c.B(&b.SuppressHeader)
	for i := range b.Rows {
		r := &b.Rows[i]
		c.BD(&r.TextHeight)
		c.BS(&r.TextAlign)
		c.CMC(&r.TextColor)
		c.CMC(&r.FillColor)
		c.B(&r.BgEnabled)
		for k := range r.Borders {
			c.BS(&r.Borders[k].Lineweight)
			c.B(&r.Borders[k].Visible)
			c.CMC(&r.Borders[k].Color)
		}
		if c.R2007 {
			c.BL(&r.DataType)
			c.BL(&r.UnitType)
			c.TV(&r.FormatStr)
		}
	}
}

func (b *TableStyle) handles(c *codec, o *Object) {
	if c.R2010 {
		c.H(&b.UnknownOwner)
		b.TableCell.code(c, true)
		for i := range b.CellStyles {
			b.CellStyles[i].code(c, true)
		}
		return
	}
	for i := range b.Rows {
		c.H(&b.Rows[i].TextStyle)
	}
}

// CellStyleMap §20.4.102.
type CellStyleMap struct {
	CellStyles []CellStyle
}

func (b *CellStyleMap) data(c *codec, o *Object) {
	n := c.BLCount(len(b.CellStyles))
	b.CellStyles = Slice(c, b.CellStyles, n)
	for i := range b.CellStyles {
		b.CellStyles[i].code(c, false)
	}
}

func (b *CellStyleMap) handles(c *codec, o *Object) {
	for i := range b.CellStyles {
		b.CellStyles[i].code(c, true)
	}
}
