// layout prints the R2004+ physical layout of DWG files: header fields, page map entries and the
// system pages, to compare a written file with an AutoCAD-written one.
package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

var magic = func() []byte {
	out := make([]byte, 0x100)
	seed := uint32(1)
	for i := range out {
		seed = seed*0x343FD + 0x269EC3
		out[i] = byte(seed >> 16)
	}
	return out
}()

func main() {
	for _, path := range os.Args[1:] {
		raw, err := os.ReadFile(path)
		if err != nil {
			fmt.Println(err)
			continue
		}
		h := make([]byte, 0x6C)
		for i := range h {
			h[i] = raw[0x80+i] ^ magic[i]
		}
		u32 := func(o int) uint32 { return binary.LittleEndian.Uint32(h[o:]) }
		u64 := func(o int) uint64 { return binary.LittleEndian.Uint64(h[o:]) }
		fmt.Printf("%s len=%d\n", path, len(raw))
		fmt.Printf("  gaps root/left/right=%d/%d/%d unk24=%d lastPageId=%d lastPageEnd=%#x secondHdr=%#x gapAmount=%d pageAmount=%d\n",
			u32(0x18), u32(0x1C), u32(0x20), u32(0x24), u32(0x28), u64(0x2C), u64(0x34), u32(0x3C), u32(0x40))
		fmt.Printf("  pageMapId=%d pageMapAddr=%#x(+0x100) secMapId=%d pageArraySize=%d gapArraySize=%d\n",
			u32(0x50), u64(0x54), u32(0x5C), u32(0x60), u32(0x64))
		fmt.Printf("  prefix 0x0B..0x18: % x\n", raw[0x0B:0x18])
		fmt.Printf("  prefix 0x20..0x48: % x\n", raw[0x20:0x48])
		pm := int64(u64(0x54)) + 0x100
		typ := binary.LittleEndian.Uint32(raw[pm:])
		dsz := binary.LittleEndian.Uint32(raw[pm+4:])
		csz := binary.LittleEndian.Uint32(raw[pm+8:])
		fmt.Printf("  page map @%#x type=%#x dsize=%d csize=%d -> data ends %#x\n", pm, typ, dsz, csz, pm+20+int64(csz))
		tail := raw[pm+20+int64(csz):]
		if len(tail) > 64 {
			tail = tail[:64]
		}
		fmt.Printf("  after page map data: % x\n", tail)
		sh := int64(u64(0x34))
		if sh+0x6C <= int64(len(raw)) {
			fmt.Printf("  second header == first: %v\n", string(raw[sh:sh+0x6C]) == string(raw[0x80:0xEC]))
		}
		fmt.Printf("  bytes 0xEC..0x100 == magic: %v\n", string(raw[0xEC:0x100]) == string(magic[0xEC:0x100]))
	}
}
