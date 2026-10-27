package dwg

import (
	"bytes"

	"github.com/LeeBhin/dwgconv/internal/bits"
)

func init() {
	bodyFactory[0x25] = func() Body { return &Acis{} } // REGION
	bodyFactory[0x26] = func() Body { return &Acis{} } // 3DSOLID
	bodyFactory[0x27] = func() Body { return &Acis{} } // BODY
	bodyFactory[0x1F2] = func() Body { return &ProxyEntity{} }
}

// AcisWire is one wire of the wireframe/silhouette data (§20.4.41).
type AcisWire struct {
	Type         byte
	Selection    int32
	Color        int16
	AcisIndex    int32
	Points       [][3]float64
	HasTransform bool
	XAxis, YAxis [3]float64
	ZAxis, Trans [3]float64
	Scale        float64
	HasRotation  bool
	HasReflect   bool
	HasShear     bool
}

func (w *AcisWire) code(c *codec) {
	c.RC(&w.Type)
	c.BL(&w.Selection)
	c.BS(&w.Color)
	c.BL(&w.AcisIndex)
	n := c.BLCount(len(w.Points))
	w.Points = Slice(c, w.Points, n)
	for i := range w.Points {
		c.BD3(&w.Points[i])
	}
	c.B(&w.HasTransform)
	if w.HasTransform {
		c.BD3(&w.XAxis)
		c.BD3(&w.YAxis)
		c.BD3(&w.ZAxis)
		c.BD3(&w.Trans)
		c.BD(&w.Scale)
		c.B(&w.HasRotation)
		c.B(&w.HasReflect)
		c.B(&w.HasShear)
	}
}

func cWires(c *codec, ws []AcisWire) []AcisWire {
	n := c.BLCount(len(ws))
	ws = Slice(c, ws, n)
	for i := range ws {
		ws[i].code(c)
	}
	return ws
}

// AcisSilhouette §20.4.41.
type AcisSilhouette struct {
	VPID        int32
	Target      [3]float64
	Dir         [3]float64
	Up          [3]float64
	Perspective bool
	Wires       []AcisWire
}

// AcisData is one ACIS block: version 1 keeps the encrypted SAT blocks as stored,
// version 2 keeps the SAT/SAB file bytes.
type AcisData struct {
	Blocks [][]byte // version 1 (each block as stored, before the 0x9F-c decoding)
	File   []byte   // version 2
}

var sabEnd = []byte("End\x0E\x02of\x0E\x04ACIS\x0D\x04data")
var satEnd = []byte("End-of-ACIS-data")

func (a *AcisData) code(c *codec, version int16) {
	if version == 1 {
		if c.w {
			for _, b := range a.Blocks {
				n := int32(len(b))
				c.BL(&n)
				c.wd.Raw(b)
			}
			var zero int32
			c.BL(&zero)
			return
		}
		for {
			var n int32
			c.BL(&n)
			if n == 0 {
				return
			}
			a.Blocks = append(a.Blocks, c.dat.Raw(int(n)))
		}
	}
	// version 2: no length; scan to the end marker (unverified for SAT text).
	if c.w {
		c.wd.Raw(a.File)
		return
	}
	var buf []byte
	for {
		buf = append(buf, c.dat.RC())
		if bytes.HasSuffix(buf, sabEnd) || bytes.HasSuffix(buf, satEnd) {
			a.File = buf
			return
		}
	}
}

// SAT decodes version 1 blocks into SAT text (§20.4.41: chars 0x20..0x7E map to 0x9F-c).
func (a *AcisData) SAT() []byte {
	var out []byte
	for _, b := range a.Blocks {
		for _, ch := range b {
			if ch >= 0x20 && ch <= 0x7E {
				ch = 0x9F - ch
			}
			if ch == '\t' {
				ch = ' '
			}
			out = append(out, ch)
		}
	}
	return out
}

