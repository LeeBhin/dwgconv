// secget writes a section's decompressed bytes to a file: secget file.dwg "AcDb:AcDsPrototype_1b" out.bin
package main

import (
	"fmt"
	"os"

	"github.com/LeeBhin/dwgconv/internal/container"
)

func main() {
	f, err := container.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	s := f.Section(os.Args[2])
	if s == nil {
		fmt.Println("no such section")
		os.Exit(1)
	}
	if err := os.WriteFile(os.Args[3], s.Data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("%s: %d bytes -> %s\n", os.Args[2], len(s.Data), os.Args[3])
}
