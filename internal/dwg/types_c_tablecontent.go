package dwg

import (
	"strings"
	"unicode/utf16"
)

// Table content (AcDbTableContent, §20.4.97 with §20.4.98-20.4.100): from R2010 the ACAD_TABLE
// entity embeds it in place of the cell structures of the R2004-R2007 layout. The classes
// AcDbLinkedData, AcDbLinkedTableData, AcDbFormattedTableData and AcDbTableContent never exist on
// their own, so they are one structure here. ContentFormat and CellStyle are shared with TABLESTYLE
// (types_e_table.go); here a cell style is followed by its id only (BL 90), not by class and name.
// Handles are coded in data order by the handles pass.

// TableValue is a typed value (§20.4.99), used by cell contents and custom data. A string value
// is not a TV but a sized byte block in the main stream (UTF-16LE with a terminating null, as
// AutoCAD writes it); Str is derived from Bytes on read and encoded into Bytes on write when
// Bytes is not set.
type TableValue struct {
	Flags    int32   // 93, R2007+; bit 1 = kGeneral, then no typed data follows
	DataType int32   // 90: 0 unknown, 1 long, 2 double, 4 string, 8 date, 16/32 point, 64 object id, 512 general
	Long     int32   // 91
	Double   float64 // 140
	Str      string  // 1
	Bytes    []byte  // string / date / points / general: BL size + bytes
	ObjID    Ref     // 64: handle stream
	Unit     int32   // 94, R2007+
	Format   string  // 300, R2007+
	ValueStr string  // 302, R2007+
}

func (v *TableValue) code(c *codec) {
	if c.R2007 {
		c.BL(&v.Flags)
	}
	c.BL(&v.DataType)
	if !c.R2007 || v.Flags&1 == 0 {
		switch v.DataType {
		case 0, 1:
			c.BL(&v.Long)
		case 2:
			c.BD(&v.Double)
		case 4:
			if c.w && v.Bytes == nil && v.Str != "" {
				for _, u := range utf16.Encode([]rune(v.Str)) {
					v.Bytes = append(v.Bytes, byte(u), byte(u>>8))
				}
				v.Bytes = append(v.Bytes, 0, 0)
			}
			n := c.BLCount(len(v.Bytes))
			c.Raw(&v.Bytes, n)
			if !c.w {
				u := make([]uint16, 0, len(v.Bytes)/2)
				for i := 0; i+1 < len(v.Bytes); i += 2 {
					u = append(u, uint16(v.Bytes[i])|uint16(v.Bytes[i+1])<<8)
				}
				v.Str = strings.TrimRight(string(utf16.Decode(u)), "\x00")
			}
		case 8, 16, 32, 512:
			n := c.BLCount(len(v.Bytes))
			c.Raw(&v.Bytes, n)
		}
	}
	if c.R2007 {
		c.BL(&v.Unit)
		c.TV(&v.Format)
		c.TV(&v.ValueStr)
	}
}

func (v *TableValue) handles(c *codec) {
	if v.DataType == 64 && (!c.R2007 || v.Flags&1 == 0) {
		c.H(&v.ObjID)
	}
}

// CustomDataItem is one key/value of a custom data collection (§20.4.100).
type CustomDataItem struct {
	Name  string // 300
	Value TableValue
}

func codeCustomData(c *codec, items *[]CustomDataItem) {
	n := c.BLCount(len(*items))
	*items = Slice(c, *items, n)
	for i := range *items {
		c.TV(&(*items)[i].Name)
		(*items)[i].Value.code(c)
	}
}

func handlesCustomData(c *codec, items []CustomDataItem) {
	for i := range items {
		items[i].Value.handles(c)
	}
}

// TableColumn is one column of the linked table data.
type TableColumn struct {
	Name       string // 300
	Custom     int32  // 91
	CustomData []CustomDataItem
	Style      CellStyle // overrides for the column
	StyleID    int32     // 90: base cell style in the table style, 0 if none
	Width      float64   // 40
}

