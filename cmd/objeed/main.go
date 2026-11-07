// objeed prints the EED (xdata) items of objects as raw bytes: objeed file.dwg <hexhandle>...
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/LeeBhin/dwgconv/internal/dwg"
)

func main() {
	d, err := dwg.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	for _, s := range os.Args[2:] {
		h, _ := strconv.ParseUint(s, 16, 64)
		o := d.ByHandle[h]
		if o == nil {
			fmt.Printf("%X: missing\n", h)
			continue
		}
		fmt.Printf("%X %s type=%d eed=%d err=%v left=%d/%d\n", h, o.TypeName(d.Classes.Classes), o.Type, len(o.EED), o.DecodeErr, o.LeftData, o.LeftHandles)
		for i, e := range o.EED {
			app := d.ByHandle[e.App.H]
			name := "?"
			if app != nil {
				if a, ok := app.Body.(*dwg.AppID); ok {
					name = a.Name
				}
			}
			fmt.Printf("  [%d] app=%X(%s) data=% X\n", i, e.App.H, name, e.Data)
		}
	}
}
