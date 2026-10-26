package dwg

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"dwgconv/internal/bits"
)

var (
	sentinelHeaderStart = []byte{0xCF, 0x7B, 0x1F, 0x23, 0xFD, 0xDE, 0x38, 0xA9, 0x5F, 0x7C, 0x68, 0xB8, 0x4E, 0x6D, 0x33, 0x5F}
	sentinelHeaderEnd   = []byte{0x30, 0x84, 0xE0, 0xDC, 0x02, 0x21, 0xC7, 0x56, 0xA0, 0x83, 0x97, 0x47, 0xB1, 0x92, 0xCC, 0xA0}
)

// JulianDate is a date stored as Julian day plus milliseconds into the day (§9, TDCREATE etc.).
// For durations (TDINDWG, TDUSRTIMER) Day counts days.
type JulianDate struct {
	Day int32
	Ms  int32
}

// Float returns the DXF form: day + fraction of day.
func (d JulianDate) Float() float64 { return float64(d.Day) + float64(d.Ms)/86400000.0 }

// Header is the AcDb:Header section.
type Header struct {
	Vars HeaderVars
	// Trailer is the 4 unknown BS of §9 ("R14+, type 5/6 only"). They are the last fields of the
	// handle stream (R2007+) or of the single stream (R2004), not of the data stream.
	Trailer [4]int16
	// HasSizeHi is the extra RL after the size (R2010/R2013 with maintenance > 3, always for R2018).
	HasSizeHi bool
	// pad holds the original trailing padding bits; reused so an unchanged header re-encodes identically.
	pad     uint64
	padBits int
	// LeftData/LeftHandles are bits the decoder did not consume (diagnostics; 0 when the layout is fully known).
	LeftData, LeftHandles, LeftStr int
}

// headerHasSizeHi reports whether the R2010+ extra size RL is present (§9).
func headerHasSizeHi(v Version, maint byte) bool {
	return v.R2018 || (v.R2010 && maint > 3)
}

// DecodeHeader parses the AcDb:Header section. maint is the file's maintenance version (file offset 0x0B).
func DecodeHeader(data []byte, v Version, text TextCodec, maint byte) (*Header, error) {
	if len(data) < 38 || !bytes.Equal(data[:16], sentinelHeaderStart) {
		return nil, fmt.Errorf("header: bad start sentinel")
	}
	h := &Header{HasSizeHi: headerHasSizeHi(v, maint)}
	size := int(binary.LittleEndian.Uint32(data[16:]))
	dataStart := 20
	if h.HasSizeHi {
		dataStart = 24
	}
	dataEnd := dataStart + size
	if dataEnd+2+16 > len(data) {
		return nil, fmt.Errorf("header: size %d out of range", size)
	}
	if got, want := bits.CRC8(0xC0C1, data[16:dataEnd]), binary.LittleEndian.Uint16(data[dataEnd:]); got != want {
		return nil, fmt.Errorf("header: CRC %04x, stored %04x", got, want)
	}
	r := bits.NewReader(data[:dataEnd])
	c := &codec{Version: v, text: text}
	if v.R2007 {
		// The RL bit size is counted from its own position (as in AcDb:Classes).
		bitsize := int(binary.LittleEndian.Uint32(data[dataStart:]))
		endbit := dataStart*8 + bitsize
		c.dat = r.Sub((dataStart+4)*8, endbit)
		c.str = c.dat
		str, datEnd := stringStream(r, endbit)
		if str != nil {
			c.str = str
		}
		c.dat.End = datEnd
		c.hdl = r.Sub(endbit, dataEnd*8)
	} else {
		c.dat = r.Sub(dataStart*8, dataEnd*8)
		c.str, c.hdl = c.dat, c.dat
	}
	if err := catch(func() { h.Vars.code(c); h.codeTrailer(c) }); err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	h.LeftData = c.dat.Remaining()
	if c.str != c.dat {
		h.LeftStr = c.str.Remaining()
	}
	if v.R2007 {
		h.LeftHandles = c.hdl.Remaining()
	}
	tail := c.hdl
	if n := tail.Remaining(); n > 0 && n < 8 {
		h.padBits = n
		h.pad = tail.Bits(n)
		if v.R2007 {
			h.LeftHandles = 0
		} else {
			h.LeftData = 0
		}
	}
	return h, nil
}

