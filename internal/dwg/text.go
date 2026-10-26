package dwg

import (
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/korean"
)

// TextCodec converts R2004 8-bit strings in the drawing codepage.
type TextCodec struct {
	enc encoding.Encoding
	cp  int
}

// CodepageID is the DWG codepage number this codec was made for.
func (t TextCodec) CodepageID() int { return t.cp }

// codepages maps the DWG file header codepage number (offset 0x13) to its encoding and DXF name.
// Only entries verified against real drawings are listed; others fall back to Windows-1252.
var codepages = map[int]struct {
	enc  encoding.Encoding
	name string
}{
	30: {charmap.Windows1252, "ANSI_1252"},
	40: {korean.EUCKR, "ANSI_949"}, // WHATWG EUC-KR is Windows-949
}

func textCodecFor(cp int) TextCodec {
	if e, ok := codepages[cp]; ok {
		return TextCodec{e.enc, cp}
	}
	return TextCodec{charmap.Windows1252, cp}
}

// CodepageName returns the DXF $DWGCODEPAGE name for a header codepage number.
func CodepageName(cp int) string {
	if e, ok := codepages[cp]; ok {
		return e.name
	}
	return "ANSI_1252"
}

func (t TextCodec) Decode(b []byte) string {
	for len(b) > 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-1]
	}
	s, err := t.enc.NewDecoder().Bytes(b)
	if err != nil {
		return string(b)
	}
	return string(s)
}

func (t TextCodec) Encode(s string) []byte {
	if s == "" {
		return nil
	}
	b, err := t.enc.NewEncoder().Bytes([]byte(s))
	if err != nil {
		b = []byte(s)
	}
	return append(b, 0)
}

// TextCodecFor returns the codec for a DWG codepage number.
func TextCodecFor(cp int) TextCodec { return textCodecFor(cp) }

// CodepageNumber returns the DWG codepage number for a DXF $DWGCODEPAGE name (30 = ANSI_1252 if unknown).
func CodepageNumber(name string) int {
	for n, e := range codepages {
		if e.name == name {
			return n
		}
	}
	return 30
}
