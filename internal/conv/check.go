package conv

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// TypeCheck is the per-type result of comparing our DXF mapping with a reference DXF (ODA).
type TypeCheck struct {
	Count       int    // objects of this type with a mapping
	ExportEqual int    // our exported record equals the reference record
	ImportEqual int    // importing the reference record reproduces the decoded object
	ExportDiff  string // first export difference
	ImportDiff  string // first import difference
}

// Check compares every mapped object of d with the record of the same handle in ref.
func Check(d *dwg.Document, ref *dxf.File) map[string]*TypeCheck {
	x := newCtx(d)
	recs := map[uint64]*dxf.Record{}
	for _, s := range ref.Sections {
		for _, r := range s.Records {
			h := r.Handle(5)
			if h == 0 {
				h = r.Handle(105)
			}
			if h != 0 {
				recs[h] = r
			}
		}
	}
	out := map[string]*TypeCheck{}
	for _, o := range d.Objects {
		mp := mappingFor(o)
		if mp == nil {
			continue
		}
		name := o.TypeName(d.Classes.Classes)
		tc := out[name]
		if tc == nil {
			tc = &TypeCheck{}
			out[name] = tc
		}
		tc.Count++
		r := recs[o.Handle]
		if r == nil {
			if tc.ExportDiff == "" {
				tc.ExportDiff = fmt.Sprintf("%X: no reference record", o.Handle)
			}
			continue
		}
		// export
		var buf bytes.Buffer
		w := dxf.NewWriter(&buf)
		w.UnescapeU = !d.Version.R2007
		x.exportRecord(w, o)
		w.Flush()
		ours := parseGroups(buf.String())
		theirs := append([]dxf.Group{{Code: 0, Value: r.Type}}, r.Groups...)
		if d := groupsDiff(ours, theirs); d == "" {
			tc.ExportEqual++
		} else if tc.ExportDiff == "" {
			tc.ExportDiff = fmt.Sprintf("%X: %s", o.Handle, d)
		}
		// import
		got := x.importRecord(r, o.IsEntity)
		if got == nil {
			continue
		}
		if d := objectDiff(o, got, x.xdicOmitted(o)); d == "" {
			tc.ImportEqual++
		} else if tc.ImportDiff == "" {
			tc.ImportDiff = fmt.Sprintf("%X: %s", o.Handle, d)
		}
	}
	return out
}

func parseGroups(s string) []dxf.Group {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	var out []dxf.Group
	for i := 0; i+1 < len(lines); i += 2 {
		c, err := strconv.Atoi(strings.TrimSpace(lines[i]))
		if err != nil {
			break
		}
		out = append(out, dxf.Group{Code: c, Value: lines[i+1]})
	}
	return out
}

func valuesEqual(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == b {
		return true
	}
	fa, ea := strconv.ParseFloat(a, 64)
	fb, eb := strconv.ParseFloat(b, 64)
	if ea == nil && eb == nil {
		return fa == fb || math.Abs(fa-fb) <= 1e-12*math.Max(1, math.Abs(fb))
	}
	return strings.EqualFold(a, b) && isHex(a)
}

func isHex(s string) bool {
	_, err := strconv.ParseUint(s, 16, 64)
	return err == nil
}

func groupsDiff(a, b []dxf.Group) string {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i].Code != b[i].Code || !valuesEqual(a[i].Value, b[i].Value) {
			return fmt.Sprintf("group %d: ours %d=%q, ref %d=%q", i, a[i].Code, a[i].Value, b[i].Code, b[i].Value)
		}
	}
	if len(a) != len(b) {
		if len(a) > n {
			return fmt.Sprintf("ours has extra %d=%q", a[n].Code, a[n].Value)
		}
		return fmt.Sprintf("ref has extra %d=%q", b[n].Code, b[n].Value)
	}
	return ""
}

