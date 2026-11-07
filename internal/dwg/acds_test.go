package dwg

import (
	"bytes"
	"os"
	"testing"
)

// The section of an ODA-written R2018 file with three solids: parse it, rebuild it from the records
// and parse the result again.
func TestAcDsRoundTrip(t *testing.T) {
	b, err := os.ReadFile("testdata/acds_oda_r18.bin")
	if err != nil {
		t.Skip(err)
	}
	m, err := ParseAcDs(b)
	if err != nil {
		t.Fatal(err)
	}
	want := map[uint64]int{0x176: 1720, 0x2E1: 8809, 0x37D: 2207}
	if len(m) != len(want) {
		t.Fatalf("records: %d", len(m))
	}
	for h, n := range want {
		if len(m[h]) != n || !bytes.HasPrefix(m[h], []byte("ASM BinaryFile")) {
			t.Errorf("%X: %d bytes", h, len(m[h]))
		}
	}
	recs := []AcDsRecord{{0x176, m[0x176]}, {0x2E1, m[0x2E1]}, {0x37D, m[0x37D]}}
	out := BuildAcDs(recs)
	if len(out) != len(b) {
		t.Errorf("size %d, ODA %d", len(out), len(b))
	}
	if !bytes.Equal(out, b) {
		i := 0
		for i < len(out) && i < len(b) && out[i] == b[i] {
			i++
		}
		t.Errorf("differs from the ODA section at %#x", i)
	}
	m2, err := ParseAcDs(out)
	if err != nil {
		t.Fatal(err)
	}
	for h := range want {
		if !bytes.Equal(m2[h], m[h]) {
			t.Errorf("%X differs after rebuild", h)
		}
	}
}
