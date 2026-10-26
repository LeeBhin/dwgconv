// Package bits implements the DWG bit-level codes (Open Design Specification §2).
package bits

import (
	"encoding/binary"
	"errors"
	"math"
	"unicode/utf16"
)

// ErrOverrun is returned (via panic recovery in callers) when a read passes the stream end.
var ErrOverrun = errors.New("bits: read past end of stream")

// Handle is a handle reference (§2.13): CODE.COUNTER.BYTES.
type Handle struct {
	Code  byte
	Value uint64
}

// Reader reads MSB-first bits from a byte slice.
type Reader struct {
	data []byte
	Pos  int // bit position
	End  int // bit limit (exclusive)
}

func NewReader(data []byte) *Reader { return &Reader{data: data, End: len(data) * 8} }

// Sub returns a reader over the same bytes limited to [start,end) bits.
func (r *Reader) Sub(start, end int) *Reader { return &Reader{data: r.data, Pos: start, End: end} }

func (r *Reader) Remaining() int { return r.End - r.Pos }

func (r *Reader) B() byte {
	if r.Pos >= r.End {
		panic(ErrOverrun)
	}
	v := (r.data[r.Pos>>3] >> (7 - uint(r.Pos&7))) & 1
	r.Pos++
	return v
}

func (r *Reader) Bits(n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = v<<1 | uint64(r.B())
	}
	return v
}

func (r *Reader) Bool() bool { return r.B() == 1 }
func (r *Reader) BB() byte   { return byte(r.Bits(2)) }

// B3 reads 1-3 bits (§2.1).
func (r *Reader) B3() byte {
	var v byte
	for i := 0; i < 3; i++ {
		b := r.B()
		v = v<<1 | b
		if b == 0 {
			break
		}
	}
	return v
}

func (r *Reader) RC() byte { return byte(r.Bits(8)) }

func (r *Reader) Raw(n int) []byte {
	out := make([]byte, n)
	if r.Pos&7 == 0 && r.Pos+n*8 <= r.End {
		copy(out, r.data[r.Pos>>3:])
		r.Pos += n * 8
		return out
	}
	for i := range out {
		out[i] = r.RC()
	}
	return out
}

func (r *Reader) RS() int16   { return int16(binary.LittleEndian.Uint16(r.Raw(2))) }
func (r *Reader) RL() int32   { return int32(binary.LittleEndian.Uint32(r.Raw(4))) }
func (r *Reader) RD() float64 { return math.Float64frombits(binary.LittleEndian.Uint64(r.Raw(8))) }

// BS bitshort (§2.2).
func (r *Reader) BS() int16 {
	switch r.BB() {
	case 0:
		return r.RS()
	case 1:
		return int16(r.RC())
	case 2:
		return 0
	default:
		return 256
	}
}

// BL bitlong (§2.3).
func (r *Reader) BL() int32 {
	switch r.BB() {
	case 0:
		return r.RL()
	case 1:
		return int32(r.RC())
	case 2:
		return 0
	default:
		panic(errors.New("bits: BL code 11"))
	}
}

// BLL bitlonglong (§2.4).
func (r *Reader) BLL() uint64 {
	n := int(r.B3())
	var v uint64
	for i := 0; i < n; i++ {
		v |= uint64(r.RC()) << (8 * i)
	}
	return v
}

// BD bitdouble (§2.5).
func (r *Reader) BD() float64 {
	switch r.BB() {
	case 0:
		return r.RD()
	case 1:
		return 1.0
	case 2:
		return 0.0
	default:
		panic(errors.New("bits: BD code 11"))
	}
}

// DD bitdouble with default (§2.9).
func (r *Reader) DD(def float64) float64 {
	switch r.BB() {
	case 0:
		return def
	case 3:
		return r.RD()
	case 1:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], math.Float64bits(def))
		copy(b[0:4], r.Raw(4))
		return math.Float64frombits(binary.LittleEndian.Uint64(b[:]))
	default:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], math.Float64bits(def))
		copy(b[4:6], r.Raw(2))
		copy(b[0:4], r.Raw(4))
		return math.Float64frombits(binary.LittleEndian.Uint64(b[:]))
	}
}

// BT bit thickness (§2.10, R2000+).
func (r *Reader) BT() float64 {
	if r.Bool() {
		return 0
	}
	return r.BD()
}

// BE bit extrusion (§2.8, R2000+).
func (r *Reader) BE() [3]float64 {
	if r.Bool() {
		return [3]float64{0, 0, 1}
	}
	return [3]float64{r.BD(), r.BD(), r.BD()}
}

// H handle reference (§2.13).
func (r *Reader) H() Handle {
	code := byte(r.Bits(4))
	n := int(r.Bits(4))
	var v uint64
	for i := 0; i < n; i++ {
		v = v<<8 | uint64(r.RC())
	}
	return Handle{Code: code, Value: v}
}

// OT object type (§2.12). R2010+ uses the bit-pair form, earlier versions a BS.
func (r *Reader) OT(r2010 bool) int {
	if !r2010 {
		return int(uint16(r.BS()))
	}
	switch r.BB() {
	case 0:
		return int(r.RC())
	case 1:
		return int(r.RC()) + 0x1F0
	default:
		return int(binary.LittleEndian.Uint16(r.Raw(2)))
	}
}

// TV8 reads an 8-bit text (R2004 and earlier): BS length then bytes in the drawing codepage.
func (r *Reader) TV8() []byte {
	n := int(uint16(r.BS()))
	return r.Raw(n)
}

// TU reads a Unicode text (R2007+): BS character count then UTF-16LE.
func (r *Reader) TU() string {
	n := int(uint16(r.BS()))
	u := make([]uint16, n)
	for i := range u {
		u[i] = uint16(r.RS())
	}
	for len(u) > 0 && u[len(u)-1] == 0 {
		u = u[:len(u)-1]
	}
	return string(utf16.Decode(u))
}
