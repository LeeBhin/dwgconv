// variants writes copies of a DWG where one object is changed one field at a time, to find which
// difference a reader rejects:
//
//	variants orig.dwg ours.dwg <hexhandle> outdir
//
// Variant "all" replaces the object with ours entirely; "field-<name>" copies only that field
// (body or common) from ours into the original object.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/LeeBhin/dwgconv/internal/dwg"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Println("usage: variants orig.dwg ours.dwg <hexhandle> outdir")
		os.Exit(2)
	}
	h, _ := strconv.ParseUint(os.Args[3], 16, 64)
	out := os.Args[4]
	os.MkdirAll(out, 0o755)
	ours, err := dwg.Load(os.Args[2])
	if err != nil {
		panic(err)
	}
	ob := ours.ByHandle[h]
	write := func(name string, mod func(o *dwg.Object)) {
		d, err := dwg.Load(os.Args[1])
		if err != nil {
			panic(err)
		}
		o := d.ByHandle[h]
		mod(o)
		o.Raw = nil
		objs, handles := d.EncodeObjects()
		d.File.Section("AcDb:AcDbObjects").Data = objs
		d.File.Section("AcDb:Handles").Data = handles
		os.WriteFile(filepath.Join(out, name+".dwg"), d.File.Write(), 0o644)
	}
	write("all", func(o *dwg.Object) { *o = *ob })
	write("hasds", func(o *dwg.Object) { o.HasDS = ob.HasDS })
	diffFields("Body", reflect.ValueOf(ob.Body).Elem(), func(path []int, name string) {
		write("field-"+name, func(o *dwg.Object) {
			dst := reflect.ValueOf(o.Body).Elem().FieldByIndex(path)
			dst.Set(reflect.ValueOf(ob.Body).Elem().FieldByIndex(path))
		})
	}, nil, ours)
}

// diffFields calls f for every leaf field of the body (one level into embedded structs).
func diffFields(prefix string, v reflect.Value, f func([]int, string), path []int, _ *dwg.Document) {
	for i := 0; i < v.NumField(); i++ {
		sf := v.Type().Field(i)
		if !sf.IsExported() {
			continue
		}
		p := append(append([]int(nil), path...), i)
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
			diffFields(prefix, v.Field(i), f, p, nil)
			continue
		}
		f(p, sf.Name)
	}
}
