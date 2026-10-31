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
// drawing needs. Complete DXF files are returned unchanged. Tables the input defines are kept; the
// ones it lacks (often BLOCK_RECORD in hand-written R12-style files) come from the template, and
// every input block gets a block record.
func withDefaults(f *dxf.File) *dxf.File {
	if f.Section("OBJECTS") != nil && tableNamed(splitTables(f.Section("TABLES")), "BLOCK_RECORD") != nil {
		return f
	}
	t, err := dxf.Parse(bytes.NewReader(templateDXF))
	if err != nil {
		return f
	}
	taken := map[uint64]bool{} // handles of the records taken from the template
	mark := func(recs []*dxf.Record) {
		for _, r := range recs {
			taken[r.Handle(5)] = true
			taken[r.Handle(105)] = true
		}
	}
	inBlocks := map[string]bool{} // block names the input defines
	if s := f.Section("BLOCKS"); s != nil {
		for _, r := range s.Records {
			if r.Type == "BLOCK" {
				n, _ := r.Value(2)
				inBlocks[nameKey(n)] = true
			}
		}
	}
	out := &dxf.File{}
	for _, name := range []string{"HEADER", "CLASSES", "TABLES", "BLOCKS", "ENTITIES", "OBJECTS"} {
		s := f.Section(name)
		switch name {
		case "HEADER", "CLASSES":
			if s == nil {
				s = t.Section(name)
			}
		case "OBJECTS":
			if s == nil {
				s = t.Section(name)
				mark(s.Records)
			}
		case "TABLES":
			// the input's tables, and the template's for the tables it lacks
			in, tpl := splitTables(s), splitTables(t.Section("TABLES"))
			s = &dxf.Section{Name: "TABLES"}
			for _, tt := range tpl {
				n, _ := tt[0].Value(2)
				if it := tableNamed(in, n); it != nil {
					s.Records = append(s.Records, it...)
				} else {
					mark(tt)
					s.Records = append(s.Records, tt...)
				}
			}
			for _, it := range in {
				if n, _ := it[0].Value(2); tableNamed(tpl, n) == nil {
					s.Records = append(s.Records, it...)
				}
			}
		case "BLOCKS":
			// the template's *Model_Space/*Paper_Space blocks the input does not define, then the input's
			// blocks; owners are left out, so blocks attach to the record of their name
			s = &dxf.Section{Name: "BLOCKS"}
			keep := false
			for _, r := range t.Section("BLOCKS").Records {
				if r.Type == "BLOCK" {
					n, _ := r.Value(2)
					keep = !inBlocks[nameKey(n)]
				}
				if keep && (r.Type == "BLOCK" || r.Type == "ENDBLK") {
					taken[r.Handle(5)] = true
					c := &dxf.Record{Type: r.Type, Groups: append([]dxf.Group(nil), r.Groups...)}
					dropGroup(c, 330)
					s.Records = append(s.Records, c)
				}
			}
			if in := f.Section("BLOCKS"); in != nil {
				s.Records = append(s.Records, in.Records...)
			}
		case "ENTITIES":
			if s == nil {
				s = &dxf.Section{Name: "ENTITIES"}
			}
		}
		if s != nil {
			out.Sections = append(out.Sections, s)
		}
	}
	// input records keep their handles unless the template already uses them (they get new ones)
	for _, name := range []string{"TABLES", "BLOCKS", "ENTITIES"} {
		if s := f.Section(name); s != nil {
			for _, r := range s.Records {
				for _, c := range []int{5, 105} {
					if h := r.Handle(c); h != 0 && taken[h] {
						dropGroup(r, c)
					}
				}
			}
		}
	}
	completeTables(out)
	addMissingLayers(out)
	return out
}

// splitTables returns the TABLE ... ENDTAB runs of a TABLES section.
func splitTables(s *dxf.Section) [][]*dxf.Record {
	if s == nil {
		return nil
	}
	var tables [][]*dxf.Record
	for _, r := range s.Records {
		switch {
		case r.Type == "TABLE":
			tables = append(tables, []*dxf.Record{r})
		case len(tables) > 0:
			tables[len(tables)-1] = append(tables[len(tables)-1], r)
		}
	}
	return tables
}

func tableNamed(tables [][]*dxf.Record, name string) []*dxf.Record {
	for _, t := range tables {
		if n, _ := t[0].Value(2); strings.EqualFold(n, name) {
			return t
		}
	}
	return nil
}

// completeTables gives table records without a handle a new one, sets their owner to their table,
// and adds a BLOCK_RECORD for every block that has none.
func completeTables(f *dxf.File) {
	tables := f.Section("TABLES")
	next := maxRecordHandle(f) + 1
	newHandle := func() string {
		h := strings.ToUpper(strconv.FormatUint(next, 16))
		next++
		return h
	}
	var brTable *dxf.Record
	brEnd := -1
	known := map[string]bool{}
	var ctl string
	for i, r := range tables.Records {
		code := 5
		if r.Type == "DIMSTYLE" {
			code = 105
		}
		if r.Type != "ENDTAB" && r.Handle(code) == 0 {
			if code == 5 {
				setHandle(r, newHandle())
			} else {
				r.Groups = append([]dxf.Group{{Code: 105, Value: newHandle()}}, r.Groups...)
			}
		}
		switch r.Type {
		case "TABLE":
			ctl = strings.ToUpper(strconv.FormatUint(r.Handle(5), 16))
			if n, _ := r.Value(2); n == "BLOCK_RECORD" {
				brTable = r
			}
		case "ENDTAB":
			if brTable != nil && brEnd < 0 {
				brEnd = i
			}
			ctl = ""
		default:
			if at, _ := ownerPos(r); at < len(r.Groups) && r.Groups[at].Code == 330 {
				r.Groups[at].Value = ctl // the table may have a new handle
			} else {
				setOwner(r, ctl)
			}
			if r.Type == "BLOCK_RECORD" {
				n, _ := r.Value(2)
				known[nameKey(n)] = true
			}
		}
	}
	blocks := f.Section("BLOCKS")
	if brEnd < 0 || blocks == nil {
		return
	}
	owner := strings.ToUpper(strconv.FormatUint(brTable.Handle(5), 16))
	var added []*dxf.Record
	for _, r := range blocks.Records {
		if r.Type != "BLOCK" {
			continue
		}
		n, _ := r.Value(2)
		if known[nameKey(n)] {
			continue
		}
		known[nameKey(n)] = true
		added = append(added, &dxf.Record{Type: "BLOCK_RECORD", Groups: []dxf.Group{
			{Code: 5, Value: newHandle()}, {Code: 330, Value: owner},
			{Code: 100, Value: "AcDbSymbolTableRecord"}, {Code: 100, Value: "AcDbBlockTableRecord"},
			{Code: 2, Value: n}, {Code: 70, Value: "0"}, {Code: 280, Value: "1"}, {Code: 281, Value: "0"},
		}})
	}
	if len(added) > 0 {
		recs := append([]*dxf.Record(nil), tables.Records[:brEnd]...)
		recs = append(recs, added...)
		tables.Records = append(recs, tables.Records[brEnd:]...)
	}
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
