// secdump prints the first bytes of a section (debug tool): secdump file.dwg <section> <n>
package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/LeeBhin/dwgconv/internal/container"
)

func main() {
	f, err := container.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	s := f.Section(os.Args[2])
	if s == nil {
		fmt.Println("no section")
		return
	}
	n, _ := strconv.Atoi(os.Args[3])
	fmt.Print(hex.Dump(s.Data[:min(n, len(s.Data))]))
}
