// seccmp compares the section data of two DWG files and, for AcDb:AcDbObjects, names the object
// (by file A's layout) that holds the first differing byte:
//
//	seccmp a.dwg b.dwg
package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/LeeBhin/dwgconv/internal/container"
	"github.com/LeeBhin/dwgconv/internal/dwg"
)

func main() {
	fa, err := container.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	fb, err := container.ReadFile(os.Args[2])
	if err != nil {
		panic(err)
	}
	for _, sa := range fa.Sections {
		sb := fb.Section(sa.Name)
		if sb == nil {
			fmt.Printf("%-24s only in A (%d bytes)\n", sa.Name, len(sa.Data))
			continue
		}
		if bytes.Equal(sa.Data, sb.Data) {
			fmt.Printf("%-24s identical (%d bytes)\n", sa.Name, len(sa.Data))
			continue
		}
		i := 0
		for i < len(sa.Data) && i < len(sb.Data) && sa.Data[i] == sb.Data[i] {
			i++
		}
		fmt.Printf("%-24s DIFFER: len %d vs %d, first diff at %d\n", sa.Name, len(sa.Data), len(sb.Data), i)
		if sa.Name == "AcDb:AcDbObjects" {
			da, err := dwg.Load(os.Args[1])
			if err != nil {
				fmt.Println(" ", err)
				continue
			}
			off := 4
			for k, o := range da.Objects {
				if i >= off && i < off+len(o.Raw) {
					fmt.Printf("  in A object #%d %X %s (off=%d size=%d, diff at +%d)\n", k, o.Handle, o.TypeName(da.Classes.Classes), off, len(o.Raw), i-off)
					lo, hi := max(0, i-8), min(len(sa.Data), i+24)
					fmt.Printf("  A: % X\n  B: % X\n", sa.Data[lo:hi], sb.Data[lo:min(len(sb.Data), hi)])
					break
				}
				off += len(o.Raw)
			}
			// count differing objects by comparing raw bytes per handle
			db, err := dwg.Load(os.Args[2])
			if err == nil {
				n := 0
				for h, o := range da.ByHandle {
					if p := db.ByHandle[h]; p == nil || !bytes.Equal(o.Raw, p.Raw) {
						if n < 10 {
							fmt.Printf("  differs: %X %s (A %d bytes, B %d)\n", h, o.TypeName(da.Classes.Classes), len(o.Raw), func() int {
								if p == nil {
									return -1
								}
								return len(p.Raw)
							}())
						}
						n++
					}
				}
				fmt.Printf("  objects differing by raw bytes: %d of %d\n", n, len(da.Objects))
			}
		}
	}
	for _, sb := range fb.Sections {
		if fa.Section(sb.Name) == nil {
			fmt.Printf("%-24s only in B (%d bytes)\n", sb.Name, len(sb.Data))
		}
	}
}
