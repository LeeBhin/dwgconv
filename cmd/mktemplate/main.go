// mktemplate reduces a DXF (as written by ODA or dwgconv) to an empty default drawing: only the
// standard table records (layer 0, ByBlock/ByLayer/Continuous, Standard styles, *Model_Space and
// *Paper_Space), no entities, the standard named-object dictionaries, and no user data (printer,
// page setups, author). The result is the template FromDXF completes minimal DXF files with.
//
//	mktemplate in.dxf out.dxf
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/LeeBhin/dwgconv/internal/dxf"
)

var keepEntries = map[string]map[string]bool{
	"VPORT":        {"*ACTIVE": true},
	"LTYPE":        {"BYBLOCK": true, "BYLAYER": true, "CONTINUOUS": true},
	"LAYER":        {"0": true},
	"STYLE":        {"STANDARD": true},
	"VIEW":         {},
	"UCS":          {},
	"DIMSTYLE":     {"STANDARD": true},
	"BLOCK_RECORD": {"*MODEL_SPACE": true, "*PAPER_SPACE": true},
}

// root dictionary entries kept (standard ones every AutoCAD drawing has)
var keepRoot = map[string]bool{"ACAD_COLOR": true, "ACAD_GROUP": true, "ACAD_LAYOUT": true, "ACAD_MATERIAL": true,
	"ACAD_MLEADERSTYLE": true, "ACAD_MLINESTYLE": true, "ACAD_PLOTSETTINGS": true, "ACAD_PLOTSTYLENAME": true,
	"ACAD_SCALELIST": true, "ACAD_TABLESTYLE": true, "ACAD_VISUALSTYLE": true, "ACAD_DETAILVIEWSTYLE": true,
	"ACAD_SECTIONVIEWSTYLE": true, "ACDBVARIABLEDICTIONARY": true}

// dictionaries whose entries are user content: kept empty
var emptyDicts = map[string]bool{"ACAD_GROUP": true, "ACAD_PLOTSETTINGS": true}

// header variables reset to neutral defaults (user, paths, name references)
var headerDefaults = map[string]string{
	"$LASTSAVEDBY": "", "$PROJECTNAME": "", "$HYPERLINKBASE": "", "$STYLESHEET": "",
	"$CLAYER": "0", "$TEXTSTYLE": "Standard", "$CELTYPE": "ByLayer", "$DIMSTYLE": "Standard",
	"$DIMTXSTY": "Standard", "$CMLSTYLE": "Standard", "$UCSNAME": "", "$UCSBASE": "", "$UCSORTHOREF": "",
	"$PUCSNAME": "", "$PUCSBASE": "", "$PUCSORTHOREF": "", "$DIMBLK": "", "$DIMBLK1": "", "$DIMBLK2": "",
	"$DIMLDRBLK": "", "$DIMLTYPE": "", "$DIMLTEX1": "", "$DIMLTEX2": "",
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: mktemplate in.dxf out.dxf")
		os.Exit(2)
	}
	fh, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	f, err := dxf.Parse(fh)
	fh.Close()
	if err != nil {
		panic(err)
	}
	keep := map[uint64]bool{}
	out := &dxf.File{}
	for _, s := range f.Sections {
		ns := &dxf.Section{Name: s.Name}
		switch s.Name {
		case "HEADER":
			for _, r := range s.Records {
				if v, ok := headerDefaults[r.Type]; ok && len(r.Groups) > 0 {
					r.Groups = r.Groups[:1]
					r.Groups[0].Value = v
				}
				ns.Records = append(ns.Records, r)
			}
		case "CLASSES":
			ns.Records = s.Records
		case "TABLES":
			table := ""
			for _, r := range s.Records {
				switch r.Type {
				case "TABLE":
					table, _ = r.Value(2)
					keep[r.Handle(5)] = true
					ns.Records = append(ns.Records, r)
					continue
				case "ENDTAB":
					ns.Records = append(ns.Records, r)
					continue
				}
				name, _ := r.Value(2)
				want, filtered := keepEntries[table]
				if filtered && !want[strings.ToUpper(name)] {
					continue
				}
				if table == "APPID" && !strings.HasPrefix(strings.ToUpper(name), "ACAD") {
					continue
				}
				h := r.Handle(5)
				if h == 0 {
					h = r.Handle(105)
				}
				keep[h] = true
				ns.Records = append(ns.Records, r)
			}
		case "BLOCKS":
			for _, r := range s.Records {
				if (r.Type == "BLOCK" || r.Type == "ENDBLK") && keep[r.Handle(330)] {
					keep[r.Handle(5)] = true
					ns.Records = append(ns.Records, r)
				}
			}
		case "ENTITIES", "ACDSDATA":
			// no entities; the data store belongs to the dropped content
			if s.Name == "ACDSDATA" {
				continue
			}
		case "OBJECTS":
			ns.Records = objects(s.Records, keep)
		}
		out.Sections = append(out.Sections, ns)
	}
	// drop references to records that were not kept
	for _, s := range out.Sections {
		for _, r := range s.Records {
			r.Groups = scrubRefs(r, keep)
		}
	}
	o, err := os.Create(os.Args[2])
	if err != nil {
		panic(err)
	}
	if err := out.Write(o); err != nil {
		panic(err)
	}
	o.Close()
}

