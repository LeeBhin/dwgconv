// undocdump prints, for each object of a class, its undecoded body bits next to the ODA DXF record.
//
//	undocdump CLASSNAME file.dwg ref.dxf
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/LeeBhin/dwgconv/internal/bits"
	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

func main() {
	cls, path, refPath := os.Args[1], os.Args[2], os.Args[3]
	d, err := dwg.Load(path)
	if err != nil {
		panic(err)
	}
	fh, _ := os.Open(refPath)
	ref, _ := dxf.Parse(fh)
	recs := map[uint64]*dxf.Record{}
	for _, s := range ref.Sections {
		for _, r := range s.Records {
			recs[r.Handle(5)] = r
		}
	}
	for _, o := range d.Objects {
		if o.TypeName(d.Classes.Classes) != cls {
			continue
		}
		b, ok := o.Body.(*dwg.RawBody)
		if !ok {
			fmt.Printf("%X: body %T\n", o.Handle, o.Body)
			continue
		}
		var sb strings.Builder
		for i := 0; i < b.DataBits; i++ {
			sb.WriteByte('0' + (b.Data[i>>3]>>(7-uint(i&7)))&1)
		}
		fmt.Printf("== %s %X dataBits=%d strBits=%d handles=%v\n%s\n", d.File.Version, o.Handle, b.DataBits, b.StrBits, b.Handles, sb.String())
		if b.StrBits > 0 {
			sr := bits.NewReader(b.Str)
			sr.End = b.StrBits
			var ss []string
			func() {
				defer func() { recover() }()
				for sr.Remaining() >= 16 {
					ss = append(ss, fmt.Sprintf("%q", sr.TU()))
				}
			}()
			fmt.Printf("strings: %s (left %d)\n", strings.Join(ss, " | "), sr.Remaining())
		}
		if f := os.Getenv("FMT"); f != "" {
			parse(b, f, d)
		}
		if r := recs[o.Handle]; r != nil {
			for _, g := range r.Groups {
				fmt.Printf("  %d=%s\n", g.Code, strings.TrimSpace(g.Value))
			}
		}
	}
}

// parse reads the data bits with a comma-separated list of bit codes (TV reads 8-bit text for R2004).
// A code may be prefixed with a repeat count, e.g. 3*BD.
func parse(b *dwg.RawBody, f string, d *dwg.Document) {
	r := bits.NewReader(b.Data)
	r.End = b.DataBits
	var out []string
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("  !!", e, "after:", strings.Join(out, " "))
		}
	}()
	for _, code := range strings.Split(f, ",") {
		n := 1
		if i := strings.Index(code, "*"); i > 0 {
			fmt.Sscanf(code[:i], "%d", &n)
			code = code[i+1:]
		}
		for k := 0; k < n; k++ {
			var v any
			switch code {
			case "B":
				v = r.B()
			case "BB":
				v = r.BB()
			case "BS":
				v = r.BS()
			case "BL":
				v = r.BL()
			case "BD":
				v = r.BD()
			case "RC":
				v = r.RC()
			case "RS":
				v = r.RS()
			case "RL":
				v = r.RL()
			case "RD":
				v = r.RD()
			case "3BD":
				v = [3]float64{r.BD(), r.BD(), r.BD()}
			case "2RD":
				v = [2]float64{r.RD(), r.RD()}
			case "TV":
				v = d.Text.Decode(r.TV8())
			case "CMC":
				i, rgb, fl := r.BS(), r.BL(), r.RC()
				v = fmt.Sprintf("cmc(%d,%x,%d)", i, uint32(rgb), fl)
			default:
				v = "?" + code
			}
			out = append(out, fmt.Sprintf("%s=%v", code, v))
		}
	}
	fmt.Printf("  parsed: %s\n  left %d bits: ", strings.Join(out, " "), r.Remaining())
	for r.Remaining() > 0 && r.Pos < r.End {
		fmt.Print(r.B())
	}
	fmt.Println()
}