// CellAttr is a block attribute value of a block content.
type CellAttr struct {
	Def   Ref    // 330
	Value string // 301
	Index int32  // 92, from 1
}

// CellContent is one content of a cell: a value, a field or a block.
type CellContent struct {
	Type      int32 // 90: 1 value, 2 field, 4 block
	Value     TableValue
	Field     Ref // 340
	Block     Ref // 340
	Attrs     []CellAttr
	HasFormat int16 // 170
	Format    ContentFormat
}

func (t *CellContent) code(c *codec) {
	c.BL(&t.Type)
	if t.Type == 1 {
		t.Value.code(c)
	}
	n := c.BLCount(len(t.Attrs))
	t.Attrs = Slice(c, t.Attrs, n)
	for i := range t.Attrs {
		c.TV(&t.Attrs[i].Value)
		c.BL(&t.Attrs[i].Index)
	}
	c.BS(&t.HasFormat)
	if t.HasFormat != 0 {
		t.Format.code(c, false)
	}
}

func (t *CellContent) handles(c *codec) {
	switch t.Type {
	case 1:
		t.Value.handles(c)
	case 2:
		c.H(&t.Field)
	case 4:
		c.H(&t.Block)
	}
	for i := range t.Attrs {
		c.H(&t.Attrs[i].Def)
	}
	if t.HasFormat != 0 {
		t.Format.code(c, true)
	}
}

// CellGeometry (§20.4.98) — layout of a cell content, optional.
type CellGeometry struct {
	TopLeft  [3]float64
	Center   [3]float64
	ContentW float64
	ContentH float64
	Width    float64
	Height   float64
	Flags    int32
}

// ContentCell is one cell of the linked table data (TableCell is the pre-R2010 cell).
type ContentCell struct {
	State      int32  // 90: 1 content locked, 2 content readonly, 4 linked, 8 content modified, 0x10 format locked, 0x20 format readonly, 0x40 format modified
	Tooltip    string // 300
	Custom     int32  // 91
	CustomData []CustomDataItem
	HasLinked  int32 // 92
	Link       Ref   // 340
	LinkRows   int32 // 93
	LinkCols   int32 // 94
	LinkUnk    int32 // 96
	Contents   []CellContent
	Style      CellStyle // overrides for the cell
	StyleID    int32     // 90
	HasGeom    int32     // 91: the unknown block below follows
	Unknown91  int32
	Unknown40  float64
	Unknown41  float64
	GeomFlags  int32
	UnknownH   Ref
	Geometry   CellGeometry
}

func (t *ContentCell) code(c *codec) {
	c.BL(&t.State)
	c.TV(&t.Tooltip)
	c.BL(&t.Custom)
	codeCustomData(c, &t.CustomData)
	c.BL(&t.HasLinked)
	if t.HasLinked != 0 {
		c.BL(&t.LinkRows)
		c.BL(&t.LinkCols)
		c.BL(&t.LinkUnk)
	}
	n := c.BLCount(len(t.Contents))
	t.Contents = Slice(c, t.Contents, n)
	for i := range t.Contents {
		t.Contents[i].code(c)
	}
	t.Style.codeBody(c, false)
	c.BL(&t.StyleID)
	c.BL(&t.HasGeom)
	if t.HasGeom != 0 {
		c.BL(&t.Unknown91)
		c.BD(&t.Unknown40)
		c.BD(&t.Unknown41)
		c.BL(&t.GeomFlags)
		if t.GeomFlags != 0 {
			g := &t.Geometry
			c.BD3(&g.TopLeft)
			c.BD3(&g.Center)
			c.BD(&g.ContentW)
			c.BD(&g.ContentH)
			c.BD(&g.Width)
			c.BD(&g.Height)
			c.BL(&g.Flags)
		}
	}
}

