package dwg

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// bcodec reads or writes the byte-aligned little-endian sections (SummaryInfo, AppInfo, Template,
// ObjFreeSpace, AuxHeader, FileDepList, RevHistory, Preview); like codec, one method serves both.
type bcodec struct {
	w    bool
	buf  []byte
	p    int
	text TextCodec
}

func (c *bcodec) need(n int) {
	if !c.w && c.p+n > len(c.buf) {
		panic(fmt.Errorf("section: read past end (%d+%d > %d)", c.p, n, len(c.buf)))
	}
}

func (c *bcodec) U8(v *byte) {
	if c.w {
		c.buf = append(c.buf, *v)
		return
	}
	c.need(1)
	*v = c.buf[c.p]
	c.p++
}

func (c *bcodec) I16(v *int16) {
	if c.w {
		c.buf = binary.LittleEndian.AppendUint16(c.buf, uint16(*v))
		return
	}
	c.need(2)
	*v = int16(binary.LittleEndian.Uint16(c.buf[c.p:]))
	c.p += 2
}

func (c *bcodec) I32(v *int32) {
	if c.w {
		c.buf = binary.LittleEndian.AppendUint32(c.buf, uint32(*v))
		return
	}
	c.need(4)
	*v = int32(binary.LittleEndian.Uint32(c.buf[c.p:]))
	c.p += 4
}

func (c *bcodec) U32(v *uint32) {
	if c.w {
		c.buf = binary.LittleEndian.AppendUint32(c.buf, *v)
		return
	}
	c.need(4)
	*v = binary.LittleEndian.Uint32(c.buf[c.p:])
	c.p += 4
}

func (c *bcodec) U64(v *uint64) {
	if c.w {
		c.buf = binary.LittleEndian.AppendUint64(c.buf, *v)
		return
	}
	c.need(8)
	*v = binary.LittleEndian.Uint64(c.buf[c.p:])
	c.p += 8
}

func (c *bcodec) Bytes(v *[]byte, n int) {
	if c.w {
		c.buf = append(c.buf, *v...)
		return
	}
	c.need(n)
	*v = append([]byte(nil), c.buf[c.p:c.p+n]...)
	c.p += n
}

// Rest codes all remaining bytes (reading) or the stored bytes (writing).
func (c *bcodec) Rest(v *[]byte) {
	if c.w {
		c.buf = append(c.buf, *v...)
		return
	}
	*v = append([]byte(nil), c.buf[c.p:]...)
	c.p = len(c.buf)
}

func (c *bcodec) Julian(v *JulianDate) { c.I32(&v.Day); c.I32(&v.Ms) }

// Count codes an element count of the given width (2 or 4 bytes).
func (c *bcodec) Count(n int, width int) int {
	if width == 2 {
		v := int16(n)
		c.I16(&v)
		return int(uint16(v))
	}
	v := int32(n)
	c.I32(&v)
	return int(v)
}

// Str16 is a string with an Int16 length that counts the terminating NUL: bytes in the drawing
// codepage (R2004, wide=false) or UTF-16 characters (R2007+, wide=true). Length 0 means no NUL.
func (c *bcodec) Str16(v *string, wide bool) {
	if c.w {
		if wide {
			u := utf16.Encode([]rune(*v))
			if len(u) > 0 || *v == "" {
				u = append(u, 0)
			}
			c.buf = binary.LittleEndian.AppendUint16(c.buf, uint16(len(u)))
			for _, x := range u {
				c.buf = binary.LittleEndian.AppendUint16(c.buf, x)
			}
			return
		}
		b := c.text.Encode(*v)
		if len(b) == 0 {
			b = []byte{0}
		}
		c.buf = binary.LittleEndian.AppendUint16(c.buf, uint16(len(b)))
		c.buf = append(c.buf, b...)
		return
	}
	var n int16
	c.I16(&n)
	if wide {
		c.need(2 * int(n))
		u := make([]uint16, n)
		for i := range u {
			u[i] = binary.LittleEndian.Uint16(c.buf[c.p+2*i:])
		}
		c.p += 2 * int(n)
		for len(u) > 0 && u[len(u)-1] == 0 {
			u = u[:len(u)-1]
		}
		*v = string(utf16.Decode(u))
		return
	}
	c.need(int(n))
	*v = c.text.Decode(c.buf[c.p : c.p+int(n)])
	c.p += int(n)
}

// Str32 is a string with an Int32 byte length and no terminator (FileDepList): UTF-16LE for
// R2007+ (wide), the drawing codepage before.
func (c *bcodec) Str32(v *string, wide bool) {
	if wide {
		if c.w {
			u := utf16.Encode([]rune(*v))
			c.buf = binary.LittleEndian.AppendUint32(c.buf, uint32(2*len(u)))
			for _, x := range u {
				c.buf = binary.LittleEndian.AppendUint16(c.buf, x)
			}
			return
		}
		var n int32
		c.I32(&n)
		c.need(int(n))
		u := make([]uint16, n/2)
		for i := range u {
			u[i] = binary.LittleEndian.Uint16(c.buf[c.p+2*i:])
		}
		c.p += int(n)
		*v = string(utf16.Decode(u))
		return
	}
	if c.w {
		b := c.text.Encode(*v)
		if len(b) > 0 {
			b = b[:len(b)-1] // Encode appends a NUL
		}
		c.buf = binary.LittleEndian.AppendUint32(c.buf, uint32(len(b)))
		c.buf = append(c.buf, b...)
		return
	}
	var n int32
	c.I32(&n)
	c.need(int(n))
	*v = c.text.Decode(c.buf[c.p : c.p+int(n)])
	c.p += int(n)
}

// decodeBytes runs f over data and reports unread bytes as an error.
func decodeBytes(data []byte, text TextCodec, f func(c *bcodec)) error {
	c := &bcodec{buf: data, text: text}
	if err := catch(func() { f(c) }); err != nil {
		return err
	}
	if c.p != len(data) {
		return fmt.Errorf("section: %d of %d bytes decoded", c.p, len(data))
	}
	return nil
}

func encodeBytes(text TextCodec, f func(c *bcodec)) []byte {
	c := &bcodec{w: true, text: text}
	f(c)
	return c.buf
}
