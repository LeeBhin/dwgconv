package conv

import (
	"bufio"
	"bytes"
	_ "embed"
	"strconv"
	"strings"

	"github.com/LeeBhin/dwgconv/internal/dxf"
)

// Binary ACIS data (SAB, modeler version 2) is carried by the ACDSDATA section, as AutoCAD 2013+
// and ODA write it: the entity keeps 290 and a GUID (2), and an ACDSRECORD of the ASM_Data schema
// holds the bytes under the entity's handle (320). SAT text (version 1) stays inline as 1/3 groups.

// acdsSchemaDXF is the fixed schema part of the section (ODA's, group per line).
//
//go:embed acdsdata_schema.dxf
var acdsSchemaDXF []byte

type asmRecord struct {
	handle uint64
	data   []byte
}

// exportAcds writes the ACDSDATA section when an entity carried binary ACIS data.
func (x *ctx) exportAcds(w *dxf.Writer) {
	if len(x.asmOut) == 0 {
		return
	}
	w.Str(0, "SECTION")
	w.Str(2, "ACDSDATA")
	sc := bufio.NewScanner(bytes.NewReader(acdsSchemaDXF))
	for sc.Scan() {
		code, _ := strconv.Atoi(strings.TrimSpace(sc.Text()))
		if !sc.Scan() {
			break
		}
		w.Group(dxf.Group{Code: code, Value: sc.Text()})
	}
	for _, r := range x.asmOut {
		w.Str(0, "ACDSRECORD")
		w.Int(90, 1) // schema 1: AcDb3DSolid_ASM_Data
		w.Str(2, "AcDbDs::ID")
		w.Int(280, 10)
		w.Handle(320, r.handle)
		w.Str(2, "ASM_Data")
		w.Int(280, 15)
		w.Int(94, int64(len(r.data)))
		w.Binary(310, r.data)
	}
	w.Str(0, "ENDSEC")
}

// importAcds collects the ASM_Data records of the ACDSDATA section by entity handle.
func (x *ctx) importAcds(f *dxf.File) {
	x.asm = map[uint64][]byte{}
	s := f.Section("ACDSDATA")
	if s == nil {
		return
	}
	for _, r := range s.Records {
		if r.Type != "ACDSRECORD" {
			continue
		}
		var h uint64
		var data []byte
		asm := false
		for _, g := range r.Groups {
			switch g.Code {
			case 320:
				h = g.Handle()
			case 2:
				asm = g.Value == "ASM_Data"
			case 310:
				if asm {
					data = append(data, dxf.DecodeBinary(g.Value)...)
				}
			}
		}
		if h != 0 && len(data) > 0 {
			x.asm[h] = data
		}
	}
}
