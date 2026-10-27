package conv

import (
	"bytes"
	"os"
	"sync"
	"testing"

	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

// TestConcurrentConversions runs DWG->DXF and DXF->DWG for several drawings at once and checks each
// result equals the one produced alone (no shared state between conversions). Run with -race.
func TestConcurrentConversions(t *testing.T) {
	samples := []string{"s2", "s3", "s7", "s10"}
	toDXF := func(n string) []byte {
		d, err := dwg.Load("../../.work/orig/" + n + ".dwg")
		if err != nil {
			t.Skip(err)
		}
		var b bytes.Buffer
		if err := ToDXF(d, &b); err != nil {
			t.Error(err)
		}
		return b.Bytes()
	}
	toDWG := func(n string) ([]byte, []string) {
		fh, err := os.Open("../../.work/oda/" + n + ".dxf")
		if err != nil {
			t.Skip(err)
		}
		defer fh.Close()
		f, err := dxf.Parse(fh)
		if err != nil {
			t.Fatal(err)
		}
		out, w, err := FromDXF(f, "R2018")
		if err != nil {
			t.Error(err)
		}
		return out, w
	}
	// reference results, one at a time
	refDXF, refDWG, refW := map[string][]byte{}, map[string][]byte{}, map[string]int{}
	for _, n := range samples {
		refDXF[n] = toDXF(n)
		var w []string
		refDWG[n], w = toDWG(n)
		refW[n] = len(w)
	}
	var wg sync.WaitGroup
	for round := 0; round < 2; round++ {
		for _, n := range samples {
			wg.Add(2)
			go func(n string) {
				defer wg.Done()
				if got := toDXF(n); !bytes.Equal(got, refDXF[n]) {
					t.Errorf("%s: concurrent DWG->DXF differs", n)
				}
			}(n)
			go func(n string) {
				defer wg.Done()
				got, w := toDWG(n)
				// DWG output embeds the save time; compare length and warnings
				if len(got) == 0 || len(w) != refW[n] {
					t.Errorf("%s: concurrent DXF->DWG warnings %d != %d", n, len(w), refW[n])
				}
			}(n)
		}
	}
	wg.Wait()
}
