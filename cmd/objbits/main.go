// objbits prints an object's body data bits (after the common data) and handle bits: objbits file.dwg <hexhandle>...
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
		data, hdl, err := d.BodyBits(o)
		fmt.Printf("== %X %s type=%d raw=%d bytes err=%v\n data(%d bits): %s\n hdl(%d bits): %s\n", h, o.TypeName(d.Classes.Classes), o.Type, len(o.Raw), err, len(data), data, len(hdl), hdl)
	}
}