// Encode serializes the AcDb:Header section (sentinels, size, data, CRC).
func (h *Header) Encode(v Version, text TextCodec) []byte {
	c := &codec{Version: v, w: true, text: text, wd: &bits.Writer{}}
	if v.R2007 {
		c.ws, c.wh = &bits.Writer{}, &bits.Writer{}
	} else {
		c.ws, c.wh = c.wd, c.wd
	}
	h.Vars.code(c)
	h.codeTrailer(c)
	out := &bits.Writer{}
	if v.R2007 {
		out.RL(0) // bit size, patched below
		out.AppendWriter(c.wd)
		appendStringStream(out, c.ws)
		endbit := out.Len()
		out.AppendWriter(c.wh)
		h.finishPad(out)
		body := out.Bytes()
		binary.LittleEndian.PutUint32(body, uint32(endbit))
		return h.wrap(body)
	}
	out.AppendWriter(c.wd)
	h.finishPad(out)
	return h.wrap(out.Bytes())
}

func (h *Header) codeTrailer(c *codec) {
	for i := range h.Trailer {
		if c.w {
			c.wh.BS(h.Trailer[i])
		} else {
			h.Trailer[i] = c.hdl.BS()
		}
	}
}

func (h *Header) finishPad(out *bits.Writer) {
	if n := (8 - out.Len()%8) % 8; n > 0 && n == h.padBits {
		out.Bits(h.pad, n)
	}
	out.PadByte()
}

func (h *Header) wrap(body []byte) []byte {
	buf := append([]byte(nil), sentinelHeaderStart...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(body)))
	if h.HasSizeHi {
		buf = binary.LittleEndian.AppendUint32(buf, 0)
	}
	buf = append(buf, body...)
	buf = binary.LittleEndian.AppendUint16(buf, bits.CRC8(0xC0C1, buf[16:]))
	return append(buf, sentinelHeaderEnd...)
}

// appendStringStream appends an R2007+ string stream followed by its size and the presence flag (§20.1).
func appendStringStream(out, str *bits.Writer) {
	n := str.Len()
	if n == 0 {
		out.B(0)
		return
	}
	out.AppendWriter(str)
	if n >= 0x8000 {
		out.RS(int16(n >> 15))
		out.RS(int16(n&0x7FFF | 0x8000))
	} else {
		out.RS(int16(n))
	}
	out.B(1)
}

// codeBLL3 codes a bitlonglong whose byte count is a fixed 3-bit field. Real files store
// REQUIREDVERSIONS this way; §2.4 describes the length as a variable 1-3 bit 3B code.
func codeBLL3(c *codec, v *uint64) {
	if c.w {
		n := 0
		for x := *v; x != 0; x >>= 8 {
			n++
		}
		c.wd.Bits(uint64(n), 3)
		for i := 0; i < n; i++ {
			c.wd.RC(byte(*v >> (8 * uint(i))))
		}
		return
	}
	n := int(c.dat.Bits(3))
	*v = 0
	for i := 0; i < n; i++ {
		*v |= uint64(c.dat.RC()) << (8 * uint(i))
	}
}

// DXFIndex returns the DXF color number of a CMC color. R2004+ stores the color method in the
// high byte of RGB (0xC0 BYLAYER, 0xC1 BYBLOCK, 0xC2 true color, 0xC3 ACI in the low byte) and
// leaves Index 0; earlier files only set Index. True colors return their nearest ACI unknown: 0 is
// never produced for them, the caller writes group 420 with TrueColor instead.
func (col CmColor) DXFIndex() int {
	switch uint32(col.RGB) >> 24 {
	case 0xC0:
		return 256
	case 0xC1:
		return 0
	case 0xC3:
		return int(col.RGB & 0xFF)
	case 0xC2:
		return 256
	}
	if col.Index != 0 || col.RGB == 0 {
		return int(col.Index)
	}
	return 256
}

// TrueColor reports the 24-bit RGB of a true color CMC (method 0xC2).
func (col CmColor) TrueColor() (int32, bool) {
	if uint32(col.RGB)>>24 == 0xC2 {
		return col.RGB & 0xFFFFFF, true
	}
	return 0, false
}

// ACIColor returns a CMC color for an ACI number (256 BYLAYER, 0 BYBLOCK), as AutoCAD writes it.
func ACIColor(aci int) CmColor {
	switch aci {
	case 256:
		return CmColor{RGB: int32(-0x40000000)} // 0xC0000000
	case 0:
		return CmColor{RGB: int32(-0x3F000000)} // 0xC1000000
	}
	return CmColor{RGB: int32(-0x3D000000) | int32(aci&0xFF)} // 0xC3000000 | aci
}
