// dxfcheck compares our DXF mapping with a reference DXF of the same drawing (from ODA):
//
//	dxfcheck file.dwg reference.dxf
package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/LeeBhin/dwgconv/internal/conv"
	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: dxfcheck file.dwg reference.dxf")
		os.Exit(2)
	}
	d, err := dwg.Load(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fh, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ref, err := dxf.Parse(fh)
	fh.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	res := conv.Check(d, ref)
	names := make([]string, 0, len(res))
	for k := range res {
		names = append(names, k)
	}
	sort.Strings(names)
	fmt.Printf("%-22s %7s %7s %7s\n", "type", "count", "export", "import")
	for _, k := range names {
		r := res[k]
		fmt.Printf("%-22s %7d %7d %7d\n", k, r.Count, r.ExportEqual, r.ImportEqual)
		if r.ExportDiff != "" {
			fmt.Printf("    export: %s\n", r.ExportDiff)
		}
		if r.ImportDiff != "" {
			fmt.Printf("    import: %s\n", r.ImportDiff)
		}
	}
}
