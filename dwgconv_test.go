package dwgconv

import (
	"bytes"
	"os"
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
