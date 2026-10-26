package bits

import "encoding/binary"

// CRC-16 used across DWG (§2.14.1): reflected polynomial 0xA001.
var crc8Table [256]uint16

func init() {
	for i := range crc8Table {
		c := uint16(i)
		for k := 0; k < 8; k++ {
			if c&1 != 0 {
				c = c>>1 ^ 0xA001
			} else {
				c >>= 1
			}
		}
		crc8Table[i] = c
	}
}

func CRC8(seed uint16, p []byte) uint16 {
	dx := seed
	for _, b := range p {
		dx = dx>>8 ^ crc8Table[byte(dx)^b]
	}
	return dx
}

// MC reads a modular char (§2.6). signed applies the 0x40 negation on the last byte.
func MC(d []byte, p int, signed bool) (int64, int) {
	var v int64
	shift := 0
	for {
		b := d[p]
		p++
		if b&0x80 != 0 {
			v |= int64(b&0x7F) << shift
			shift += 7
			continue
		}
		if signed && b&0x40 != 0 {
			v |= int64(b&0x3F) << shift
			return -v, p
		}
		v |= int64(b&0x7F) << shift
		return v, p
	}
}

func PutMCUnsigned(dst []byte, v uint64) []byte {
	for {
		b := byte(v & 0x7F)
		v >>= 7
		if v != 0 {
			dst = append(dst, b|0x80)
			continue
		}
		return append(dst, b)
	}
}

func PutMCSigned(dst []byte, v int64) []byte {
	neg := v < 0
	if neg {
		v = -v
	}
	for v > 0x3F {
		dst = append(dst, byte(v&0x7F)|0x80)
		v >>= 7
	}
	b := byte(v)
	if neg {
		b |= 0x40
	}
	return append(dst, b)
}

// MS reads a modular short (§2.7).
func MS(d []byte, p int) (int, int) {
	v, shift := 0, 0
	for {
		w := binary.LittleEndian.Uint16(d[p:])
		p += 2
		v |= int(w&0x7FFF) << shift
		shift += 15
		if w&0x8000 == 0 {
			return v, p
		}
	}
}

func PutMS(dst []byte, v int) []byte {
	for {
		w := uint16(v & 0x7FFF)
		v >>= 15
		if v != 0 {
			w |= 0x8000
		}
		dst = binary.LittleEndian.AppendUint16(dst, w)
		if v == 0 {
			return dst
		}
	}
}
