// dwginfo prints the container layout of DWG files and checks that a rewrite reads back identically.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"

	"github.com/LeeBhin/dwgconv/internal/container"
)

func main() {
	rewrite := flag.String("rewrite", "", "write the re-encoded container to this path (single input)")
	flag.Parse()
	for _, path := range flag.Args() {
		f, err := container.ReadFile(path)
		if err != nil {
			fmt.Printf("%s: %v\n", path, err)
			continue
		}
		fmt.Printf("%s %s prefix@2C=%#x @30=%#x\n", path, f.Version,
			binary.LittleEndian.Uint32(f.Prefix[0x2C:]), binary.LittleEndian.Uint32(f.Prefix[0x30:]))
		for _, s := range f.Sections {
			fmt.Printf("  %-24s id=%-2d size=%-8d comp=%d max=%#x\n", s.Name, s.ID, len(s.Data), s.Compressed, s.MaxSize)
		}
		out := f.Write()
		g, err := container.Read(out)
		if err != nil {
			fmt.Printf("  rewrite: read back failed: %v\n", err)
			continue
		}
		same := len(g.Sections) == len(f.Sections)
		for i := range f.Sections {
			if !same || !bytes.Equal(f.Sections[i].Data, g.Sections[i].Data) {
				same = false
				break
			}
		}
		fmt.Printf("  rewrite: %d -> %d bytes, sections identical: %v\n", fileSize(path), len(out), same)
		if *rewrite != "" {
			if err := os.WriteFile(*rewrite, out, 0o644); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func fileSize(p string) int64 {
	st, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return st.Size()
}
