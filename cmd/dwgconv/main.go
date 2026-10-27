// dwgconv converts between DWG and DXF:
//
//	dwgconv in.dwg out.dxf
//	dwgconv in.dxf out.dwg
//	dwgconv in.dwg out.dwg   (decode and re-encode every object; for testing)
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dwgconv/internal/conv"
	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

var outVer = flag.String("version", conv.DefaultOutVersion, "DWG version to write: R2004, R2007, R2010, R2013, R2018")

func main() {
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: dwgconv [-version R2018] <in.dwg|in.dxf> <out.dxf|out.dwg>")
		os.Exit(2)
	}
	in, out := flag.Arg(0), flag.Arg(1)
	from, to := strings.ToLower(filepath.Ext(in)), strings.ToLower(filepath.Ext(out))
	var err error
	switch {
	case from == ".dwg" && to == ".dwg":
		err = rewrite(in, out)
	case from == ".dwg" && to == ".dxf":
		err = dwgToDXF(in, out)
	case from == ".dxf" && to == ".dwg":
		err = dxfToDWG(in, out)
	default:
		err = fmt.Errorf("unsupported conversion %s -> %s", from, to)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func dwgToDXF(in, out string) error {
	d, err := dwg.Load(in)
	if err != nil {
		return err
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := conv.ToDXF(d, f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func dxfToDWG(in, out string) error {
	fh, err := os.Open(in)
	if err != nil {
		return err
	}
	f, err := dxf.Parse(fh)
	fh.Close()
	if err != nil {
		return err
	}
	data, warnings, err := conv.FromDXF(f, *outVer)
	if err != nil {
		return err
	}
	if w := warnings; len(w) > 0 {
		fmt.Fprintln(os.Stderr, "skipped (no mapping):", strings.Join(w, ", "))
	}
	return os.WriteFile(out, data, 0o644)
}

func rewrite(in, out string) error {
	d, err := dwg.Load(in)
	if err != nil {
		return err
	}
	objs, handles := d.EncodeObjects()
	d.File.Section("AcDb:AcDbObjects").Data = objs
	d.File.Section("AcDb:Handles").Data = handles
	return os.WriteFile(out, d.File.Write(), 0o644)
}
