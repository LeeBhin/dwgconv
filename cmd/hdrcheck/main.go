// hdrcheck decodes and re-encodes the non-object sections and reports whether the bytes are identical.
// With -variants it writes copies with sections removed or replaced by defaults (for ODA checks).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"dwgconv/internal/container"
	"dwgconv/internal/dwg"
)

func main() {
	dump := flag.Bool("dump", false, "print decoded header variables")
	variants := flag.String("variants", "", "write section variants of each input under this directory")
	flag.Parse()
	if *variants != "" {
		for _, path := range flag.Args() {
			writeVariants(path, *variants)
		}
		return
	}
	for _, path := range flag.Args() {
		f, err := container.ReadFile(path)
		if err != nil {
			fmt.Printf("%s: %v\n", path, err)
			continue
		}
		fmt.Printf("%s %s\n", path, f.Version)
		for _, r := range dwg.CheckSections(f) {
			fmt.Printf("  %-24s %s\n", r.Name, r.Result)
		}
		if *dump {
			h, err := dwg.DecodeFileHeader(f)
			if err == nil {
				fmt.Printf("%+v\n", h.Vars)
			}
		}
	}
}

// writeVariants writes copies of in with non-object sections removed or replaced by defaults,
// to test with the ODA converter what a from-scratch writer must emit.
func writeVariants(in, outDir string) {
	base := filepath.Base(in)
	variants := map[string]func(f *container.File){
		"noDS":     func(f *container.File) { drop(f, "AcDb:AcDsPrototype_1b") },
		"noAIH":    func(f *container.File) { drop(f, "AcDb:AppInfoHistory") },
		"noPrev":   func(f *container.File) { set(f, "AcDb:Preview", dwg.DefaultPreview().Encode()) },
		"dropPrev": func(f *container.File) { drop(f, "AcDb:Preview") },
		"defaults": func(f *container.File) {
			v, _ := dwg.FileVersion(f)
			text := dwg.FileText(f)
			jd := dwg.JulianDate{Day: 2461340, Ms: 0}
			set(f, "AcDb:SummaryInfo", dwg.DefaultSummaryInfo(jd, jd).Encode(v, text))
			set(f, "AcDb:AppInfo", dwg.DefaultAppInfo("dwgconv", "", "<ProductInformation name =\"dwgconv\"/>").Encode(text))
			set(f, "AcDb:RevHistory", dwg.DefaultRevHistory().Encode())
			set(f, "AcDb:Template", dwg.DefaultTemplate(v, 1).Encode(v))
			h, _ := dwg.DecodeFileHeader(f)
			set(f, "AcDb:ObjFreeSpace", dwg.DefaultObjFreeSpace(1000, jd).Encode(v))
			set(f, "AcDb:AuxHeader", dwg.DefaultAuxHeader(f.Version, 0, jd, jd, h.Vars.HANDSEED.H).Encode(v))
		},
		"hdrdefaults": func(f *container.File) {
			v, _ := dwg.FileVersion(f)
			if !v.R2018 {
				return
			}
			text := dwg.FileText(f)
			orig, _ := dwg.DecodeFileHeader(f)
			h := dwg.HeaderDefaults()
			dwg.CopyHeaderRefs(h, orig)
			set(f, "AcDb:Header", h.Encode(v, text))
		},
		"all": nil,
	}
	variants["all"] = func(f *container.File) {
		for _, k := range []string{"noDS", "noAIH", "noPrev", "defaults"} {
			variants[k](f)
		}
	}
	for name, fn := range variants {
		f, err := container.ReadFile(in)
		if err != nil {
			panic(err)
		}
		fn(f)
		dir := filepath.Join(outDir, name)
		os.MkdirAll(dir, 0o755)
		if err := os.WriteFile(filepath.Join(dir, base), f.Write(), 0o644); err != nil {
			panic(err)
		}
		fmt.Println("wrote", name)
	}
}

func drop(f *container.File, name string) {
	out := f.Sections[:0]
	for _, s := range f.Sections {
		if s.Name != name {
			out = append(out, s)
		}
	}
	f.Sections = out
}

func set(f *container.File, name string, data []byte) {
	if s := f.Section(name); s != nil {
		s.Data = data
	}
}
