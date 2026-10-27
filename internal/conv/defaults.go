package conv

import (
	"bytes"
	_ "embed"
	"strconv"
	"strings"

	"github.com/LeeBhin/dwgconv/internal/dxf"
)

// template.dxf is an empty AutoCAD drawing (standard table records, *Model_Space/*Paper_Space,
// layouts and named-object dictionaries; made with cmd/mktemplate). It completes DXF files that
// carry only entities.
//
//go:embed template.dxf
var templateDXF []byte

// withDefaults returns f completed with the template when it lacks the tables or the objects a
// drawing needs. Complete DXF files are returned unchanged.
func withDefaults(f *dxf.File) *dxf.File {
	if f.Section("TABLES") != nil && f.Section("OBJECTS") != nil {
		return f
	}
	t, err := dxf.Parse(bytes.NewReader(templateDXF))
	if err != nil {
		return f
	}
	pick := func(name string) *dxf.Section {
		if s := f.Section(name); s != nil {
			return s
		}
		return t.Section(name)
	}
	taken := map[uint64]bool{} // handles of the records taken from the template
	mark := func(s *dxf.Section) {
		for _, r := range s.Records {
			taken[r.Handle(5)] = true
			taken[r.Handle(105)] = true
		}
	}
	out := &dxf.File{}
	for _, name := range []string{"HEADER", "CLASSES", "TABLES", "BLOCKS", "ENTITIES", "OBJECTS"} {
		s := pick(name)
		switch name {
		case "TABLES", "OBJECTS":
			if f.Section(name) == nil {
				mark(s)
			}
		case "BLOCKS":
			// the template's *Model_Space/*Paper_Space blocks, then the input's blocks
			tb := t.Section("BLOCKS")
			mark(tb)
			s = &dxf.Section{Name: "BLOCKS", Records: append([]*dxf.Record(nil), tb.Records...)}
			if in := f.Section("BLOCKS"); in != nil {
				s.Records = append(s.Records, in.Records...)
			}
		case "ENTITIES":
			if s = f.Section("ENTITIES"); s == nil {
				s = &dxf.Section{Name: "ENTITIES"}
			}
		}
		if s != nil {
			out.Sections = append(out.Sections, s)
		}
	}
	// input records keep their handles unless the template already uses them (they get new ones)
	for _, name := range []string{"BLOCKS", "ENTITIES"} {
		if s := f.Section(name); s != nil {
			for _, r := range s.Records {
				if h := r.Handle(5); h != 0 && taken[h] {
					dropGroup(r, 5)
				}
			}
		}
	}
	addMissingLayers(out)
	return out
}

func dropGroup(r *dxf.Record, code int) {
	for i, g := range r.Groups {
		if g.Code == code {
			r.Groups = append(r.Groups[:i], r.Groups[i+1:]...)
			return
		}
	}
}

// addMissingLayers adds a LAYER record (a copy of layer 0) for every layer name entities use that
// the LAYER table does not define.
func addMissingLayers(f *dxf.File) {
	tables := f.Section("TABLES")
	if tables == nil {
		return
	}
	known := map[string]bool{}
	var zero *dxf.Record
	end := -1
	inLayer := false
	for i, r := range tables.Records {
		switch {
		case r.Type == "TABLE":
			n, _ := r.Value(2)
			inLayer = n == "LAYER"
		case r.Type == "ENDTAB":
			if inLayer {
				end = i
			}
			inLayer = false
		case inLayer && r.Type == "LAYER":
			n, _ := r.Value(2)
			known[strings.ToUpper(n)] = true
			if n == "0" {
				zero = r
			}
		}
	}
	if zero == nil || end < 0 {
		return
	}
	next := maxRecordHandle(f) + 1
	var added []*dxf.Record
	for _, name := range []string{"BLOCKS", "ENTITIES"} {
		s := f.Section(name)
		if s == nil {
			continue
		}
		for _, r := range s.Records {
			n, ok := r.Value(8)
			if !ok || known[strings.ToUpper(n)] {
				continue
			}
			known[strings.ToUpper(n)] = true
			l := &dxf.Record{Type: "LAYER", Groups: append([]dxf.Group(nil), zero.Groups...)}
			for i := range l.Groups {
				switch l.Groups[i].Code {
				case 5:
					l.Groups[i].Value = strings.ToUpper(strconv.FormatUint(next, 16))
				case 2:
					l.Groups[i].Value = n
				}
			}
			next++
			added = append(added, l)
		}
	}
	if len(added) > 0 {
		recs := append([]*dxf.Record(nil), tables.Records[:end]...)
		recs = append(recs, added...)
		tables.Records = append(recs, tables.Records[end:]...)
	}
}
