package conv

import (
	"os"
	"strings"
	"testing"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// TestEEDAgainstODA decodes the EED of every object in the samples and compares it with the
// xdata ODA wrote to its DXF (set DWGCONV_ODA_DXF to the folder with s<N>.dxf files).
func TestEEDAgainstODA(t *testing.T) {
	ref := os.Getenv("DWGCONV_ODA_DXF")
	if ref == "" {
		t.Skip("DWGCONV_ODA_DXF not set")
	}
	for _, n := range []string{"s2", "s3", "s7"} {
		d, err := dwg.Load("../../.work/orig/" + n + ".dwg")
		if err != nil {
			t.Fatal(err)
		}
		fh, err := os.Open(ref + "/" + n + ".dxf")
		if err != nil {
			t.Fatal(err)
		}
		f, err := dxf.Parse(fh)
		fh.Close()
		if err != nil {
			t.Fatal(err)
		}
		xdata := map[uint64][]dxf.Group{}
		for _, s := range f.Sections {
			for _, r := range s.Records {
				h := r.Handle(5)
				if h == 0 {
					h = r.Handle(105)
				}
				for i, g := range r.Groups {
					if g.Code >= 1000 {
						xdata[h] = r.Groups[i:]
						break
					}
				}
			}
		}
		total, match := 0, 0
		for _, o := range d.Objects {
			if len(o.EED) == 0 {
				continue
			}
			var got []dxf.Group
			for _, e := range o.EED {
				app := "?"
				if a := d.ByHandle[e.App.H]; a != nil {
					app = "app"
				}
				gs, ok := eedToGroups(e.Data, d.Version.R2007, d.Text)
				if !ok {
					t.Errorf("%s %x: undecodable EED", n, o.Handle)
				}
				got = append(got, dxf.Group{Code: 1001, Value: app})
				got = append(got, gs...)
				if back := groupsToEED(gs, d.Version.R2007, d.Text); string(back) != string(e.Data) {
					t.Errorf("%s %x: EED re-encode differs\n orig %x\n back %x", n, o.Handle, e.Data, back)
				}
			}
			total++
			want := xdata[o.Handle]
			if same(got, want) {
				match++
			} else if total-match <= 3 {
				t.Logf("%s %x: got %v\n want %v", n, o.Handle, got, want)
			}
		}
		t.Logf("%s: objects with EED %d, xdata equal to ODA %d", n, total, match)
	}
}

func same(a, b []dxf.Group) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Code == 1001 {
			continue // app names need the APPID table
		}
		if a[i].Code != b[i].Code || strings.TrimSpace(a[i].Value) != strings.TrimSpace(b[i].Value) {
			return false
		}
	}
	return true
}
