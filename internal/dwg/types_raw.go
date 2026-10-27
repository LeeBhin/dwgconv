package dwg

import "github.com/LeeBhin/dwgconv/internal/bits"

// RawBody keeps the type-specific part of an object whose layout is not decoded (classes the
// specification does not document, e.g. AEC_*): the remaining data bits, the string stream
// bits (R2007+) and the remaining handle references. It re-encodes the object unchanged and is
// what DXF carries in ACAD_PROXY_OBJECT/ENTITY records (groups 310/311 and 330-360).
type RawBody struct {
	Data     []byte // data bits after the common data, MSB first
	DataBits int
	Str      []byte // string stream bits (R2007+)
	StrBits  int
	Handles  []Ref
}

func (b *RawBody) data(c *codec, o *Object) {
	if c.w {
		c.wd.AppendBits(b.Data, 0, b.DataBits)
		if c.R2007 && b.StrBits > 0 {
			c.ws.AppendBits(b.Str, 0, b.StrBits)
		}
		return
	}
	b.Data, b.DataBits = takeBits(c.dat, c.dat.Remaining())
	if c.R2007 && c.str != c.dat {
		b.Str, b.StrBits = takeBits(c.str, c.str.Remaining())
	}
}

func (b *RawBody) handles(c *codec, o *Object) {
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

// takeBits copies n bits from r into a fresh MSB-first buffer.
func takeBits(r *bits.Reader, n int) ([]byte, int) {
	w := &bits.Writer{}
	for i := 0; i < n; i++ {
		w.B(r.B())
	}
	return w.Bytes(), n
}
