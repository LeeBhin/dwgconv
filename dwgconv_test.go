package dwgconv

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/LeeBhin/dwgconv/internal/container"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

// TestRoundTrip converts a sample DWG to DXF and back, for every output version, and checks the
// result reads back (samples in .work/orig, not part of the repository).
func TestRoundTrip(t *testing.T) {
	in, err := os.ReadFile(".work/orig/s2.dwg")
	if err != nil {
		t.Skip("sample not present:", err)
	}
	var dxfBuf bytes.Buffer
	if err := DWGToDXF(bytes.NewReader(in), &dxfBuf); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"", "R2004", "R2007", "R2010", "R2013", "R2018"} {
		out, _, err := DXFToDWG(bytes.NewReader(dxfBuf.Bytes()), v)
		if err != nil {
			t.Fatalf("%q: %v", v, err)
		}
		if _, err := container.Read(out); err != nil {
			t.Errorf("%q: written DWG does not read back: %v", v, err)
		}
	}
	if _, _, err := DXFToDWG(bytes.NewReader(dxfBuf.Bytes()), "R12"); err == nil {
		t.Error("unsupported version accepted")
	}
}

// TestEntityWithoutOwner covers entities written by editors without an owner (330): they belong to
// model space, as AutoCAD reads them (they used to be dropped).
func TestEntityWithoutOwner(t *testing.T) {
	in, err := os.ReadFile(".work/orig/s2.dwg")
	if err != nil {
		t.Skip("sample not present:", err)
	}
	var dxfBuf bytes.Buffer
	if err := DWGToDXF(bytes.NewReader(in), &dxfBuf); err != nil {
		t.Fatal(err)
	}
	line := "  0\r\nLINE\r\n  5\r\nFFFF0\r\n100\r\nAcDbEntity\r\n  8\r\n0\r\n100\r\nAcDbLine\r\n" +
		" 10\r\n1.5\r\n 20\r\n2.5\r\n 30\r\n0\r\n 11\r\n3.5\r\n 21\r\n4.5\r\n 31\r\n0\r\n"
	src := dxfBuf.String()
	i := strings.Index(src, "ENTITIES\r\n")
	if i < 0 {
		t.Fatal("no ENTITIES section")
	}
	i += len("ENTITIES\r\n")
	edited := src[:i] + line + src[i:]
	out, warnings, err := DXFToDWG(strings.NewReader(edited), "R2018")
	if err != nil {
		t.Fatal(err)
	}
	var back bytes.Buffer
	if err := DWGToDXF(bytes.NewReader(out), &back); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(back.String(), "\r\n  5\r\nFFFF0\r\n") {
		t.Errorf("entity without owner was dropped (warnings %v)", warnings)
	}
}

// TestMinimalDXF covers a DXF with only an ENTITIES section (no handles, owners, tables or blocks):
// it is completed with the default drawing, and an unknown layer is created.
func TestMinimalDXF(t *testing.T) {
	src := "0\r\nSECTION\r\n2\r\nENTITIES\r\n0\r\nLINE\r\n8\r\nEDIT\r\n10\r\n0.0\r\n20\r\n0.0\r\n11\r\n100.0\r\n21\r\n0.0\r\n0\r\nENDSEC\r\n0\r\nEOF\r\n"
	for _, v := range []string{"R2004", "R2007", "R2010", "R2013", "R2018"} {
		out, warnings, err := DXFToDWG(strings.NewReader(src), v)
		if err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		var back bytes.Buffer
		if err := DWGToDXF(bytes.NewReader(out), &back); err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		got := back.String()
		if !strings.Contains(got, "\r\nLINE\r\n") || !strings.Contains(got, "\r\nEDIT\r\n") {
			t.Errorf("%s: line or its layer missing after round trip (warnings %v)", v, warnings)
		}
	}
}

