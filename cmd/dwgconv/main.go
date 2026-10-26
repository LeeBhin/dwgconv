// dwgconv converts between DWG and DXF:
//
//	dwgconv in.dwg out.dxf
//	dwgconv in.dxf out.dwg
//	dwgconv in.dwg out.dwg   (decode and re-encode every object; for testing)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dwgconv/internal/dwg"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: dwgconv <in.dwg|in.dxf> <out.dxf|out.dwg>")
		os.Exit(2)
	}
	in, out := os.Args[1], os.Args[2]
	from, to := strings.ToLower(filepath.Ext(in)), strings.ToLower(filepath.Ext(out))
	var err error
	switch {
	case from == ".dwg" && to == ".dwg":
		err = rewrite(in, out)
	default:
		err = fmt.Errorf("unsupported conversion %s -> %s", from, to)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
