// Package dwg decodes and encodes DWG objects (Open Design Specification §20).
//
// Every object type describes its fields once, in a method taking *codec; the same
// method reads when the codec is decoding and writes when it is encoding.
package dwg

import (
	"github.com/LeeBhin/dwgconv/internal/bits"
)

// Ref is a handle reference resolved to an absolute handle. Code is the reference
// code (§2.13); relative codes (6, 8, 0xA, 0xC) are re-derived on write when possible.
type Ref struct {
	Code byte
	H    uint64
}

func (r Ref) IsNull() bool { return r.H == 0 }

// Version flags.
type Version struct {
	R2007, R2010, R2013, R2018 bool
}

func versionOf(s string) Version {
	return Version{R2007: s >= "AC1021", R2010: s >= "AC1024", R2013: s >= "AC1027", R2018: s >= "AC1032"}
}

// codec carries the three object streams (§20.1): data, strings (R2007+) and handles.
type codec struct {
	Version
	w      bool
	self   uint64 // handle of the object being coded (base for relative refs)
	text   TextCodec
	dat    *bits.Reader
	str    *bits.Reader
	hdl    *bits.Reader
	wd, ws *bits.Writer
	wh     *bits.Writer
}

func (c *codec) B(v *bool) {
	if c.w {
		c.wd.Bool(*v)
	} else {
		*v = c.dat.Bool()
	}
}

func (c *codec) BB(v *byte) {
	if c.w {
		c.wd.BB(*v)
	} else {
		*v = c.dat.BB()
	}
}

func (c *codec) B3(v *byte) {
	if c.w {
		c.wd.B3(*v)
	} else {
		*v = c.dat.B3()
	}
}

func (c *codec) RC(v *byte) {
	if c.w {
		c.wd.RC(*v)
	} else {
		*v = c.dat.RC()
	}
}

func (c *codec) RS(v *int16) {
	if c.w {
		c.wd.RS(*v)
	} else {
		*v = c.dat.RS()
	}
}

func (c *codec) RL(v *int32) {
	if c.w {
		c.wd.RL(*v)
	} else {
		*v = c.dat.RL()
	}
}

func (c *codec) RD(v *float64) {
	if c.w {
		c.wd.RD(*v)
	} else {
		*v = c.dat.RD()
	}
}

func (c *codec) BS(v *int16) {
	if c.w {
		c.wd.BS(*v)
	} else {
		*v = c.dat.BS()
	}
}

func (c *codec) BL(v *int32) {
	if c.w {
		c.wd.BL(*v)
	} else {
		*v = c.dat.BL()
	}
}

func (c *codec) BLL(v *uint64) {
	if c.w {
		c.wd.BLL(*v)
	} else {
		*v = c.dat.BLL()
	}
}

func (c *codec) BD(v *float64) {
	if c.w {
		c.wd.BD(*v)
	} else {
		*v = c.dat.BD()
	}
}

func (c *codec) DD(v *float64, def float64) {
	if c.w {
		c.wd.DD(*v, def)
	} else {
		*v = c.dat.DD(def)
	}
}

func (c *codec) BT(v *float64) {
	if c.w {
		c.wd.BT(*v)
	} else {
		*v = c.dat.BT()
	}
}

func (c *codec) BE(v *[3]float64) {
	if c.w {
		c.wd.BE(*v)
	} else {
		*v = c.dat.BE()
	}
}

func (c *codec) Raw(v *[]byte, n int) {
	if c.w {
		c.wd.Raw(*v)
	} else {
		*v = c.dat.Raw(n)
	}
}

func (c *codec) RD2(v *[2]float64) { c.RD(&v[0]); c.RD(&v[1]) }
func (c *codec) RD3(v *[3]float64) { c.RD(&v[0]); c.RD(&v[1]); c.RD(&v[2]) }
func (c *codec) BD2(v *[2]float64) { c.BD(&v[0]); c.BD(&v[1]) }
func (c *codec) BD3(v *[3]float64) { c.BD(&v[0]); c.BD(&v[1]); c.BD(&v[2]) }

// DD2 reads/writes 2 DD values with the given defaults.
func (c *codec) DD2(v *[2]float64, def [2]float64) { c.DD(&v[0], def[0]); c.DD(&v[1], def[1]) }

// HData reads/writes a handle stored in the data stream (absolute form).
func (c *codec) HData(v *Ref) {
	if c.w {
		c.wd.H(bits.Handle{Code: v.Code, Value: v.H})
	} else {
		h := c.dat.H()
		*v = Ref{Code: h.Code, H: h.Value}
	}
}

// H reads/writes a handle in the handle stream.
func (c *codec) H(v *Ref) {
	if c.w {
		c.wh.H(encodeRef(*v, c.self))
		return
	}
	h := c.hdl.H()
	*v = resolveRef(h, c.self)
}

func resolveRef(h bits.Handle, base uint64) Ref {
	switch h.Code {
	case 6:
		return Ref{6, base + 1}
	case 8:
		return Ref{8, base - 1}
	case 0xA:
		return Ref{0xA, base + h.Value}
	case 0xC:
		return Ref{0xC, base - h.Value}
	}
	return Ref{h.Code, h.Value}
}

func encodeRef(r Ref, base uint64) bits.Handle {
	switch r.Code {
	case 6:
		if r.H == base+1 {
			return bits.Handle{Code: 6}
		}
	case 8:
		if r.H == base-1 {
			return bits.Handle{Code: 8}
		}
	case 0xA:
		if r.H > base {
			return bits.Handle{Code: 0xA, Value: r.H - base}
		}
	case 0xC:
		if r.H < base {
			return bits.Handle{Code: 0xC, Value: base - r.H}
		}
	default:
		return bits.Handle{Code: r.Code, Value: r.H}
	}
	// relative form no longer fits: soft pointer, absolute
	return bits.Handle{Code: 4, Value: r.H}
}

// TV is variable text: R2007+ from the string stream (UTF-16), earlier inline in the drawing codepage.
func (c *codec) TV(v *string) {
	if c.R2007 {
		if c.w {
			c.ws.TU(*v)
		} else {
			*v = c.str.TU()
		}
		return
	}
	if c.w {
		c.wd.TV8(c.text.Encode(*v))
	} else {
		*v = c.text.Decode(c.dat.TV8())
	}
}

// Count helpers for loops: reading returns the stored count, writing stores len.
func (c *codec) BLCount(n int) int {
	v := int32(n)
	c.BL(&v)
	return int(v)
}

func (c *codec) BSCount(n int) int {
	v := int16(n)
	c.BS(&v)
	return int(uint16(v))
}

func (c *codec) RLCount(n int) int {
	v := int32(n)
	c.RL(&v)
	return int(v)
}

// Slice resizes s to n when reading and returns it.
func Slice[T any](c *codec, s []T, n int) []T {
	if c.w {
		return s
	}
	return make([]T, n)
}

// CmColor is the CMC color of non-entity objects (§2.11, R2004+ form).
type CmColor struct {
	Index    int16
	RGB      int32
	Flags    byte
	Name     string
	BookName string
}

func (c *codec) CMC(v *CmColor) {
	c.BS(&v.Index)
	c.BL(&v.RGB)
	c.RC(&v.Flags)
	if v.Flags&1 != 0 {
		c.TV(&v.Name)
	}
	if v.Flags&2 != 0 {
		c.TV(&v.BookName)
	}
}
