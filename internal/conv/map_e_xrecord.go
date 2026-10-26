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

// XRECORD data (§20.4.104) is a byte stream of RS group code + value, the value encoded by the code's type.

type eKind int

const (
	eStr eKind = iota
	ePoint
	eDouble
	eInt8
	eInt16
	eInt32
	eInt64
	eBool
	eBinary
	eHandle
)

// eKindOf classifies DXF group codes by value type (standard DXF group code ranges).
func eKindOf(c int) eKind {
	switch {
	case c >= 0 && c <= 9, c == 100, c == 101, c == 102, c >= 300 && c <= 309, c >= 410 && c <= 419,
		c >= 430 && c <= 439, c >= 470 && c <= 479, c == 999, c >= 1000 && c <= 1009:
		return eStr
	case c >= 10 && c <= 18, c >= 110 && c <= 112, c >= 210 && c <= 219, c >= 1010 && c <= 1013:
		return ePoint
	case c >= 19 && c <= 59, c >= 113 && c <= 149, c >= 220 && c <= 239, c >= 460 && c <= 469, c >= 1014 && c <= 1059:
		return eDouble
	case c >= 60 && c <= 79, c >= 170 && c <= 179, c >= 270 && c <= 279, c >= 370 && c <= 389,
		c >= 400 && c <= 409, c >= 1060 && c <= 1070:
		return eInt16
	case c >= 90 && c <= 99, c >= 420 && c <= 429, c >= 440 && c <= 459, c == 1071:
		return eInt32
	case c >= 160 && c <= 169:
		return eInt64
	case c >= 280 && c <= 289:
		return eInt8
	case c >= 290 && c <= 299:
		return eBool
	case c >= 310 && c <= 319, c == 1004:
		return eBinary
	case c == 105, c >= 320 && c <= 369, c >= 390 && c <= 399, c >= 480 && c <= 481:
		return eHandle
	}
	return eStr
}

// eXRecordGroups decodes the XRECORD byte stream into DXF groups.
func eXRecordGroups(data []byte, r2007 bool, text dwg.TextCodec) (out []dxf.Group) {
	defer func() { recover() }() // truncated data: keep what was decoded
	p := 0
	f64 := func() string {
		v := math.Float64frombits(binary.LittleEndian.Uint64(data[p:]))
		p += 8
		return dxf.FormatFloat(v)
	}
	for p+2 <= len(data) {
		c := int(int16(binary.LittleEndian.Uint16(data[p:])))
		p += 2
		switch eKindOf(c) {
		case eStr:
			n := int(binary.LittleEndian.Uint16(data[p:]))
			p += 2
			var s string
			if r2007 {
				u := make([]uint16, n)
				for i := range u {
					u[i] = binary.LittleEndian.Uint16(data[p:])
					p += 2
				}
				s = string(utf16.Decode(u))
			} else {
				p++ // codepage
				s = text.Decode(data[p : p+n])
				p += n
			}
			out = append(out, dxf.Group{Code: c, Value: s})
		case ePoint:
			out = append(out, dxf.Group{Code: c, Value: f64()}, dxf.Group{Code: c + 10, Value: f64()},
				dxf.Group{Code: c + 20, Value: f64()})
		case eDouble:
			out = append(out, dxf.Group{Code: c, Value: f64()})
		case eInt8, eBool:
			out = append(out, dxf.Group{Code: c, Value: strconv.Itoa(int(data[p]))})
			p++
		case eInt16:
			out = append(out, dxf.Group{Code: c, Value: strconv.Itoa(int(int16(binary.LittleEndian.Uint16(data[p:]))))})
			p += 2
		case eInt32:
			out = append(out, dxf.Group{Code: c, Value: strconv.Itoa(int(int32(binary.LittleEndian.Uint32(data[p:]))))})
			p += 4
		case eInt64:
			out = append(out, dxf.Group{Code: c, Value: strconv.FormatInt(int64(binary.LittleEndian.Uint64(data[p:])), 10)})
			p += 8
		case eBinary:
			n := int(data[p])
			p++
			out = append(out, dxf.Group{Code: c, Value: hexString(data[p : p+n])})
			p += n
		case eHandle:
			h := binary.LittleEndian.Uint64(data[p:])
			p += 8
			out = append(out, dxf.Group{Code: c, Value: strings.ToUpper(strconv.FormatUint(h, 16))})
		}
	}
	return out
}

// eXRecordData encodes DXF groups back to the XRECORD byte stream.
func eXRecordData(gs []dxf.Group, r2007 bool, text dwg.TextCodec) []byte {
	var out []byte
	for i := 0; i < len(gs); i++ {
		g := gs[i]
		c := g.Code
		out = binary.LittleEndian.AppendUint16(out, uint16(int16(c)))
		switch eKindOf(c) {
		case eStr:
			if r2007 {
				u := utf16.Encode([]rune(g.Value))
				out = binary.LittleEndian.AppendUint16(out, uint16(len(u)))
				for _, x := range u {
					out = binary.LittleEndian.AppendUint16(out, x)
				}
			} else {
				b := text.Encode(g.Value)
				if len(b) > 0 && b[len(b)-1] == 0 {
					b = b[:len(b)-1]
				}
				out = binary.LittleEndian.AppendUint16(out, uint16(len(b)))
				out = append(out, byte(text.CodepageID()))
				out = append(out, b...)
			}
		case ePoint:
			for k := 0; k < 3; k++ {
				v := 0.0
				if i+k < len(gs) && gs[i+k].Code == c+10*k {
					v = gs[i+k].Float()
				}
				out = binary.LittleEndian.AppendUint64(out, math.Float64bits(v))
			}
			i += 2
		case eDouble:
			out = binary.LittleEndian.AppendUint64(out, math.Float64bits(g.Float()))
		case eInt8, eBool:
			out = append(out, byte(g.Int()))
		case eInt16:
			out = binary.LittleEndian.AppendUint16(out, uint16(g.Int()))
		case eInt32:
			out = binary.LittleEndian.AppendUint32(out, uint32(g.Int()))
		case eInt64:
			out = binary.LittleEndian.AppendUint64(out, uint64(g.Int()))
		case eBinary:
			b := dxf.DecodeBinary(g.Value)
			out = append(out, byte(len(b)))
			out = append(out, b...)
		case eHandle:
			out = binary.LittleEndian.AppendUint64(out, g.Handle())
		}
	}
	return out
}
