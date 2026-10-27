// r21pages prints the page layout of an R2007 (AC1021) DWG: r21pages file.dwg
package main

import (
	"fmt"
	"os"

	"github.com/LeeBhin/dwgconv/internal/container"
)

func main() {
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	s, err := container.R21Layout(raw)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Print(s)
}
