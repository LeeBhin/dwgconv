package container

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

// TestR21PlainPagesMatchAutoCAD re-encodes the plain (encoding 1) pages of an AutoCAD 2007 file
// and expects the stored bytes back exactly.
func TestR21PlainPagesMatchAutoCAD(t *testing.T) {
	raw, err := os.ReadFile("../../.work/orig/s10.dwg")
	if err != nil {
		t.Skip(err)
	}
	h, _ := readR21Header(raw)
	F := h.Fields
	rnd := r21RandomBytes(F[r21RandomSeed])
	pm, _ := readR21SysPage(raw, int64(F[r21PagesMapOffset])+r21Base, F[r21PagesMapSizeCompressed], F[r21PagesMapSizeUncomp], F[r21PagesMapCorrection])
	var o int64
	pos, size := map[int64]int64{}, map[int64]int64{}
	for p := 0; p+16 <= len(pm); p += 16 {
		id := int64(binary.LittleEndian.Uint64(pm[p+8:]))
		if id < 0 {
			id = -id
		}
		pos[id], size[id] = o+r21Base, int64(binary.LittleEndian.Uint64(pm[p:]))
		o += size[id]
	}
	sm, _ := readR21SysPage(raw, pos[int64(F[r21SectionsMapId])], F[r21SectionsMapSizeComp], F[r21SectionsMapSizeUncomp], F[r21SectionsMapCorrection])
	u := func(p int) uint64 { return binary.LittleEndian.Uint64(sm[p:]) }
	n := 0
	for p := 0; p+0x40 <= len(sm); {
		max, nl, enc, np := int(u(p+8)), int(u(p+0x20)), int(u(p+0x30)), int(u(p+0x38))
		p += 0x40 + nl
		for k := 0; k < np; k++ {
			id, us := int64(u(p+16)), int(u(p+24))
			p += 0x38
			if enc != 1 {
				continue
			}
			// the alignment bytes after the data are whatever AutoCAD had in memory: take them as stored
			got := rsEncodePlain(raw[pos[id]:pos[id]+int64(align(us, 8))], max, rnd)
			if !bytes.Equal(got, raw[pos[id]:pos[id]+size[id]]) {
				t.Errorf("page %d (max %#x, usize %#x): re-encoded bytes differ", id, max, us)
			}
			n++
		}
	}
	if n == 0 {
		t.Fatal("no plain pages")
	}
}
