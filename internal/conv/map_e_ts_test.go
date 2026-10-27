package conv

import (
	"os"
	"reflect"
	"testing"

	"github.com/LeeBhin/dwgconv/internal/dwg"
)

// TestETableStyleRebuilt compares TABLESTYLE/CELLSTYLEMAP of an original DWG with the FromDXF output (ETS=s2).
func TestETableStyleRebuilt(t *testing.T) {
	n := os.Getenv("ETS")
	if n == "" {
		t.Skip()
	}
	a, _ := dwg.Load(eWork() + "/orig/" + n + ".dwg")
	b, _ := dwg.Load(eWork() + "/fromdxfE/" + n + ".dwg")
	for _, o := range a.Objects {
		if o.TypeName(a.Classes.Classes) != "TABLESTYLE" {
			continue
		}
		g := b.ByHandle[o.Handle]
		if g == nil {
			t.Logf("%X missing in output", o.Handle)
			continue
		}
		t.Logf("%X orig %d bytes, rebuilt %d bytes, body diff: %q, xdic %X -> %X",
			o.Handle, len(o.Raw), len(g.Encode(b.Version, b.Text)),
			valueDiff("Body", reflect.ValueOf(o.Body), reflect.ValueOf(g.Body)), o.XDic.H, g.XDic.H)
	}
}
