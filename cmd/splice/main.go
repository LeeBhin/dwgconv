// splice copies sections from one DWG into another of the same version, to find which section a
// reader rejects:
//
//	splice from.dwg into.dwg out.dwg "AcDb:Header" ["AcDb:Classes" ...]
package main

import (
	"fmt"
	"os"

	"github.com/LeeBhin/dwgconv/internal/container"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Println(`usage: splice from.dwg into.dwg out.dwg "AcDb:Header" ...`)
		os.Exit(2)
	}
	from, err := container.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	into, err := container.ReadFile(os.Args[2])
	if err != nil {
		panic(err)
	}
	for _, name := range os.Args[4:] {
		s, d := from.Section(name), into.Section(name)
		if s == nil || d == nil {
			fmt.Println("missing section", name)
			continue
		}
		d.Data = s.Data
	}
	if err := os.WriteFile(os.Args[3], into.Write(), 0o644); err != nil {
		panic(err)
	}
}
