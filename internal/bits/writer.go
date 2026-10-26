package bits

import (
	"encoding/binary"
	"math"
	"unicode/utf16"
)

// Writer writes MSB-first bits.
type Writer struct {
	buf []byte
	n   int // bits written
}

func (w *Writer) Len() int { return w.n }

func (w *Writer) Bytes() []byte { return w.buf[:(w.n+7)/8] }

func (w *Writer) B(v byte) {
	if w.n&7 == 0 {
		w.buf = append(w.buf, 0)
	}
	if v&1 != 0 {
		w.buf[w.n>>3] |= 1 << (7 - uint(w.n&7))
	}
	w.n++
}

func (w *Writer) Bool(v bool) {
	if v {
		w.B(1)
	} else {
		w.B(0)
	}
}

func (w *Writer) Bits(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		w.B(byte(v >> uint(i)))
	}
}

func (w *Writer) BB(v byte) { w.Bits(uint64(v), 2) }

// B3 writes 1-3 bits so that Reader.B3 returns v (v in 0..7 of the forms 0,10,110,111).
func (w *Writer) B3(v byte) {
	switch v {
	case 0:
		w.Bits(0, 1)
	case 2:
		w.Bits(2, 2)
	case 6:
		w.Bits(6, 3)
	case 7:
		w.Bits(7, 3)
	default:
		panic("bits: B3 value not representable")
	}
}

func (w *Writer) RC(v byte) {
	if w.n&7 == 0 {
		w.buf = append(w.buf, v)
		w.n += 8
		return
	}
	w.Bits(uint64(v), 8)
}

func (w *Writer) Raw(b []byte) {
	for _, x := range b {
		w.RC(x)
	}
}

func (w *Writer) RS(v int16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], uint16(v))
	w.Raw(b[:])
}

func (w *Writer) RL(v int32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(v))
	w.Raw(b[:])
}

func (w *Writer) RD(v float64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], math.Float64bits(v))
	w.Raw(b[:])
}

func (w *Writer) BS(v int16) {
	switch {
	case v == 0:
		w.BB(2)
	case v == 256:
		w.BB(3)
	case v > 0 && v < 256:
		w.BB(1)
		w.RC(byte(v))
	default:
		w.BB(0)
		w.RS(v)
	}
}

func (w *Writer) BL(v int32) {
	switch {
	case v == 0:
		w.BB(2)
	case v > 0 && v < 256:
		w.BB(1)
		w.RC(byte(v))
	default:
		w.BB(0)
		w.RL(v)
	}
}

// BLL writes the length in 3B form; supported lengths 0,2,6,7 (enough for any value we emit; widen with zero bytes).
func (w *Writer) BLL(v uint64) {
	n := 0
	for x := v; x != 0; x >>= 8 {
		n++
	}
	switch {
	case n == 0:
	case n <= 2:
		n = 2
	case n <= 6:
		n = 6
	default:
		n = 7
		if v>>56 != 0 {
			panic("bits: BLL too large")
		}
	}
	w.B3(byte(n))
	for i := 0; i < n; i++ {
		w.RC(byte(v >> (8 * uint(i))))
	}
}

func (w *Writer) BD(v float64) {
	switch {
	case v == 1.0:
		w.BB(1)
	case v == 0.0 && !math.Signbit(v):
		w.BB(2)
	default:
		w.BB(0)
		w.RD(v)
	}
}

// DD writes a bitdouble with default (§2.9), choosing the shortest exact form.
func (w *Writer) DD(v, def float64) {
	vb, db := math.Float64bits(v), math.Float64bits(def)
	switch {
	case vb == db:
		w.BB(0)
	case vb>>32 == db>>32:
		w.BB(1)
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], vb)
		w.Raw(b[0:4])
	case vb>>48 == db>>48:
		w.BB(2)
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], vb)
		w.Raw(b[4:6])
		w.Raw(b[0:4])
	default:
		w.BB(3)
		w.RD(v)
	}
}

func (w *Writer) BT(v float64) {
	if v == 0 {
		w.B(1)
		return
	}
	w.B(0)
	w.BD(v)
}

func (w *Writer) BE(v [3]float64) {
	if v == [3]float64{0, 0, 1} {
		w.B(1)
		return
	}
	w.B(0)
	w.BD(v[0])
	w.BD(v[1])
	w.BD(v[2])
}

func (w *Writer) H(h Handle) {
	n := 0
	for x := h.Value; x != 0; x >>= 8 {
		n++
	}
	w.Bits(uint64(h.Code), 4)
	w.Bits(uint64(n), 4)
	for i := n - 1; i >= 0; i-- {
		w.RC(byte(h.Value >> (8 * uint(i))))
	}
}

func (w *Writer) OT(t int, r2010 bool) {
	if !r2010 {
		w.BS(int16(uint16(t)))
		return
	}
	switch {
	case t < 0x100:
		w.BB(0)
		w.RC(byte(t))
	case t >= 0x1F0 && t < 0x2F0:
		w.BB(1)
		w.RC(byte(t - 0x1F0))
	default:
		w.BB(2)
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], uint16(t))
		w.Raw(b[:])
	}
}

func (w *Writer) TV8(b []byte) {
	w.BS(int16(len(b)))
	w.Raw(b)
}

// TU writes BS character count then UTF-16LE. Non-empty strings get a terminating NUL counted in the length.
func (w *Writer) TU(s string) {
	if s == "" {
		w.BS(0)
		return
	}
	u := append(utf16.Encode([]rune(s)), 0)
	w.BS(int16(len(u)))
	for _, c := range u {
		w.RS(int16(c))
	}
}

// AppendBits copies n bits starting at bit position start of src.
func (w *Writer) AppendBits(src []byte, start, n int) {
	for p := start; p < start+n; p++ {
		w.B((src[p>>3] >> (7 - uint(p&7))) & 1)
	}
}

// AppendWriter copies all bits of other.
func (w *Writer) AppendWriter(o *Writer) { w.AppendBits(o.buf, 0, o.n) }

// PadByte pads with zero bits to the next byte boundary.
func (w *Writer) PadByte() {
	for w.n&7 != 0 {
		w.B(0)
	}
}
