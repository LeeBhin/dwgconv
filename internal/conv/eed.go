// Package conv converts between decoded DWG documents and DXF.
package conv

import (
	"encoding/binary"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// eedToGroups decodes one application's extended data (§28) into DXF 1000-1071 groups.
func eedToGroups(data []byte, r2007 bool, text dwg.TextCodec) (out []dxf.Group, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	p := 0
	f64 := func() float64 {
		v := math.Float64frombits(binary.LittleEndian.Uint64(data[p:]))
		p += 8
		return v
	}
	for p < len(data) {
		code := int(data[p])
		p++
		switch {
		case code == 0:
			var s string
			if r2007 {
				n := int(binary.LittleEndian.Uint16(data[p:]))
				p += 2
				u := make([]uint16, n)
				for i := range u {
					u[i] = binary.LittleEndian.Uint16(data[p:])
					p += 2
				}
				s = string(utf16.Decode(u))
			} else {
				n := int(data[p])
				p += 3 // length byte + codepage short
				s = text.Decode(data[p : p+n])
				p += n
			}
			out = append(out, dxf.Group{Code: 1000, Value: s})
		case code == 2:
			v := "{"
			if data[p] == 1 {
				v = "}"
			}
			p++
			out = append(out, dxf.Group{Code: 1002, Value: v})
		case code == 3 || code == 5:
			h := binary.BigEndian.Uint64(data[p:])
			p += 8
			out = append(out, dxf.Group{Code: 1000 + code, Value: strings.ToUpper(strconv.FormatUint(h, 16))})
		case code == 4:
			n := int(data[p])
			p++
			out = append(out, dxf.Group{Code: 1004, Value: strings.ToUpper(hexString(data[p : p+n]))})
			p += n
		case code >= 10 && code <= 13:
			x, y, z := f64(), f64(), f64()
			c := 1000 + code
			out = append(out, dxf.Group{Code: c, Value: dxf.FormatFloat(x)},
				dxf.Group{Code: c + 10, Value: dxf.FormatFloat(y)},
				dxf.Group{Code: c + 20, Value: dxf.FormatFloat(z)})
		case code >= 40 && code <= 42:
			out = append(out, dxf.Group{Code: 1000 + code, Value: dxf.FormatFloat(f64())})
		case code == 70:
			v := int16(binary.LittleEndian.Uint16(data[p:]))
			p += 2
			out = append(out, dxf.Group{Code: 1070, Value: strconv.Itoa(int(v))})
		case code == 71:
			v := int32(binary.LittleEndian.Uint32(data[p:]))
			p += 4
			out = append(out, dxf.Group{Code: 1071, Value: strconv.Itoa(int(v))})
		default:
			return out, false
		}
	}
	return out, true
}

// groupsToEED encodes DXF xdata groups (after the 1001 app name) back to the EED byte form.
func groupsToEED(gs []dxf.Group, r2007 bool, text dwg.TextCodec) []byte {
	var out []byte
	for i := 0; i < len(gs); i++ {
		g := gs[i]
		code := g.Code - 1000
		switch {
		case code == 0:
			out = append(out, 0)
			if r2007 {
				u := utf16.Encode([]rune(g.Value))
				out = binary.LittleEndian.AppendUint16(out, uint16(len(u)))
				for _, c := range u {
					out = binary.LittleEndian.AppendUint16(out, c)
				}
			} else {
				b := text.Encode(g.Value)
				if len(b) > 0 && b[len(b)-1] == 0 {
					b = b[:len(b)-1]
				}
				out = append(out, byte(len(b)))
				out = binary.BigEndian.AppendUint16(out, uint16(text.CodepageID())) // big-endian (observed)
				out = append(out, b...)
			}
		case code == 2:
			v := byte(0)
			if strings.TrimSpace(g.Value) == "}" {
				v = 1
			}
			out = append(out, 2, v)
		case code == 3 || code == 5:
			h, _ := strconv.ParseUint(strings.TrimSpace(g.Value), 16, 64)
			out = append(out, byte(code))
			out = binary.BigEndian.AppendUint64(out, h)
		case code == 4:
			b := dxf.DecodeBinary(g.Value)
			out = append(out, 4, byte(len(b)))
			out = append(out, b...)
		case code >= 10 && code <= 13:
			out = append(out, byte(code))
			for k := 0; k < 3; k++ {
				v := 0.0
				if i+k < len(gs) {
					v = gs[i+k].Float()
				}
				out = binary.LittleEndian.AppendUint64(out, math.Float64bits(v))
			}
			i += 2
		case code >= 40 && code <= 42:
			out = append(out, byte(code))
			out = binary.LittleEndian.AppendUint64(out, math.Float64bits(g.Float()))
		case code == 70:
			out = append(out, 70)
			out = binary.LittleEndian.AppendUint16(out, uint16(g.Int()))
		case code == 71:
			out = append(out, 71)
			out = binary.LittleEndian.AppendUint32(out, uint32(g.Int()))
		}
	}
	return out
}

func hexString(b []byte) string {
	const digits = "0123456789ABCDEF"
	out := make([]byte, len(b)*2)
	for i, x := range b {
		out[2*i], out[2*i+1] = digits[x>>4], digits[x&0xF]
	}
	return string(out)
}
