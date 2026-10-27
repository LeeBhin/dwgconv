package container

// 64-bit CRCs of the R2007 format (§5.12): "normal" (ECMA-182, MSB first, result inverted) and
// "mirrored" (bit-reflected, not inverted). The byte processing order inside each 8-byte block
// follows the spec's table.

var crcNormalTable, crcMirrorTable [256]uint64

func init() {
	const poly = 0x42F0E1EBA9EA3693
	for i := 0; i < 256; i++ {
		c := uint64(i) << 56
		for k := 0; k < 8; k++ {
			if c&(1<<63) != 0 {
				c = c<<1 ^ poly
			} else {
				c <<= 1
			}
		}
		crcNormalTable[i] = c
	}
	const rpoly = 0x95AC9329AC4BC9B5 // reflected 0xAD93D23594C935A9 (the spec table matches this, not ECMA-182)
	for i := 0; i < 256; i++ {
		c := uint64(i)
		for k := 0; k < 8; k++ {
			if c&1 != 0 {
				c = c>>1 ^ rpoly
			} else {
				c >>= 1
			}
		}
		crcMirrorTable[i] = c
	}
}

// crcOrder is the byte order the CRC visits inside blocks of 1..8 bytes (§5.12 table).
// The table is recursive like the literal copy table: an 8-byte block is 4[4], 4[0], each
// 4-byte block 2[2], 2[0] (so 8 bytes are visited 6,7,4,5,2,3,0,1).
var crcOrder = [9][]int{
	1: {0},
	2: {0, 1},
	3: {0, 1, 2},
	4: {2, 3, 0, 1},
	5: {2, 3, 0, 1, 4},
	6: {2, 3, 0, 1, 4, 5},
	7: {2, 3, 0, 1, 4, 5, 6},
	8: {6, 7, 4, 5, 2, 3, 0, 1},
}

func crc64Bytes(crc uint64, data []byte, mirrored bool) uint64 {
	step := func(b byte) {
		if mirrored {
			crc = crcMirrorTable[byte(crc)^b] ^ crc>>8
		} else {
			crc = crcNormalTable[byte(crc>>56)^b] ^ crc<<8
		}
	}
	for len(data) >= 8 {
		for _, i := range crcOrder[8] {
			step(data[i])
		}
		data = data[8:]
	}
	if n := len(data); n > 0 {
		for _, i := range crcOrder[n] {
			step(data[i])
		}
	}
	return crc
}

// crc64Normal and crc64Mirrored compute the two CRC flavors from an initial value.
func crc64Normal(seed uint64, data []byte) uint64   { return ^crc64Bytes(seed, data, false) }
func crc64Mirrored(seed uint64, data []byte) uint64 { return crc64Bytes(seed, data, true) }

func updateSeed1(seed uint64, n uint32) uint64 {
	seed = (seed+uint64(n))*0x343FD + 0x269EC3
	seed |= seed*(0x343FD<<32) + (0x269EC3 << 32)
	return ^seed
}

func updateSeed2(seed uint64, n uint32) uint64 {
	seed = (seed+uint64(n))*0x343FD + 0x269EC3
	seed = seed*((1<<32)+0x343FD) + (uint64(n) + 0x269EC3)
	return ^seed
}

// pageChecksum is the 32-bit data page checksum of §5.4.1.
func pageChecksum21(seed uint64, data []byte) uint32 {
	seed = (seed+uint64(len(data)))*0x343FD + 0x269EC3
	s1, s2 := uint32(seed&0xFFFF), uint32((seed>>16)&0xFFFF)
	add := func(b byte) { s1 += uint32(b); s2 += s1 }
	add2 := func(p []byte) { add(p[0]); add(p[1]) }
	for len(data) > 0 {
		n := min(0x15B0, len(data))
		chunk := data[:n]
		data = data[n:]
		for len(chunk) >= 8 {
			add2(chunk[6:])
			add2(chunk[4:])
			add2(chunk[2:])
			add2(chunk[0:])
			chunk = chunk[8:]
		}
		switch len(chunk) {
		case 1:
			add(chunk[0])
		case 2:
			add2(chunk)
		case 3:
			add2(chunk)
			add(chunk[2])
		case 4:
			add2(chunk[2:])
			add2(chunk)
		case 5:
			add2(chunk[2:])
			add2(chunk)
			add(chunk[4])
		case 6:
			add2(chunk[2:])
			add2(chunk)
			add2(chunk[4:])
		case 7:
			add2(chunk[2:])
			add2(chunk)
			add2(chunk[4:])
			add(chunk[6])
		}
		s1 %= 0xFFF1
		s2 %= 0xFFF1
	}
	return s2<<16 | s1&0xFFFF
}
