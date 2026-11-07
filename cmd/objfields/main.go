// objfields prints the decoded fields of objects (nested structs, slices truncated) so a misparsed
// layout can be compared field by field with the reference DXF: objfields file.dwg <hexhandle>...
// With DWGCONV_KEEPBAD=1 the partially decoded body of a failed decode is shown too.
package main

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/LeeBhin/dwgconv/internal/dwg"
)

func dump(prefix string, v reflect.Value, depth int) {
	if depth > 12 || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			fmt.Printf("%s = nil\n", prefix)
			return
		}
		dump(prefix, v.Elem(), depth)
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" { // unexported
				continue
			}
			dump(prefix+"."+f.Name, v.Field(i), depth+1)
		}
	case reflect.Slice, reflect.Array:
		n := v.Len()
		if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
			b := v.Bytes()
			if len(b) > 56 {
				fmt.Printf("%s = [%d bytes] % X ... % X\n", prefix, n, b[:24], b[len(b)-32:])
				return
			}
			fmt.Printf("%s = [%d bytes] % X\n", prefix, n, b)
			return
		}
		if v.Type().Elem().Kind() == reflect.Float64 && n <= 16 {
			var parts []string
			for i := 0; i < n; i++ {
				parts = append(parts, strconv.FormatFloat(v.Index(i).Float(), 'g', 10, 64))
			}
			fmt.Printf("%s = [%s]\n", prefix, strings.Join(parts, " "))
			return
		}
		fmt.Printf("%s = [%d]\n", prefix, n)
		for i := 0; i < n && i < 400; i++ {
			dump(fmt.Sprintf("%s[%d]", prefix, i), v.Index(i), depth+1)
		}
	default:
		fmt.Printf("%s = %v\n", prefix, v.Interface())
	}
}

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
		fmt.Printf("== %X %s type=%d raw=%d err=%v left=%d/%d eed=%d\n", h, o.TypeName(d.Classes.Classes), o.Type, len(o.Raw), o.DecodeErr, o.LeftData, o.LeftHandles, len(o.EED))
		n := len(o.Raw)
		if n > 48 {
			n = 48
		}
		fmt.Printf("raw head: % X\n", o.Raw[:n])
		if o.Ent != nil {
			fmt.Printf("ent: HasGraphic=%v graphic=%d bytes EntMode=%d HasDS=%v\n", o.Ent.HasGraphic, len(o.Ent.Graphic), o.Ent.EntMode, o.Ent.HasDS)
		}
		if o.Body == nil {
			fmt.Println("body = nil (set DWGCONV_KEEPBAD=1 to keep a partially decoded body)")
			continue
		}
		dump("body", reflect.ValueOf(o.Body), 0)
	}
}
