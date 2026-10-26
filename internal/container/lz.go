package container

import (
	"errors"
	"fmt"
)

// R2004 LZ77 variant (§4.7).

func literalLength(src []byte, p int) (n, next int, opcode int) {
	b := src[p]
	p++
	switch {
	case b >= 0x01 && b <= 0x0F:
		return int(b) + 3, p, -1
	case b == 0:
		total := 0x0F
		for src[p] == 0 {
			total += 0xFF
			p++
		}
		total += int(src[p])
		return total + 3, p + 1, -1
	default:
		return 0, p, int(b)
	}
}

func longOffset(src []byte, p int) (int, int) {
	b := src[p]
	p++
	if b != 0 {
		return int(b), p
	}
	total := 0xFF
	for src[p] == 0 {
		total += 0xFF
		p++
	}
	return total + int(src[p]), p + 1
}

func twoByteOffset(src []byte, p int) (off, lit, next int) {
	f := src[p]
	return int(f>>2) | int(src[p+1])<<6, int(f & 3), p + 2
}

// Decompress inflates src into exactly outSize bytes.
func Decompress(src []byte, outSize int) (out []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("lz: corrupt stream: %v", r)
		}
	}()
	out = make([]byte, 0, outSize)
	lit, p, op := literalLength(src, 0)
	out = append(out, src[p:p+lit]...)
	p += lit
	for p < len(src) || op >= 0 {
		if op < 0 {
			op = int(src[p])
			p++
		}
		if op == 0x11 {
			break
		}
		var cb, off, lc int
		switch {
		case op >= 0x40:
			cb = (op >> 4) - 1
			off = int(src[p])<<2 | (op&0x0C)>>2
			p++
			lc = op & 3
		case op == 0x10:
			cb, p = longOffset(src, p)
			cb += 9
			off, lc, p = twoByteOffset(src, p)
			off += 0x3FFF
		case op >= 0x12 && op <= 0x1F:
			cb = op&0x0F + 2
			off, lc, p = twoByteOffset(src, p)
			off += 0x3FFF
		case op == 0x20:
			cb, p = longOffset(src, p)
			cb += 0x21
			off, lc, p = twoByteOffset(src, p)
		case op >= 0x21 && op <= 0x3F:
			cb = op - 0x1E
			off, lc, p = twoByteOffset(src, p)
		default:
			return nil, fmt.Errorf("lz: bad opcode %#x", op)
		}
		op = -1
		if lc == 0 {
			lc, p, op = literalLength(src, p)
		}
		s := len(out) - off - 1
		if s < 0 {
			return nil, errors.New("lz: back reference before start")
		}
		for k := 0; k < cb; k++ {
			out = append(out, out[s+k])
		}
		out = append(out, src[p:p+lc]...)
		p += lc
	}
	if len(out) != outSize {
		return nil, fmt.Errorf("lz: got %d bytes, want %d", len(out), outSize)
	}
	return out, nil
}

// --- compression ---

func putLiteralLength(dst []byte, n int) []byte {
	// n >= 4
	if n <= 0x12 {
		return append(dst, byte(n-3))
	}
	r := n - 3 - 0x0F
	dst = append(dst, 0)
	for r > 0xFF {
		dst = append(dst, 0)
		r -= 0xFF
	}
	return append(dst, byte(r))
}

func putLongOffset(dst []byte, v int) []byte {
	// v >= 1
	if v <= 0xFF {
		return append(dst, byte(v))
	}
	r := v - 0xFF
	dst = append(dst, 0)
	for r > 0xFF {
		dst = append(dst, 0)
		r -= 0xFF
	}
	return append(dst, byte(r))
}

const (
	minMatch  = 3
	maxOffset = 0x3FFF + 0x3FFF // longest encodable back distance - 1
	hashBits  = 14
	maxChain  = 32
	maxMatch  = 0x1000
)

// putMatch emits the opcode for a match of length cb at back distance dist (>=1),
// followed by litCount literals from lits.
func putMatch(dst []byte, cb, dist int, lits []byte) []byte {
	off := dist - 1
	lc := len(lits)
	code := 0
	if lc >= 1 && lc <= 3 {
		code = lc
	}
	switch {
	case off <= 0x3FF && cb <= 14:
		dst = append(dst, byte((cb+1)<<4|(off&3)<<2|code), byte(off>>2))
	case off <= 0x3FFF:
		if cb <= 0x21 {
			dst = append(dst, byte(cb+0x1E))
		} else {
			dst = append(dst, 0x20)
			dst = putLongOffset(dst, cb-0x21)
		}
		dst = append(dst, byte((off&0x3F)<<2|code), byte(off>>6))
	default:
		o := off - 0x3FFF
		// The spec allows 0x12-0x1F here, but AutoCAD only emits 0x12-0x17 and readers
		// treat bit 3 of the opcode as part of the offset, so lengths above 9 use 0x10.
		if cb <= 9 {
			dst = append(dst, byte(0x10|(cb-2)))
		} else {
			dst = append(dst, 0x10)
			dst = putLongOffset(dst, cb-9)
		}
		dst = append(dst, byte((o&0x3F)<<2|code), byte(o>>6))
	}
	if lc >= 4 {
		dst = putLiteralLength(dst, lc)
	}
	return append(dst, lits...)
}

func hash3(b []byte, i int) int {
	v := uint32(b[i]) | uint32(b[i+1])<<8 | uint32(b[i+2])<<16
	return int((v * 2654435761) >> (32 - hashBits))
}

// Compress produces a stream that Decompress inflates back to src. len(src) must be >= 4.
func Compress(src []byte) []byte {
	n := len(src)
	head := make([]int, 1<<hashBits)
	for i := range head {
		head[i] = -1
	}
	prev := make([]int, n)
	insert := func(i int) {
		if i+2 < n {
			h := hash3(src, i)
			prev[i] = head[h]
			head[h] = i
		}
	}
	type match struct{ pos, cb, dist, litStart int }
	var matches []match
	litStart := 0
	i := 0
	for i < n {
		bestLen, bestDist := 0, 0
		// The first literal run must be >= 4 bytes, so matching starts at offset 4.
		if i >= 4 && i+minMatch <= n {
			h := hash3(src, i)
			for c, k := head[h], 0; c >= 0 && k < maxChain; c, k = prev[c], k+1 {
				d := i - c
				if d > maxOffset+1 {
					break
				}
				l := 0
				for i+l < n && l < maxMatch && src[c+l] == src[i+l] {
					l++
				}
				if d-1 > 0x3FFF && l < 4 {
					continue // far form can't encode length 3
				}
				if l > bestLen {
					bestLen, bestDist = l, d
				}
			}
		}
		if bestLen >= minMatch {
			matches = append(matches, match{pos: i, cb: bestLen, dist: bestDist, litStart: litStart})
			for k := 0; k < bestLen; k++ {
				insert(i + k)
			}
			i += bestLen
			litStart = i
			continue
		}
		insert(i)
		i++
	}
	out := make([]byte, 0, n/2+16)
	firstEnd := n
	if len(matches) > 0 {
		firstEnd = matches[0].pos
	}
	out = putLiteralLength(out, firstEnd)
	out = append(out, src[:firstEnd]...)
	for k, m := range matches {
		end := n
		if k+1 < len(matches) {
			end = matches[k+1].pos
		}
		out = putMatch(out, m.cb, m.dist, src[m.pos+m.cb:end])
	}
	return append(out, 0x11)
}
