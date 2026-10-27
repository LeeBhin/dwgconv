// Package dwgconv converts between DWG (R2004, R2007, R2010, R2013, R2018) and ASCII DXF.
//
// The functions are safe for concurrent use: each call works on its own data.
package dwgconv

import (
	"fmt"
	"io"

	"github.com/LeeBhin/dwgconv/internal/container"
	"github.com/LeeBhin/dwgconv/internal/conv"
	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

// DefaultVersion is the DWG version DXFToDWG writes when version is empty.
const DefaultVersion = conv.DefaultOutVersion

// DWGToDXF reads a DWG and writes it as an AC1032 ASCII DXF.
func DWGToDXF(dwgIn io.Reader, dxfOut io.Writer) error {
	raw, err := io.ReadAll(dwgIn)
	if err != nil {
		return err
	}
	f, err := container.Read(raw)
	if err != nil {
		return fmt.Errorf("dwgconv: %w", err)
	}
	d, err := dwg.Decode(f)
	if err != nil {
		return fmt.Errorf("dwgconv: %w", err)
	}
	return conv.ToDXF(d, dxfOut)
}

// DXFToDWG reads an ASCII DXF and returns a DWG of the given version ("R2004", "R2007", "R2010",
// "R2013" or "R2018"; empty means DefaultVersion). warnings lists DXF record types that were left
// out because they have no DWG mapping, as "TYPE×count".
func DXFToDWG(dxfIn io.Reader, version string) (dwgData []byte, warnings []string, err error) {
	if version == "" {
		version = DefaultVersion
	}
	f, err := dxf.Parse(dxfIn)
	if err != nil {
		return nil, nil, fmt.Errorf("dwgconv: %w", err)
	}
	return conv.FromDXF(f, version)
}
