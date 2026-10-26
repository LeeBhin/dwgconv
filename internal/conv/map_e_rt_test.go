package conv

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// TestERoundTrip imports every reference record of type T and exports it again (DXF -> object -> DXF).
func TestERoundTrip(t *testing.T) {
	typ := os.Getenv("T")
	if typ == "" {
		t.Skip()
	}
	for n := 1; n <= 9; n++ {
		d, _ := dwg.Load(fmt.Sprintf(eWork()+"/orig/s%d.dwg", n))
		fh, _ := os.Open(fmt.Sprintf(eWork()+"/oda/s%d.dxf", n))
		ref, _ := dxf.Parse(fh)
		fh.Close()
		x := newCtx(d)
		ok, total := 0, 0
		for _, s := range ref.Sections {
			for _, r := range s.Records {
				if r.Type != typ {
					continue
				}
				total++
				o := x.importRecord(r, false)
				var buf bytes.Buffer
				w := dxf.NewWriter(&buf)
				w.UnescapeU = !d.Version.R2007
				x.exportRecord(w, o)
				w.Flush()
				if df := groupsDiff(parseGroups(buf.String()), append([]dxf.Group{{Code: 0, Value: r.Type}}, r.Groups...)); df != "" {
					t.Logf("s%d %X: %s", n, r.Handle(5), df)
				} else {
					ok++
				}
			}
		}
		if total > 0 {
			t.Logf("s%d %s: %d/%d round-trip equal", n, typ, ok, total)
		}
	}
}
