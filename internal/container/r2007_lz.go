package container

import "fmt"

// R2007 (R21) LZ77 variant (§5.10).

// literalOrder lists, for a literal run of 1..32 bytes, the (length, source offset) blocks in
// target order (§5.10.1 table). Blocks are plain copies.
var literalOrder = [33][][2]int{
	1:  {{1, 0}},
	2:  {{1, 1}, {1, 0}},
	3:  {{1, 2}, {1, 1}, {1, 0}},
	4:  {{1, 0}, {1, 1}, {1, 2}, {1, 3}},
	5:  {{1, 4}, {4, 0}},
	6:  {{1, 5}, {4, 1}, {1, 0}},
	7:  {{2, 5}, {4, 1}, {1, 0}},
	8:  {{4, 0}, {4, 4}},
	9:  {{1, 8}, {8, 0}},
	10: {{1, 9}, {8, 1}, {1, 0}},
	11: {{2, 9}, {8, 1}, {1, 0}},
	12: {{4, 8}, {8, 0}},
	13: {{1, 12}, {4, 8}, {8, 0}},
	14: {{1, 13}, {4, 9}, {8, 1}, {1, 0}},
	15: {{2, 13}, {4, 9}, {8, 1}, {1, 0}},
	16: {{8, 8}, {8, 0}},
	17: {{8, 9}, {1, 8}, {8, 0}},
	18: {{1, 17}, {16, 1}, {1, 0}},
	19: {{3, 16}, {16, 0}},
	20: {{4, 16}, {16, 0}},
	21: {{1, 20}, {4, 16}, {16, 0}},
	22: {{2, 20}, {4, 16}, {16, 0}},
	23: {{3, 20}, {4, 16}, {16, 0}},
	24: {{8, 16}, {16, 0}},
	25: {{8, 17}, {1, 16}, {16, 0}},
	26: {{1, 25}, {8, 17}, {1, 16}, {16, 0}},
	27: {{2, 25}, {8, 17}, {1, 16}, {16, 0}},
	28: {{4, 24}, {8, 16}, {16, 0}},
	29: {{1, 28}, {4, 24}, {8, 16}, {16, 0}},
	30: {{2, 28}, {4, 24}, {8, 16}, {16, 0}},
	31: {{1, 30}, {4, 26}, {8, 18}, {16, 2}, {2, 0}},
	32: {{16, 16}, {16, 0}},
}

// copyLiteral appends n literal bytes from src in the R21 byte order.
func copyLiteral(dst, src []byte, n int) []byte {
	for n >= 32 {
		dst = appendBlocks(dst, src, literalOrder[32])
		src, n = src[32:], n-32
	}
	if n > 0 {
		dst = appendBlocks(dst, src, literalOrder[n])
	}
	return dst
}

// appendBlocks copies the blocks in order. A block of n > 1 bytes is itself copied with row n of
// the table (observed: the 2-, 3- and 16-byte blocks come out in their row's order).
func appendBlocks(dst, src []byte, blocks [][2]int) []byte {
	for _, b := range blocks {
		blk := src[b[1] : b[1]+b[0]]
		if b[0] == 1 || len(literalOrder[b[0]]) == 1 {
			dst = append(dst, blk...)
			continue
		}
		if isIdentityRow(b[0]) {
			dst = append(dst, blk...)
			continue
		}
		dst = appendBlocks(dst, blk, literalOrder[b[0]])
	}
	return dst
}

// isIdentityRow reports rows that copy their bytes in source order (4 and 8 bytes).
func isIdentityRow(n int) bool { return n == 4 || n == 8 }

// chunkPerm returns, for a literal chunk of n <= 32 bytes, the source index of each output byte.
func chunkPerm(n int) []int {
	idx := make([]byte, n)
	for i := range idx {
		idx[i] = byte(i)
	}
	out := appendBlocks(nil, idx, literalOrder[n])
	perm := make([]int, n)
	for i, v := range out {
		perm[i] = int(v)
	}
	return perm
}