// TestTablesWithoutBlockRecords covers a hand-written R12-style DXF: a TABLES section (LAYER and
// LTYPE, no handles) but no BLOCK_RECORD table, no OBJECTS and no entity handles. Every entity must
// survive, in model space or (67=1) paper space, and block B1 must get a block record.
func TestTablesWithoutBlockRecords(t *testing.T) {
	src := strings.Join([]string{
		"0", "SECTION", "2", "HEADER", "9", "$ACADVER", "1", "AC1009", "0", "ENDSEC",
		"0", "SECTION", "2", "TABLES",
		"0", "TABLE", "2", "LTYPE", "70", "1",
		"0", "LTYPE", "2", "CONTINUOUS", "70", "0", "3", "Solid line", "72", "65", "73", "0", "40", "0.0",
		"0", "ENDTAB",
		"0", "TABLE", "2", "LAYER", "70", "2",
		"0", "LAYER", "2", "0", "70", "0", "62", "7", "6", "CONTINUOUS",
		"0", "LAYER", "2", "WALLS", "70", "0", "62", "1", "6", "CONTINUOUS",
		"0", "ENDTAB",
		"0", "ENDSEC",
		"0", "SECTION", "2", "BLOCKS",
		"0", "BLOCK", "8", "0", "2", "*MODEL_SPACE", "70", "0", "10", "0.0", "20", "0.0", "30", "0.0", "3", "*MODEL_SPACE",
		"0", "ENDBLK", "8", "0",
		"0", "BLOCK", "67", "1", "8", "0", "2", "*PAPER_SPACE", "70", "0", "10", "0.0", "20", "0.0", "30", "0.0", "3", "*PAPER_SPACE",
		"0", "ENDBLK", "67", "1", "8", "0",
		"0", "BLOCK", "8", "0", "2", "B1", "70", "0", "10", "0.0", "20", "0.0", "30", "0.0", "3", "B1",
		"0", "CIRCLE", "8", "0", "10", "0.0", "20", "0.0", "30", "0.0", "40", "1.0",
		"0", "ENDBLK", "8", "0",
		"0", "ENDSEC",
		"0", "SECTION", "2", "ENTITIES",
		"0", "INSERT", "8", "0", "2", "B1", "10", "5.0", "20", "5.0", "30", "0.0",
		"0", "LINE", "8", "WALLS", "10", "0.0", "20", "0.0", "30", "0.0", "11", "100.0", "21", "0.0", "31", "0.0",
		"0", "CIRCLE", "8", "0", "10", "50.0", "20", "50.0", "30", "0.0", "40", "25.0",
		"0", "TEXT", "8", "0", "10", "10.0", "20", "10.0", "30", "0.0", "40", "2.5", "1", "HELLO",
		"0", "LINE", "67", "1", "8", "0", "10", "0.0", "20", "0.0", "30", "0.0", "11", "10.0", "21", "10.0", "31", "0.0",
		"0", "ENDSEC",
		"0", "EOF", "",
	}, "\r\n")
	want := map[string]int{"LINE": 2, "CIRCLE": 1, "TEXT": 1, "INSERT": 1}
	for _, v := range []string{"R2004", "R2007", "R2010", "R2013", "R2018"} {
		out, warnings, err := DXFToDWG(strings.NewReader(src), v)
		if err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		var back bytes.Buffer
		if err := DWGToDXF(bytes.NewReader(out), &back); err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		text := back.String()
		f, err := dxf.Parse(&back)
		if err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		got := map[string]int{}
		paper := 0
		if s := f.Section("ENTITIES"); s != nil {
			for _, r := range s.Records {
				got[r.Type]++
				if p, _ := r.Value(67); strings.TrimSpace(p) == "1" {
					paper++
				}
			}
		}
		for typ, n := range want {
			if got[typ] != n {
				t.Errorf("%s: %s count %d, want %d (warnings %v)", v, typ, got[typ], n, warnings)
			}
		}
		if paper != 1 {
			t.Errorf("%s: %d paper space entities, want 1", v, paper)
		}
		if len(warnings) > 0 {
			t.Errorf("%s: warnings %v", v, warnings)
		}
		inB1 := 0 // records of block B1 (BLOCK, CIRCLE, ENDBLK)
		if s := f.Section("BLOCKS"); s != nil {
			cur := ""
			for _, r := range s.Records {
				if r.Type == "BLOCK" {
					cur, _ = r.Value(2)
				}
				if cur == "B1" {
					inB1++
				}
			}
		}
		if inB1 != 3 {
			t.Errorf("%s: block B1 has %d records, want 3", v, inB1)
		}
		if !strings.Contains(text, "\r\nWALLS\r\n") || !strings.Contains(text, "\r\nHELLO\r\n") {
			t.Errorf("%s: layer WALLS or text value missing", v)
		}
	}
}