// Acis is REGION / 3DSOLID / BODY (§20.4.41). Unverified: no samples contain these.
type Acis struct {
	Empty           bool
	Unknown         bool
	Version         int16
	Data            AcisData `conv:"derived"` // DXF carries the decoded SAT text, not the block split
	Wireframe       bool
	PointPresent    bool
	Point           [3]float64
	NumIsolines     int32
	IsolinesPresent bool
	Wires           []AcisWire
	Silhouettes     []AcisSilhouette
	Empty2          bool
	Data2           AcisData
	Unknown2007     int32
	HistoryID       Ref // 350, R2007+
}

func (b *Acis) data(c *codec, o *Object) {
	c.B(&b.Empty)
	if !b.Empty {
		c.B(&b.Unknown)
		c.BS(&b.Version)
		b.Data.code(c, b.Version)
		c.B(&b.Wireframe)
		if b.Wireframe {
			c.B(&b.PointPresent)
			if b.PointPresent {
				c.BD3(&b.Point)
			}
			c.BL(&b.NumIsolines)
			c.B(&b.IsolinesPresent)
			b.Wires = cWires(c, b.Wires)
			n := c.BLCount(len(b.Silhouettes))
			b.Silhouettes = Slice(c, b.Silhouettes, n)
			for i := range b.Silhouettes {
				s := &b.Silhouettes[i]
				c.BL(&s.VPID)
				c.BD3(&s.Target)
				c.BD3(&s.Dir)
				c.BD3(&s.Up)
				c.B(&s.Perspective)
				s.Wires = cWires(c, s.Wires)
			}
		}
		c.B(&b.Empty2)
		if !b.Empty2 {
			b.Data2.code(c, b.Version)
		}
	}
	if c.R2007 {
		c.BL(&b.Unknown2007)
	}
}

func (b *Acis) handles(c *codec, o *Object) {
	if c.R2007 {
		c.H(&b.HistoryID)
	}
}

// ProxyEntity ACAD_PROXY_ENTITY (§20.4.90). The proxy data bits, string stream bits
// and object handles are kept verbatim. Unverified: no samples contain proxies.
type ProxyEntity struct {
	ClassID      int32 // 91
	DrawingFmt   int32 // 95, before R2018
	Version      int32 // 71, R2018+
	MaintVersion int32 // 97, R2018+
	OrigDXF      bool  // 70
	DataBits     []byte
	NDataBits    int
	StrBits      []byte `conv:"derived"` // R2007+ string stream bits; not in DXF
	NStrBits     int    `conv:"derived"`
	Handles      []Ref
}

// cRawBits reads all remaining bits of r, or writes the stored bits to w.
func cRawBits(c *codec, r *bits.Reader, w *bits.Writer, data *[]byte, n *int) {
	if c.w {
		w.AppendBits(*data, 0, *n)
		return
	}
	var tmp bits.Writer
	*n = r.Remaining()
	for r.Remaining() > 0 {
		tmp.B(r.B())
	}
	*data = append([]byte(nil), tmp.Bytes()...)
}

func (b *ProxyEntity) data(c *codec, o *Object) {
	c.BL(&b.ClassID)
	if c.R2018 {
		c.BL(&b.Version)
		c.BL(&b.MaintVersion)
	} else {
		c.BL(&b.DrawingFmt)
	}
	c.B(&b.OrigDXF)
	cRawBits(c, c.dat, c.wd, &b.DataBits, &b.NDataBits)
	if c.R2007 && (c.w || c.str != c.dat) {
		cRawBits(c, c.str, c.ws, &b.StrBits, &b.NStrBits)
	}
}

func (b *ProxyEntity) handles(c *codec, o *Object) {
	if c.w {
		for i := range b.Handles {
			c.H(&b.Handles[i])
		}
		return
	}
	for c.hdl.Remaining() >= 8 {
		var r Ref
		c.H(&r)
		b.Handles = append(b.Handles, r)
	}
}
