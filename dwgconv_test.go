package dwgconv

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/LeeBhin/dwgconv/internal/container"
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
