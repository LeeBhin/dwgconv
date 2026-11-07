package dxf

import (
	"bytes"
	"testing"
)

// TOLERANCE text carries a line break; written raw it split the value and the next read failed
// with "bad group code".
func TestCaretRoundTrip(t *testing.T) {
	cases := []struct{ in, out string }{
		{"{\\Fgdt;r}%%v1%%v2%%v%%v%%v\n10", "{\\Fgdt;r}%%v1%%v2%%v%%v%%v^J10"}, // same as ODA
		{"a\tb\rc", "a^Ib^Mc"},
		{"\\S1^2;", "\\S1^2;"}, // MTEXT stack: unchanged
		{"x^Jy", "x^ Jy"},      // a literal ^J must not read back as a newline
		{"^ ", "^  "},
		{"end^", "end^"},
	}
	for _, c := range cases {
		if got := encodeCaret(c.in); got != c.out {
			t.Errorf("encodeCaret(%q) = %q, want %q", c.in, got, c.out)
		}
		if got := decodeCaret(c.out); got != c.in {
			t.Errorf("decodeCaret(%q) = %q, want %q", c.out, got, c.in)
		}
	}

	f := &File{Sections: []*Section{{Name: "ENTITIES", Records: []*Record{
		{Type: "TOLERANCE", Groups: []Group{{1, "a\nb"}, {11, "1.0"}}},
	}}}}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	g, err := Parse(&buf)
	if err != nil {
		t.Fatal(err)
	}
	got := g.Section("ENTITIES").Records[0].Groups
	if len(got) != 2 || got[0].Value != "a\nb" || got[1].Code != 11 {
		t.Fatalf("round trip = %+v", got)
	}
}