// objectDiff compares the DXF-representable parts of two objects and names the first difference.
func objectDiff(want, got *dwg.Object, xdicOmitted bool) string {
	if want.IsEntity {
		we, ge := *want.Ent, *got.Ent
		// fields DXF cannot carry
		ge.HasGraphic, ge.Graphic = we.HasGraphic, we.Graphic
		ge.HasDS, ge.FullVS, ge.FaceVS, ge.EdgeVS = we.HasDS, we.FullVS, we.FaceVS, we.EdgeVS
		ge.FullVSH, ge.FaceVSH, ge.EdgeVSH, ge.Shadow = we.FullVSH, we.FaceVSH, we.EdgeVSH, we.Shadow
		if ge.Invisible&1 == we.Invisible&1 {
			ge.Invisible = we.Invisible
		}
		if lineweightToDXF(ge.Lineweight) == lineweightToDXF(we.Lineweight) {
			ge.Lineweight = we.Lineweight
		}
		if d := valueDiff("Ent", reflect.ValueOf(we), reflect.ValueOf(ge)); d != "" {
			return d
		}
		if we.EntMode == 0 && want.Owner.H != got.Owner.H {
			return fmt.Sprintf("Owner %X != %X", want.Owner.H, got.Owner.H)
		}
	}
	if len(want.Reactors) != len(got.Reactors) {
		return fmt.Sprintf("Reactors %d != %d", len(want.Reactors), len(got.Reactors))
	}
	if !xdicOmitted && (want.XDicMiss != got.XDicMiss || want.XDic.H != got.XDic.H) {
		return "XDic"
	}
	if len(want.EED) != len(got.EED) {
		return fmt.Sprintf("EED count %d != %d", len(want.EED), len(got.EED))
	}
	return valueDiff("Body", reflect.ValueOf(want.Body), reflect.ValueOf(got.Body))
}

// valueDiff deep-compares ignoring Ref.Code and tolerating float rounding.
func valueDiff(path string, a, b reflect.Value) string {
	if a.Kind() == reflect.Pointer || a.Kind() == reflect.Interface {
		if a.IsNil() || b.IsNil() {
			if a.IsNil() != b.IsNil() {
				return path + ": nil mismatch"
			}
			return ""
		}
		return valueDiff(path, a.Elem(), b.Elem())
	}
	if a.Type() == reflect.TypeOf(dwg.Ref{}) {
		if a.FieldByName("H").Uint() != b.FieldByName("H").Uint() {
			return fmt.Sprintf("%s: %X != %X", path, a.FieldByName("H").Uint(), b.FieldByName("H").Uint())
		}
		return ""
	}
	switch a.Kind() {
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			// derived fields are recomputed from others on write; DXF does not carry them
			if f := a.Type().Field(i); !f.IsExported() || f.Tag.Get("conv") == "derived" {
				continue
			}
			if d := valueDiff(path+"."+a.Type().Field(i).Name, a.Field(i), b.Field(i)); d != "" {
				return d
			}
		}
	case reflect.Slice, reflect.Array:
		if a.Len() != b.Len() {
			return fmt.Sprintf("%s: len %d != %d", path, a.Len(), b.Len())
		}
		for i := 0; i < a.Len(); i++ {
			if d := valueDiff(fmt.Sprintf("%s[%d]", path, i), a.Index(i), b.Index(i)); d != "" {
				return d
			}
		}
	case reflect.Float32, reflect.Float64:
		x, y := a.Float(), b.Float()
		if x != y && math.Abs(x-y) > 1e-12*math.Max(1, math.Abs(x)) {
			return fmt.Sprintf("%s: %v != %v", path, x, y)
		}
	case reflect.String:
		if dxf.UnescapeU(a.String()) != dxf.UnescapeU(b.String()) {
			return fmt.Sprintf("%s: %q != %q", path, a.String(), b.String())
		}
	default:
		if a.CanInterface() && !reflect.DeepEqual(a.Interface(), b.Interface()) {
			return fmt.Sprintf("%s: %v != %v", path, a.Interface(), b.Interface())
		}
	}
	return ""
}