func (t *ContentCell) handles(c *codec) {
	handlesCustomData(c, t.CustomData)
	if t.HasLinked != 0 {
		c.H(&t.Link)
	}
	for i := range t.Contents {
		t.Contents[i].handles(c)
	}
	t.Style.codeBody(c, true)
	if t.HasGeom != 0 {
		c.H(&t.UnknownH)
	}
}

// ContentRow is one row of the linked table data.
type ContentRow struct {
	Cells      []ContentCell
	Custom     int32 // 91
	CustomData []CustomDataItem
	Style      CellStyle // overrides for the row
	StyleID    int32     // 90
	Height     float64   // 40
}

// TableContent is the embedded AcDbTableContent of an R2010+ table.
type TableContent struct {
	Name        string // 1
	Description string // 300
	Columns     []TableColumn
	Rows        []ContentRow
	FieldRefs   []Ref      // hard owners of the AcDbField objects referenced by contents
	Style       CellStyle  // the table's overrides of the table style's "Table" cell style
	Merged      [][4]int32 // 91 top row, 92 left column, 93 bottom row, 94 right column
	TableStyle  Ref        // 340
}

func (t *TableContent) code(c *codec) {
	c.TV(&t.Name)
	c.TV(&t.Description)
	nc := c.BLCount(len(t.Columns))
	t.Columns = Slice(c, t.Columns, nc)
	for i := range t.Columns {
		col := &t.Columns[i]
		c.TV(&col.Name)
		c.BL(&col.Custom)
		codeCustomData(c, &col.CustomData)
		col.Style.codeBody(c, false)
		c.BL(&col.StyleID)
		c.BD(&col.Width)
	}
	nr := c.BLCount(len(t.Rows))
	t.Rows = Slice(c, t.Rows, nr)
	for i := range t.Rows {
		r := &t.Rows[i]
		n := c.BLCount(len(r.Cells))
		r.Cells = Slice(c, r.Cells, n)
		for j := range r.Cells {
			r.Cells[j].code(c)
		}
		c.BL(&r.Custom)
		codeCustomData(c, &r.CustomData)
		r.Style.codeBody(c, false)
		c.BL(&r.StyleID)
		c.BD(&r.Height)
	}
	nf := c.BLCount(len(t.FieldRefs))
	t.FieldRefs = Slice(c, t.FieldRefs, nf)
	t.Style.codeBody(c, false)
	nm := c.BLCount(len(t.Merged))
	t.Merged = Slice(c, t.Merged, nm)
	for i := range t.Merged {
		for k := 0; k < 4; k++ {
			c.BL(&t.Merged[i][k])
		}
	}
}

func (t *TableContent) handles(c *codec) {
	for i := range t.Columns {
		handlesCustomData(c, t.Columns[i].CustomData)
		t.Columns[i].Style.codeBody(c, true)
	}
	for i := range t.Rows {
		r := &t.Rows[i]
		for j := range r.Cells {
			r.Cells[j].handles(c)
		}
		handlesCustomData(c, r.CustomData)
		r.Style.codeBody(c, true)
	}
	for i := range t.FieldRefs {
		c.H(&t.FieldRefs[i])
	}
	t.Style.codeBody(c, true)
	c.H(&t.TableStyle)
}

// TableBreakHeight is a manual break position of the R2010+ table.
type TableBreakHeight struct {
	Position [3]float64
	Height   float64
	Flags    int32
}

// TableRowRange is a break row range of the R2010+ table (there is always one).
type TableRowRange struct {
	Position [3]float64
	StartRow int32
	EndRow   int32
}

// TableBreak is the optional break data of the R2010+ table.
type TableBreak struct {
	Options  int32 // 1 enable, 2 repeat top labels, 4 repeat bottom labels, 8 manual positions, 16 manual heights
	Flow     int32 // 1 right, 2 vertical, 4 left
	Spacing  float64
	Unknown1 int32
	Unknown2 int32
	Heights  []TableBreakHeight
}
