// dwgcheck decodes every object, re-encodes it and reports per type whether the bytes are identical.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"sort"

	"dwgconv/internal/dwg"
)

type stat struct {
	n, decoded, exact, errs int
	firstErr                string
	firstDiff               uint64
}

func main() {
	verbose := flag.Bool("v", false, "list classes")
	show := flag.String("show", "", "hex handle: print original and re-encoded bytes")
	flag.Parse()
	total := map[string]*stat{}
	for _, path := range flag.Args() {
		d, err := dwg.Load(path)
		if err != nil {
			fmt.Printf("%s: %v\n", path, err)
			continue
		}
		fmt.Printf("%s: %d objects, %d classes\n", path, len(d.Objects), len(d.Classes.Classes))
		if *verbose {
			for _, k := range d.Classes.Classes {
				fmt.Printf("  class %d %s (%s) item=%#x\n", k.Num, k.DXFName, k.CppName, k.ItemClass)
			}
		}
		if *show != "" {
			var h uint64
			fmt.Sscanf(*show, "%x", &h)
			if o := d.ByHandle[h]; o != nil {
				fmt.Printf("orig %x\nenc  %x\nleft data=%d handles=%d owner=%v xdicmiss=%v xdic=%v reactors=%v\nent=%+v\n",
					o.Raw, o.Encode(d.Version, d.Text), o.LeftData, o.LeftHandles, o.Owner, o.XDicMiss, o.XDic, o.Reactors, o.Ent)
			}
		}
		for _, o := range d.Objects {
			name := o.TypeName(d.Classes.Classes)
			s := total[name]
			if s == nil {
				s = &stat{}
				total[name] = s
			}
			s.n++
			if o.DecodeErr != nil {
				s.errs++
				if s.firstErr == "" {
					s.firstErr = fmt.Sprintf("%x: %v", o.Handle, o.DecodeErr)
				}
				continue
			}
			if o.Body == nil {
				continue
			}
			s.decoded++
			if bytes.Equal(o.Encode(d.Version, d.Text), o.Raw) {
				s.exact++
			} else if s.firstDiff == 0 {
				s.firstDiff = o.Handle
			}
		}
	}
	names := make([]string, 0, len(total))
	for k := range total {
		names = append(names, k)
	}
	sort.Slice(names, func(i, j int) bool { return total[names[i]].n > total[names[j]].n })
	fmt.Printf("%-28s %7s %7s %7s %5s\n", "type", "count", "decoded", "exact", "err")
	for _, k := range names {
		s := total[k]
		extra := ""
		if s.firstErr != "" {
			extra = " " + s.firstErr
		} else if s.firstDiff != 0 {
			extra = fmt.Sprintf(" first diff %x", s.firstDiff)
		}
		fmt.Printf("%-28s %7d %7d %7d %5d%s\n", k, s.n, s.decoded, s.exact, s.errs, extra)
	}
}
