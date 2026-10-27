// cmpdwg compares two DWGs object by object (same handles): per type, how many objects exist in
// both, how many re-encode to identical bytes, and the first differing handle.
//
//	cmpdwg a.dwg b.dwg [TYPE...]
package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"

	"dwgconv/internal/dwg"
)

func main() {
	a, err := dwg.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	b, err := dwg.Load(os.Args[2])
	if err != nil {
		panic(err)
	}
	only := map[string]bool{}
	for _, t := range os.Args[3:] {
		only[t] = true
	}
	type st struct{ both, same, onlyA int; first uint64 }
	res := map[string]*st{}
	for _, oa := range a.Objects {
		n := oa.TypeName(a.Classes.Classes)
		if len(only) > 0 && !only[n] {
			continue
		}
		s := res[n]
		if s == nil {
			s = &st{}
			res[n] = s
		}
		ob := b.ByHandle[oa.Handle]
		if ob == nil {
			s.onlyA++
			continue
		}
		s.both++
		if bytes.Equal(oa.Raw, ob.Raw) {
			s.same++
		} else if s.first == 0 {
			s.first = oa.Handle
		}
	}
	names := make([]string, 0, len(res))
	for k := range res {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		s := res[k]
		if s.same != s.both || s.onlyA > 0 {
			fmt.Printf("%-24s both %5d same %5d onlyA %4d first %X\n", k, s.both, s.same, s.onlyA, s.first)
		}
	}
}
