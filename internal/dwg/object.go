package dwg

import (
	"encoding/binary"
	"fmt"

	"github.com/LeeBhin/dwgconv/internal/bits"
)

// EED is one extended entity data block (§28), kept as raw bytes.
type EED struct {
	App  Ref
	Data []byte
}

// Body is the type-specific part of an object.
type Body interface {
	// data codes the type-specific data stream fields (after the common data).
	data(c *codec, o *Object)
	// handles codes the type-specific handle stream fields (after the common handles).
	handles(c *codec, o *Object)
}

// EntityCommon is the common entity data (§20.4.1-2).
type EntityCommon struct {
	Graphic      []byte // proxy graphics, if present
	HasGraphic   bool
	EntMode      byte
	HasDS        bool
	Color        Color
	LtypeScale   float64
	LtypeFlags   byte
	PlotFlags    byte
	MatFlags     byte
	Shadow       byte
	FullVS       bool
	FaceVS       bool
	EdgeVS       bool
	Invisible    int16
	Lineweight   byte
	Layer        Ref
	Ltype        Ref
	Material     Ref
	PlotStyle    Ref
	FullVSH      Ref
	FaceVSH      Ref
	EdgeVSH      Ref
	ColorBookRef Ref
}

// Object is one decoded database object.
type Object struct {
	Handle    uint64
	Type      int
	IsEntity  bool
	EED       []EED
	Reactors  []Ref
	XDic      Ref
	XDicMiss  bool
	HasDS     bool
	Owner     Ref
	Ent       *EntityCommon
	Body      Body
	StrFlag   bool   // R2007+: string stream present
	Raw       []byte // original bytes (MS..CRC) when the type is not decoded
	DecodeErr error
	// LeftData/LeftHandles are the bits not consumed by the decoder (diagnostics).
	LeftData, LeftHandles int
	// pad is the original trailing padding of the handle stream; reused on encode
	// when the length matches so unchanged objects re-encode byte-identically.
	pad     uint64
	padBits int
}

// TypeName returns a readable type name.
func (o *Object) TypeName(classes []Class) string {
	if n, ok := fixedTypeNames[o.Type]; ok {
		return n
	}
	if o.Type >= 500 && o.Type-500 < len(classes) {
		return classes[o.Type-500].DXFName
	}
	return fmt.Sprintf("type%d", o.Type)
}

// splitObject parses the object envelope at off and returns the codec positioned at the start of data.
func splitObject(buf []byte, off int, v Version) (c *codec, typ int, end int, err error) {
	size, p := bits.MS(buf, off)
	var hbits int64
	if v.R2010 {
		hbits, p = bits.MC(buf, p, false)
	}
	dataStart := p
	end = p + size // CRC follows
	if end+2 > len(buf) {
		return nil, 0, 0, fmt.Errorf("object at %d: size %d out of range", off, size)
	}
	whole := bits.NewReader(buf[:end])
	whole.Pos = dataStart * 8
	c = &codec{Version: v}
	typ = whole.OT(v.R2010)
	endbit := 0
	if v.R2010 {
		endbit = end*8 - int(hbits)
	} else {
		var bitsize int32
		bitsize = whole.RL()
		endbit = dataStart*8 + int(bitsize)
	}
	c.hdl = whole.Sub(endbit, end*8)
	c.dat = whole.Sub(whole.Pos, endbit)
	c.str = c.dat
	if v.R2007 {
		// §20.1: the last data bit flags a string stream; its size is stored just before the flag.
		flagPos := endbit - 1
		if (buf[flagPos>>3]>>(7-uint(flagPos&7)))&1 == 1 {
			sz := whole.Sub(flagPos-16, flagPos)
			strSize := int(uint16(sz.RS()))
			pos := flagPos - 16
			if strSize&0x8000 != 0 {
				hi := whole.Sub(pos-16, pos)
				hiSize := int(uint16(hi.RS()))
				pos -= 16
				strSize = strSize&0x7FFF | hiSize<<15
			}
			c.str = whole.Sub(pos-strSize, pos)
			c.dat.End = pos - strSize
		} else {
			c.dat.End = flagPos
		}
	}
	return c, typ, end, nil
}

// decodeCommon reads handle, EED and the common object/entity data.
func (o *Object) decodeCommon(c *codec) {
	h := c.dat.H()
	o.Handle = h.Value
	c.self = o.Handle
	for {
		n := int(uint16(c.dat.BS()))
		if n == 0 {
			break
		}
		app := c.dat.H()
		o.EED = append(o.EED, EED{App: Ref{app.Code, app.Value}, Data: c.dat.Raw(n)})
	}
	o.codeCommonData(c)
}

