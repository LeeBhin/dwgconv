package dwg

import (
	"encoding/binary"

	"dwgconv/internal/bits"
)

// Encode serializes the AcDb:Classes section (§10.2, §5.8):
// sentinel, RL byte size, [R2010+: RL 0, RL bit size], class data, [R2007+: string stream, size, flag],
// padding, RS CRC (seed 0xC0C1, from the size field on), end sentinel, 8 zero bytes.
func (t *ClassTable) Encode(v Version, text TextCodec) []byte { return t.encode(v, text, 0, 0) }

// ClassSection is a decoded AcDb:Classes section that remembers the original padding bits,
// so an unchanged table re-encodes byte-identically.
type ClassSection struct {
	*ClassTable
	pad     uint64
	padBits int
}

// DecodeClassSection decodes AcDb:Classes and captures its trailing padding.
func DecodeClassSection(data []byte, v Version, text TextCodec) (*ClassSection, error) {
	t, err := decodeClasses(data, v, text)
	if err != nil {
		return nil, err
	}
	cs := &ClassSection{ClassTable: t}
	bodyStart := 20
	if v.R2010 {
		bodyStart = 24
	}
	n := t.bodyBits(v, text)
	if pb := (8 - n%8) % 8; pb > 0 {
		r := bits.NewReader(data)
		r.Pos = bodyStart*8 + n
		cs.padBits, cs.pad = pb, r.Bits(pb)
	}
	return cs, nil
}

// Encode serializes the section, reusing the original padding when the length still matches.
func (cs *ClassSection) Encode(v Version, text TextCodec) []byte {
	return cs.ClassTable.encode(v, text, cs.pad, cs.padBits)
}

// bodyBits returns the unpadded bit length of the encoded body (from the first size-counted byte).
func (t *ClassTable) bodyBits(v Version, text TextCodec) int {
	out, _ := t.body(v, text)
	return out.Len()
}

func (t *ClassTable) body(v Version, text TextCodec) (*bits.Writer, int) {
	c := &codec{Version: v, w: true, text: text, wd: &bits.Writer{}}
	c.ws = c.wd
	if v.R2007 {
		c.ws = &bits.Writer{}
	}
	t.code(c)
	out := &bits.Writer{}
	if v.R2007 {
		out.RL(0) // bit size, counted from this field; patched below
	}
	out.AppendWriter(c.wd)
	if v.R2007 {
		appendStringStream(out, c.ws)
	}
	return out, out.Len()
}

func (t *ClassTable) encode(v Version, text TextCodec, pad uint64, padBits int) []byte {
	out, endbit := t.body(v, text)
	if n := (8 - endbit%8) % 8; n > 0 && n == padBits {
		out.Bits(pad, n)
	}
	out.PadByte()
	body := out.Bytes()
	if v.R2007 {
		binary.LittleEndian.PutUint32(body, uint32(endbit))
	}
	buf := append([]byte(nil), sentinelClassesStart...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(body)))
	if v.R2010 {
		buf = binary.LittleEndian.AppendUint32(buf, 0)
	}
	buf = append(buf, body...)
	buf = binary.LittleEndian.AppendUint16(buf, bits.CRC8(0xC0C1, buf[16:]))
	buf = append(buf, sentinelClassesEnd...)
	return append(buf, make([]byte, 8)...)
}

// DefaultClasses returns an empty class table; a from-scratch writer appends one Class per
// custom object type it emits (numbered from 500 in order, MaxNum = last number).
func DefaultClasses() *ClassTable { return &ClassTable{MaxNum: 499} }
