// errcount summarizes decode failures of a DWG by object type: errcount file.dwg...
package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/LeeBhin/dwgconv/internal/dwg"
)

func main() {
	for _, p := range os.Args[1:] {
		d, err := dwg.Load(p)
		if err != nil {
			fmt.Println(p, err)
			continue
		}
		cnt := map[string]int{}
		var bad []string
		for _, o := range d.Objects {
			if o.DecodeErr != nil {
				k := o.TypeName(d.Classes.Classes)
				cnt[k]++
				if len(bad) < 12 {
					bad = append(bad, fmt.Sprintf("%X %s: %v", o.Handle, k, o.DecodeErr))
				}
			}
		}
		var keys []string
		for k := range cnt {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Printf("%s: %d objects, %d failed\n", p, len(d.Objects), len(bad))
		for _, k := range keys {
			fmt.Printf("  %s x%d\n", k, cnt[k])
		}
		for _, b := range bad {
			fmt.Println("   ", b)
		}
	}
}
