// objlist lists the objects of one type with chosen body fields, to compare a field across files or
// find samples of a variant: objlist TYPE field,field,... file.dwg...
package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/LeeBhin/dwgconv/internal/dwg"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: objlist TYPE field,field,... file.dwg...")
		return
	}
	typ := os.Args[1]
	fields := strings.Split(os.Args[2], ",")
	for _, p := range os.Args[3:] {
		d, err := dwg.Load(p)
		if err != nil {
			fmt.Println(p, err)
			continue
		}
		for _, o := range d.Objects {
			if o.TypeName(d.Classes.Classes) != typ {
				continue
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "%s %X", p, o.Handle)
			if o.DecodeErr != nil {
				fmt.Fprintf(&sb, " ERR=%v", o.DecodeErr)
			}
			if o.Body != nil {
				v := reflect.ValueOf(o.Body).Elem()
				for _, f := range fields {
					fv := v.FieldByName(f)
					if !fv.IsValid() {
						fmt.Fprintf(&sb, " %s=?", f)
						continue
					}
					if r, ok := fv.Interface().(dwg.Ref); ok {
						fmt.Fprintf(&sb, " %s=%d:%X", f, r.Code, r.H)
					} else {
						fmt.Fprintf(&sb, " %s=%v", f, fv.Interface())
					}
				}
			}
			fmt.Println(sb.String())
		}
	}
}