// unorderLiteral is the inverse of copyLiteral: it returns the source bytes that copyLiteral
// turns into data.
func unorderLiteral(data []byte) []byte {
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += 32 {
		n := min(32, len(data)-i)
		for k, p := range chunkPerm(n) {
			out[i+p] = data[i+k]
		}
	}
	return out
}

func r21LiteralLength(src []byte, si *int, op byte) int {
	n := int(op) + 8
	if n == 0x17 {
		b := int(src[*si])
		*si++
		n += b
		if b == 0xFF {
			for {
				v := int(src[*si]) | int(src[*si+1])<<8
				*si += 2
				n += v
				if v != 0xFFFF {
					break
				}
			}
		}
	}
	return n
}

// r21Instructions decodes a back-reference instruction (§5.10.2).
func r21Instructions(src []byte, si *int, op *byte) (offset, length int) {
	o := *op
	switch o >> 4 {
	case 0:
		length = int(o&0xF) + 0x13
		offset = int(src[*si])
		*si++
		o = src[*si]
		*si++
		length += int(o>>3) & 0x10
		offset += int(o&0x78)<<5 + 1
	case 1:
		length = int(o&0xF) + 3
		offset = int(src[*si])
		*si++
		o = src[*si]
		*si++
		offset += int(o&0xF8)<<5 + 1
	case 2:
		offset = int(src[*si]) | int(src[*si+1])<<8
		*si += 2
		length = int(o & 7)
		if o&8 == 0 {
			o = src[*si]
			*si++
			length += int(o & 0xF8)
		} else {
			offset++
			length += int(src[*si]) << 3
			*si++
			o = src[*si]
			*si++
			length += int(o&0xF8)<<8 + 0x100
		}
	default:
		length = int(o >> 4)
		offset = int(o & 15)
		o = src[*si]
		*si++
		offset += int(o&0xF8)<<1 + 1
	}
	*op = o
	return
}

// r21Trace, when set, receives each decoding step (diagnostics).
var r21Trace func(string)

// Decompress21 inflates an R2007 compressed stream into outSize bytes.
func Decompress21(src []byte, outSize int) (out []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("lz21: corrupt stream: %v", r)
		}
	}()
	out = make([]byte, 0, outSize)
	si := 0
	length := 0
	op := src[0]
	if op>>4 == 2 {
		// the 3 bytes are counted after the opcode itself (observed; §5.10 reads as if from it)
		si = 4
		length = int(src[3] & 7)
	}
	for si < len(src) {
		if length == 0 {
			op = src[si]
			si++
			length = r21LiteralLength(src, &si, op)
		}
		if r21Trace != nil {
			r21Trace(fmt.Sprintf("@%d out=%d LIT %d % x", si, len(out), length, src[si:si+length]))
		}
		out = copyLiteral(out, src[si:], length)
		si += length
		length = 0
		if si >= len(src) {
			break
		}
		// back references
		op = src[si]
		si++
		offset, n := r21Instructions(src, &si, &op)
		for {
			if r21Trace != nil {
				r21Trace(fmt.Sprintf("@%d out=%d REF off=%d len=%d", si, len(out), offset, n))
			}
			s := len(out) - offset
			if s < 0 {
				return nil, fmt.Errorf("lz21: back reference before start")
			}
			for k := 0; k < n; k++ {
				out = append(out, out[s+k])
			}
			length = int(op & 7)
			if length != 0 || si >= len(src) {
				break
			}
			op = src[si]
			si++
			if op>>4 == 0 {
				si-- // a literal length opcode: re-read by the literal step
				break
			}
			if op>>4 == 15 {
				op &= 15
			}
			offset, n = r21Instructions(src, &si, &op)
		}
	}
	if len(out) > outSize {
		out = out[:outSize]
	}
	if len(out) != outSize {
		return out, fmt.Errorf("lz21: got %d bytes, want %d", len(out), outSize)
	}
	return out, nil
}