func (o *Object) codeCommonData(c *codec) {
	if o.IsEntity {
		e := o.Ent
		c.B(&e.HasGraphic)
		if e.HasGraphic {
			n := uint64(len(e.Graphic))
			if c.R2010 {
				c.BLL(&n)
			} else {
				v := int32(n)
				c.RL(&v)
				n = uint64(v)
			}
			c.Raw(&e.Graphic, int(n))
		}
		c.BB(&e.EntMode)
	}
	n := c.BLCount(len(o.Reactors))
	o.Reactors = Slice(c, o.Reactors, n)
	c.B(&o.XDicMiss)
	if c.R2013 {
		c.B(&o.HasDS)
	}
	if o.IsEntity {
		e := o.Ent
		e.Color.code(c)
		c.BD(&e.LtypeScale)
		c.BB(&e.LtypeFlags)
		c.BB(&e.PlotFlags)
		if c.R2007 {
			c.BB(&e.MatFlags)
			c.RC(&e.Shadow)
		}
		if c.R2010 {
			c.B(&e.FullVS)
			c.B(&e.FaceVS)
			c.B(&e.EdgeVS)
		}
		c.BS(&e.Invisible)
		c.RC(&e.Lineweight)
	}
}

func (o *Object) codeCommonHandles(c *codec) {
	if !o.IsEntity || o.Ent.EntMode == 0 {
		c.H(&o.Owner)
	}
	for i := range o.Reactors {
		c.H(&o.Reactors[i])
	}
	if !o.XDicMiss {
		c.H(&o.XDic)
	}
	if !o.IsEntity {
		return
	}
	e := o.Ent
	if e.Color.flags()&0x4000 != 0 {
		c.H(&e.ColorBookRef)
	}
	c.H(&e.Layer)
	if e.LtypeFlags == 3 {
		c.H(&e.Ltype)
	}
	if c.R2007 && e.MatFlags == 3 {
		c.H(&e.Material)
	}
	if e.PlotFlags == 3 {
		c.H(&e.PlotStyle)
	}
	if c.R2010 {
		if e.FullVS {
			c.H(&e.FullVSH)
		}
		if e.FaceVS {
			c.H(&e.FaceVSH)
		}
		if e.EdgeVS {
			c.H(&e.EdgeVSH)
		}
	}
}

// Color is the entity color (ENC, §2.11). Raw holds the BS: flags in the high bits, ACI in the low 9.
type Color struct {
	Raw          int16
	RGB          int32
	Transparency int32
}

func (col Color) flags() uint16 { return uint16(col.Raw) & 0xE000 }

// ACI returns the color index (256 = BYLAYER, 0 = BYBLOCK).
func (col Color) ACI() int { return int(uint16(col.Raw) & 0x1FF) }

func (col *Color) code(c *codec) {
	c.BS(&col.Raw)
	if col.flags()&0x8000 != 0 {
		c.BL(&col.RGB)
	}
	if col.flags()&0x2000 != 0 {
		c.BL(&col.Transparency)
	}
}

// Encode serializes the object (MS..CRC).
func (o *Object) Encode(v Version, text TextCodec) []byte {
	if o.Body == nil && o.Raw != nil {
		return o.Raw
	}
	c := &codec{Version: v, w: true, text: text, self: o.Handle,
		wd: &bits.Writer{}, wh: &bits.Writer{}}
	if v.R2007 {
		c.ws = &bits.Writer{}
	} else {
		c.ws = c.wd
	}
	c.wd.OT(o.Type, v.R2010)
	bitsizePos := -1
	if !v.R2010 {
		bitsizePos = c.wd.Len()
		c.wd.RL(0) // patched below
	}
	c.wd.H(bits.Handle{Code: 0, Value: o.Handle})
	for _, e := range o.EED {
		c.wd.BS(int16(len(e.Data)))
		c.wd.H(bits.Handle{Code: e.App.Code, Value: e.App.H})
		c.wd.Raw(e.Data)
	}
	c.wd.BS(0)
	o.codeCommonData(c)
	o.Body.data(c, o)
	o.codeCommonHandles(c)
	o.Body.handles(c, o)

	out := &bits.Writer{}
	out.AppendWriter(c.wd)
	if v.R2007 {
		if o.StrFlag || c.ws.Len() > 0 {
			n := c.ws.Len()
			out.AppendWriter(c.ws)
			if n >= 0x8000 {
				out.RS(int16(n >> 15))
				out.RS(int16(n&0x7FFF | 0x8000))
			} else {
				out.RS(int16(n))
			}
			out.B(1)
		} else {
			out.B(0)
		}
	}
	endbit := out.Len()
	out.AppendWriter(c.wh)
	if n := (8 - out.Len()%8) % 8; n > 0 && n == o.padBits {
		out.Bits(o.pad, n)
	}
	out.PadByte()
	body := out.Bytes()
	if !v.R2010 {
		patchBits(body, bitsizePos, uint32(endbit))
	}
	var buf []byte
	buf = bits.PutMS(buf, len(body))
	if v.R2010 {
		buf = bits.PutMCUnsigned(buf, uint64(out.Len()-endbit))
	}
	buf = append(buf, body...)
	crc := bits.CRC8(0xC0C1, buf)
	return binary.LittleEndian.AppendUint16(buf, crc)
}

// patchBits writes a 32-bit little-endian value at bit position pos of an MSB-first stream.
func patchBits(b []byte, pos int, v uint32) {
	for i := 0; i < 32; i++ {
		byteIdx, bit := i/8, 7-i%8
		val := (v >> (8 * uint(byteIdx))) >> uint(bit) & 1
		p := pos + i
		mask := byte(1 << (7 - uint(p&7)))
		if val == 1 {
			b[p>>3] |= mask
		} else {
			b[p>>3] &^= mask
		}
	}
}