// objects keeps the root dictionary's standard entries and everything they own.
func objects(recs []*dxf.Record, keep map[uint64]bool) []*dxf.Record {
	byH := map[uint64]*dxf.Record{}
	for _, r := range recs {
		byH[r.Handle(5)] = r
	}
	if len(recs) == 0 {
		return nil
	}
	root := recs[0]
	keep[root.Handle(5)] = true
	var visit func(h uint64)
	visit = func(h uint64) {
		r := byH[h]
		if r == nil || keep[h] {
			return
		}
		if r.Type == "LAYOUT" {
			// only the layouts of the kept spaces
			if !keep[r.Handle(330)] && !layoutOfKeptBlock(r, keep) {
				return
			}
		}
		if r.Type == "MATERIAL" {
			if n, _ := r.Value(1); !map[string]bool{"BYBLOCK": true, "BYLAYER": true, "GLOBAL": true}[strings.ToUpper(n)] {
				return
			}
		}
		keep[h] = true
		if r.Type == "LAYOUT" {
			scrubPlot(r)
		}
		name := ""
		for _, g := range r.Groups {
			switch g.Code {
			case 3:
				name = g.Value
			case 350, 360:
				if r.Type == "DICTIONARY" || r.Type == "ACDBDICTIONARYWDFLT" {
					visit(g.Handle())
				}
			}
		}
		_ = name
		// extension dictionary and hard-owned children
		for i, g := range r.Groups {
			if g.Code == 360 && i > 0 && r.Groups[i-1].Code == 102 {
				visit(g.Handle())
			}
		}
	}
	// root: only standard entries
	var entries []uint64
	for i := 0; i+1 < len(root.Groups); i++ {
		if root.Groups[i].Code == 3 && (root.Groups[i+1].Code == 350 || root.Groups[i+1].Code == 360) {
			if keepRoot[strings.ToUpper(root.Groups[i].Value)] {
				entries = append(entries, root.Groups[i+1].Handle())
			}
		}
	}
	for _, h := range entries {
		r := byH[h]
		if r == nil {
			continue
		}
		dictName := ""
		for i := 0; i+1 < len(root.Groups); i++ {
			if root.Groups[i+1].Handle() == h {
				dictName = strings.ToUpper(root.Groups[i].Value)
			}
		}
		if emptyDicts[dictName] {
			keep[h] = true
			continue
		}
		visit(h)
	}
	var out []*dxf.Record
	for _, r := range recs {
		if keep[r.Handle(5)] {
			out = append(out, r)
		}
	}
	return out
}

func layoutOfKeptBlock(r *dxf.Record, keep map[uint64]bool) bool {
	for _, g := range r.Groups {
		if g.Code == 330 && keep[g.Handle()] {
			return true
		}
	}
	return false
}

// scrubPlot removes the page setup's machine-specific values (printer, paper, plot style table).
func scrubPlot(r *dxf.Record) {
	for i, g := range r.Groups {
		switch g.Code {
		case 1:
			r.Groups[i].Value = ""
		case 2:
			r.Groups[i].Value = "none_device"
		case 7:
			r.Groups[i].Value = ""
		}
		if g.Code == 100 && g.Value == "AcDbLayout" {
			return
		}
	}
}

// scrubRefs removes dictionary entries, reactors and extension dictionaries pointing to dropped
// records.
func scrubRefs(r *dxf.Record, keep map[uint64]bool) []dxf.Group {
	gs := r.Groups
	var out []dxf.Group
	for i := 0; i < len(gs); i++ {
		g := gs[i]
		if g.Code == 102 && (g.Value == "{ACAD_REACTORS" || g.Value == "{ACAD_XDICTIONARY") {
			var inner []dxf.Group
			j := i + 1
			for ; j < len(gs) && gs[j].Code != 102; j++ {
				if keep[gs[j].Handle()] {
					inner = append(inner, gs[j])
				}
			}
			if len(inner) > 0 {
				out = append(out, g)
				out = append(out, inner...)
				out = append(out, gs[j])
			}
			i = j
			continue
		}
		if (r.Type == "DICTIONARY" || r.Type == "ACDBDICTIONARYWDFLT") && g.Code == 3 && i+1 < len(gs) &&
			(gs[i+1].Code == 350 || gs[i+1].Code == 360) {
			if !keep[gs[i+1].Handle()] {
				i++
				continue
			}
		}
		out = append(out, g)
	}
	return out
}
