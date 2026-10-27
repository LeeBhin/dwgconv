package container

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// R21Layout describes the pages of an R2007 file (diagnostics): the page map and, per section,
// the section map page entries next to the file size of each page.
func R21Layout(raw []byte) (string, error) {
	h, err := readR21Header(raw)
	if err != nil {
		return "", err
	}
	F := h.Fields
	var b strings.Builder
	pm, err := readR21SysPage(raw, int64(F[r21PagesMapOffset])+r21Base, F[r21PagesMapSizeCompressed],
		F[r21PagesMapSizeUncomp], F[r21PagesMapCorrection])
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&b, "page map: comp %#x uncomp %#x factor %d | section map: comp %#x uncomp %#x factor %d\n",
		F[r21PagesMapSizeCompressed], F[r21PagesMapSizeUncomp], F[r21PagesMapCorrection],
		F[r21SectionsMapSizeComp], F[r21SectionsMapSizeUncomp], F[r21SectionsMapCorrection])
	fileSize := map[int64]int64{}
	off := map[int64]int64{}
	var o int64
	for p := 0; p+16 <= len(pm); p += 16 {
		size := int64(binary.LittleEndian.Uint64(pm[p:]))
		id := int64(binary.LittleEndian.Uint64(pm[p+8:]))
		if id < 0 {
			id = -id
		}
		fileSize[id], off[id] = size, o
		o += size
	}
	sm, err := readR21SysPage(raw, off[int64(F[r21SectionsMapId])]+r21Base, F[r21SectionsMapSizeComp],
		F[r21SectionsMapSizeUncomp], F[r21SectionsMapCorrection])
	if err != nil {
		return "", err
	}
	u := func(p int) uint64 { return binary.LittleEndian.Uint64(sm[p:]) }
	for p := 0; p+0x40 <= len(sm); {
		nl := int(u(p + 0x20))
		np := int(u(p + 0x38))
		name := ""
		for i := 0; i+1 < nl; i += 2 {
			if c := binary.LittleEndian.Uint16(sm[p+0x40+i:]); c != 0 {
				name += string(rune(c))
			}
		}
		fmt.Fprintf(&b, "%-22s max %#6x encoding %d pages %d\n", name, u(p+8), u(p+0x30), np)
		p += 0x40 + nl
		for k := 0; k < np; k++ {
			id := int64(u(p + 16))
			cs := int64(u(p + 32))
			blocks := (align(int(cs), 8) + 250) / 251
			fmt.Fprintf(&b, "   id %3d start %#7x size %#6x usize %#6x csize %#6x  file %#6x  blocks(c) %d file/255 %d\n",
				id, u(p), u(p+8), u(p+24), cs, fileSize[id], blocks, fileSize[id]/255)
			p += 0x38
		}
	}
	return b.String(), nil
}
