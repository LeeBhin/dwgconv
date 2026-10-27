// objdiff prints the same object from two DWG files side by side:
//
//	objdiff a.dwg b.dwg <hexhandle>...
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/LeeBhin/dwgconv/internal/dwg"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: objdiff a.dwg b.dwg <hexhandle>...")
		os.Exit(2)
	}
	a, err := dwg.Load(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	b, err := dwg.Load(os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, s := range os.Args[3:] {
		h, _ := strconv.ParseUint(s, 16, 64)
		for _, d := range []*dwg.Document{a, b} {
			o := d.ByHandle[h]
			if o == nil {
				fmt.Printf("%X: missing\n", h)
				continue
			}
			fmt.Printf("%X %s type=%d raw=%X\n  owner=%X reactors=%v xdicmiss=%v xdic=%X eed=%d err=%v\n  ent=%+v\n  body=%+v\n",
				h, o.TypeName(d.Classes.Classes), o.Type, o.Raw, o.Owner.H, o.Reactors, o.XDicMiss, o.XDic.H, len(o.EED), o.DecodeErr, o.Ent, o.Body)
		}
	}
}
