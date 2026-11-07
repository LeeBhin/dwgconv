// hybrid builds a DWG from file A's objects with some of them replaced by file B's version (same
// handle) and some removed, to find which object makes a reader reject another one:
//
//	hybrid -diff a.dwg b.dwg                 lists A-only, B-only and differing handles
//	hybrid a.dwg b.dwg out.dwg [-replace h,h,...|@file] [-remove h,h,...|@file] [-add h,...|@file]
//
// -add inserts B-only objects into A. Handles are hex; @file reads one per line.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/LeeBhin/dwgconv/internal/dwg"
)

func handles(spec string) []uint64 {
	if spec == "" {
		return nil
	}
	var parts []string
	if strings.HasPrefix(spec, "@") {
		b, err := os.ReadFile(spec[1:])
		if err != nil {
			panic(err)
		}
		parts = strings.Fields(string(b))
	} else {
		parts = strings.Split(spec, ",")
	}
	var out []uint64
	for _, p := range parts {
		if p == "" {
			continue
		}
		h, err := strconv.ParseUint(strings.TrimSpace(p), 16, 64)
		if err != nil {
			panic(err)
		}
		out = append(out, h)
	}
	return out
}

func main() {
	diff := flag.Bool("diff", false, "list handle sets")
	replace := flag.String("replace", "", "handles to take from B")
	remove := flag.String("remove", "", "handles to drop from A")
	add := flag.String("add", "", "B-only handles to insert into A")
	sortH := flag.Bool("sort", false, "write A's objects in handle order (as the DXF import does) instead of file order")
	flag.Parse()
	args := flag.Args()
	a, err := dwg.Load(args[0])
	if err != nil {
		panic(err)
	}
	b, err := dwg.Load(args[1])
	if err != nil {
		panic(err)
	}
	if *diff {
		var aOnly, bOnly, same, differ []uint64
		for h, o := range a.ByHandle {
			p := b.ByHandle[h]
			switch {
			case p == nil:
				aOnly = append(aOnly, h)
			case bytes.Equal(o.Raw, p.Raw):
				same = append(same, h)
			default:
				differ = append(differ, h)
			}
		}
		for h := range b.ByHandle {
			if a.ByHandle[h] == nil {
				bOnly = append(bOnly, h)
			}
		}
		for _, l := range [][]uint64{aOnly, bOnly, same, differ} {
			sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })
		}
		pr := func(name string, l []uint64) {
			var sb strings.Builder
			for _, h := range l {
				sb.WriteString(strconv.FormatUint(h, 16))
				sb.WriteByte('\n')
			}
			os.WriteFile(name+".txt", []byte(sb.String()), 0o644)
			fmt.Printf("%s: %d (-> %s.txt)\n", name, len(l), name)
		}
		pr("a_only", aOnly)
		pr("b_only", bOnly)
		pr("same", same)
		pr("differ", differ)
		return
	}
	out := args[2]
	for _, h := range handles(*replace) {
		p := b.ByHandle[h]
		o := a.ByHandle[h]
		if p == nil || o == nil {
			fmt.Printf("replace %X: missing (a=%v b=%v)\n", h, o != nil, p != nil)
			continue
		}
		*o = *p
	}
	rm := map[uint64]bool{}
	for _, h := range handles(*remove) {
		rm[h] = true
	}
	if len(rm) > 0 {
		kept := a.Objects[:0]
		for _, o := range a.Objects {
			if rm[o.Handle] {
				delete(a.ByHandle, o.Handle)
				continue
			}
			kept = append(kept, o)
		}
		a.Objects = kept
	}
	for _, h := range handles(*add) {
		p := b.ByHandle[h]
		if p == nil || a.ByHandle[h] != nil {
			fmt.Printf("add %X: skipped\n", h)
			continue
		}
		a.Add(p)
	}
	if *sortH {
		sort.SliceStable(a.Objects, func(i, j int) bool { return a.Objects[i].Handle < a.Objects[j].Handle })
	}
	objs, hs := a.EncodeObjects()
	a.File.Section("AcDb:AcDbObjects").Data = objs
	a.File.Section("AcDb:Handles").Data = hs
	if err := os.WriteFile(out, a.File.Write(), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s objects=%d\n", out, len(a.Objects))
}
