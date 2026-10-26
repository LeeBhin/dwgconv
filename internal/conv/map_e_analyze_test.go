package conv

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// TestEAnalyze lists every export/import diff for one type across the samples (env T, N=max per category).
func TestEAnalyze(t *testing.T) {
	typ := os.Getenv("T")
	if typ == "" {
		t.Skip()
	}
	cats := map[string][]string{}
	re := regexp.MustCompile(`[0-9A-F]{2,}|"[^"]*"|\d+`)
	for n := 1; n <= 9; n++ {
		d, err := dwg.Load(fmt.Sprintf(eWork()+"/orig/s%d.dwg", n))
		if err != nil {
			t.Fatal(err)
		}
		fh, _ := os.Open(fmt.Sprintf(eWork()+"/oda/s%d.dxf", n))
		ref, _ := dxf.Parse(fh)
		fh.Close()
		x := newCtx(d)
		recs := map[uint64]*dxf.Record{}
		for _, s := range ref.Sections {
			for _, r := range s.Records {
				if h := r.Handle(5); h != 0 {
					recs[h] = r
				}
			}
		}
		for _, o := range d.Objects {
			if o.TypeName(d.Classes.Classes) != typ || mappingFor(o) == nil {
				continue
			}
			r := recs[o.Handle]
			key := ""
			if r != nil && eODAOmits(x, o.Handle) {
				key = "present but predicted omitted"
			} else if r == nil {
				owner := d.ByHandle[o.Owner.H]
				on := "?"
				if owner != nil {
					on = owner.TypeName(d.Classes.Classes)
				}
				key = "missing (owner " + on + ")"
				if eODAOmits(x, o.Handle) {
					key = "missing, predicted omitted"
				}
			} else {
				var buf bytes.Buffer
				w := dxf.NewWriter(&buf)
				w.UnescapeU = !d.Version.R2007
				x.exportRecord(w, o)
				w.Flush()
				ours, theirs := parseGroups(buf.String()), append([]dxf.Group{{Code: 0, Value: r.Type}}, r.Groups...)
				if os.Getenv("FIELDS") != "" {
					ours, theirs = eFromSubclass(ours), eFromSubclass(theirs)
				}
				if df := groupsDiff(ours, theirs); df != "" {
					key = "export: " + df
				} else if got := x.importRecord(r, o.IsEntity); got != nil {
					df := x.objectDiff(o, got, x.xdicOmitted(o))
					if os.Getenv("FIELDS") != "" {
						df = valueDiff("Body", reflect.ValueOf(o.Body), reflect.ValueOf(got.Body))
					}
					if df != "" {
						key = "import: " + df
					}
				}
			}
			if key == "" {
				continue
			}
			norm := re.ReplaceAllString(key, "#")
			cats[norm] = append(cats[norm], fmt.Sprintf("s%d %X %s", n, o.Handle, key))
		}
	}
	ks := make([]string, 0, len(cats))
	for k := range cats {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return len(cats[ks[i]]) > len(cats[ks[j]]) })
	for _, k := range ks {
		t.Logf("[%d] %s", len(cats[k]), k)
		for _, e := range cats[k][:min(4, len(cats[k]))] {
			t.Logf("      %s", strings.ReplaceAll(e, "\n", " "))
		}
	}
}

// eFromSubclass drops the groups before the first subclass marker (ownership part).
func eFromSubclass(gs []dxf.Group) []dxf.Group {
	for i, g := range gs {
		if g.Code == 100 {
			return gs[i:]
		}
	}
	return gs
}

// eWork is the sample directory (.work), overridable with EWORK (e.g. /work in a container).
func eWork() string {
	if w := os.Getenv("EWORK"); w != "" {
		return w
	}
	return "C:/Users/bin/Desktop/dwgconv/.work"
}
