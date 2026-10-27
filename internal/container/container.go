// Package container reads and writes the R2004+ DWG file organization (Open Design Specification §4, §6-8).
// It handles AC1018 (R2004), AC1024 (R2010), AC1027 (R2013) and AC1032 (R2018). AC1021 (R2007) differs and is not supported.
package container

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"sort"
)

const (
	pageTypeData       = 0x4163043B
	pageTypePageMap    = 0x41630E3B
	pageTypeSectionMap = 0x4163003B
	headerMagic        = "AcFssFcAJMB"
)

// magic is the 0x100-byte pseudo random sequence of §4.1.
var magic = func() []byte {
	out := make([]byte, 0x100)
	seed := uint32(1)
	for i := range out {
		seed = seed*0x343FD + 0x269EC3
		out[i] = byte(seed >> 16)
	}
	return out
}()

// PageChecksum is the section page checksum of §4.2.
func PageChecksum(seed uint32, data []byte) uint32 {
	s1, s2 := seed&0xFFFF, seed>>16
	for len(data) > 0 {
		n := min(0x15B0, len(data))
		for _, b := range data[:n] {
			s1 += uint32(b)
			s2 += s1
		}
		s1 %= 0xFFF1
		s2 %= 0xFFF1
		data = data[n:]
	}
	return s2<<16 | s1&0xFFFF
}

// Section is one logical data section with its decompressed content.
type Section struct {
	Name       string
	ID         int32
	MaxSize    int32 // max decompressed page size
	Compressed int32 // 1 = no, 2 = yes
	Encrypted  int32
	Unknown    uint32
	Data       []byte
	// R2007 only (§5.2): section hash code and page encoding (1 = plain, 4 = interleaved RS).
	Hash     uint64
	Encoding int32
	// R21Pages are the stored pages of an R2007 section as read (diagnostics: DWGCONV_R21_ORIG
	// makes the writer reuse them instead of re-compressing).
	R21Pages []R21StoredPage
}

// R21StoredPage is one R2007 data page as stored: logical start, sizes and compressed bytes.
type R21StoredPage struct {
	Start, USize int
	Comp         []byte
}

// File is a parsed DWG container.
type File struct {

	Version  string
	Prefix   [0x80]byte // raw bytes 0x00..0x7F (version, codepage, page addresses...)
	Header   [0x6C]byte // decrypted file header data
	Sections []*Section // in section map order
	R21      *R21Header // R2007 file header
	R21Check []byte     // R2007 header page check data (0x28 bytes at 0x3D8), reused on rewrite
	// R21CheckSeq is the header page check sequence value (reused on rewrite).
	R21CheckSeq uint64
}

func (f *File) Section(name string) *Section {
	for _, s := range f.Sections {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func ReadFile(path string) (*File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Read(raw)
}

type pageLoc struct{ addr, size int64 }

func Read(raw []byte) (*File, error) {
	if len(raw) < 0x100 {
		return nil, fmt.Errorf("container: file too short")
	}
	f := &File{Version: string(raw[:6])}
	switch f.Version {
	case "AC1018", "AC1024", "AC1027", "AC1032":
	case "AC1021":
		copy(f.Prefix[:], raw[:0x80])
		if err := readR2007(raw, f); err != nil {
			return nil, err
		}
		return f, nil
	default:
		return nil, fmt.Errorf("container: unsupported version %s", f.Version)
	}
	copy(f.Prefix[:], raw[:0x80])
	for i := range f.Header {
		f.Header[i] = raw[0x80+i] ^ magic[i]
	}
	h := f.Header[:]
	if string(h[:11]) != headerMagic {
		return nil, fmt.Errorf("container: bad header id")
	}
	var chk [0x6C]byte
	copy(chk[:], h)
	binary.LittleEndian.PutUint32(chk[0x68:], 0)
	if crc32.ChecksumIEEE(chk[:]) != binary.LittleEndian.Uint32(h[0x68:]) {
		return nil, fmt.Errorf("container: header CRC mismatch")
	}
	pageMapAddr := int64(binary.LittleEndian.Uint64(h[0x54:])) + 0x100
	sectionMapID := int32(binary.LittleEndian.Uint32(h[0x5C:]))

	pm, err := readSystemPage(raw, pageMapAddr, pageTypePageMap)
	if err != nil {
		return nil, fmt.Errorf("page map: %w", err)
	}
	pages := map[int32]pageLoc{}
	addr := int64(0x100)
	for p := 0; p+8 <= len(pm); {
		num := int32(binary.LittleEndian.Uint32(pm[p:]))
		size := int64(int32(binary.LittleEndian.Uint32(pm[p+4:])))
		p += 8
		if num < 0 {
			p += 16
		} else {
			pages[num] = pageLoc{addr, size}
		}
		addr += size
	}
	smLoc, ok := pages[sectionMapID]
	if !ok {
		return nil, fmt.Errorf("container: section map page %d missing", sectionMapID)
	}
	sm, err := readSystemPage(raw, smLoc.addr, pageTypeSectionMap)
	if err != nil {
		return nil, fmt.Errorf("section map: %w", err)
	}
	n := int(binary.LittleEndian.Uint32(sm))
	p := 20
	for i := 0; i < n; i++ {
		s := &Section{}
		size := int64(binary.LittleEndian.Uint64(sm[p:]))
		pageCount := int(binary.LittleEndian.Uint32(sm[p+8:]))
		s.MaxSize = int32(binary.LittleEndian.Uint32(sm[p+12:]))
		s.Unknown = binary.LittleEndian.Uint32(sm[p+16:])
		s.Compressed = int32(binary.LittleEndian.Uint32(sm[p+20:]))
		s.ID = int32(binary.LittleEndian.Uint32(sm[p+24:]))
		s.Encrypted = int32(binary.LittleEndian.Uint32(sm[p+28:]))
		s.Name = string(bytes.TrimRight(sm[p+32:p+96], "\x00"))
		if j := bytes.IndexByte([]byte(s.Name), 0); j >= 0 {
			s.Name = s.Name[:j]
		}
		p += 96
		s.Data = make([]byte, size)
		for k := 0; k < pageCount; k++ {
			num := int32(binary.LittleEndian.Uint32(sm[p:]))
			start := int64(binary.LittleEndian.Uint64(sm[p+8:]))
			p += 16
			loc, ok := pages[num]
			if !ok {
				return nil, fmt.Errorf("section %s: page %d missing", s.Name, num)
			}
			plain, err := readDataPage(raw, loc.addr, s)
			if err != nil {
				return nil, fmt.Errorf("section %s page %d: %w", s.Name, num, err)
			}
			if start < size {
				copy(s.Data[start:], plain[:min(int64(len(plain)), size-start)])
			}
		}
		f.Sections = append(f.Sections, s)
	}
	return f, nil
}

func readSystemPage(raw []byte, addr int64, want uint32) ([]byte, error) {
	if addr+20 > int64(len(raw)) {
		return nil, fmt.Errorf("address out of range")
	}
	typ := binary.LittleEndian.Uint32(raw[addr:])
	dsize := int(binary.LittleEndian.Uint32(raw[addr+4:]))
	csize := int(binary.LittleEndian.Uint32(raw[addr+8:]))
	ctype := binary.LittleEndian.Uint32(raw[addr+12:])
	cks := binary.LittleEndian.Uint32(raw[addr+16:])
	if typ != want {
		return nil, fmt.Errorf("page type %#x, want %#x", typ, want)
	}
	comp := raw[addr+20 : addr+20+int64(csize)]
	var hdr [20]byte
	copy(hdr[:], raw[addr:addr+16])
	if PageChecksum(PageChecksum(0, hdr[:]), comp) != cks {
		return nil, fmt.Errorf("system page checksum mismatch")
	}
	if ctype != 2 {
		return append([]byte(nil), comp...), nil
	}
	return Decompress(comp, dsize)
}

func readDataPage(raw []byte, addr int64, s *Section) ([]byte, error) {
	mask := uint32(0x4164536B) ^ uint32(addr)
	var w [8]uint32
	for i := range w {
		w[i] = binary.LittleEndian.Uint32(raw[addr+int64(4*i):]) ^ mask
	}
	if w[0] != pageTypeData {
		return nil, fmt.Errorf("page type %#x", w[0])
	}
	dataSize := int64(w[2])
	data := raw[addr+32 : addr+32+dataSize]
	if PageChecksum(0, data) != w[7] {
		return nil, fmt.Errorf("data checksum mismatch")
	}
	var hdr [32]byte
	for i := range w {
		v := w[i]
		if i == 6 {
			v = 0
		}
		binary.LittleEndian.PutUint32(hdr[4*i:], v)
	}
	if PageChecksum(w[7], hdr[:]) != w[6] {
		return nil, fmt.Errorf("header checksum mismatch")
	}
	if s.Compressed == 2 {
		return Decompress(data, int(s.MaxSize))
	}
	return data, nil
}

// streamOrder is the physical order of sections in the file (§4.5).
var streamOrder = []string{"", "AcDb:SummaryInfo", "AcDb:Preview", "AcDb:VBAProject", "AcDb:AppInfo",
	"AcDb:AppInfoHistory", "AcDb:FileDepList", "AcDb:AcDsPrototype_1b", "AcDb:RevHistory", "AcDb:Security",
	"AcDb:AcDbObjects", "AcDb:ObjFreeSpace", "AcDb:Template", "AcDb:Handles", "AcDb:Classes",
	"AcDb:AuxHeader", "AcDb:Header"}

func streamRank(name string) int {
	for i, n := range streamOrder {
		if n == name {
			return i
		}
	}
	return len(streamOrder)
}

func pad(buf []byte) []byte {
	if n := (0x20 - len(buf)%0x20) % 0x20; n > 0 {
		buf = append(buf, magic[:n]...)
	}
	return buf
}

func systemPage(typ uint32, plain []byte) []byte {
	comp := Compress(plain)
	var hdr [20]byte
	binary.LittleEndian.PutUint32(hdr[0:], typ)
	binary.LittleEndian.PutUint32(hdr[4:], uint32(len(plain)))
	binary.LittleEndian.PutUint32(hdr[8:], uint32(len(comp)))
	binary.LittleEndian.PutUint32(hdr[12:], 2)
	binary.LittleEndian.PutUint32(hdr[16:], PageChecksum(PageChecksum(0, hdr[:]), comp))
	out := append(hdr[:], comp...)
	// §4.3: the compressed data is followed by a second header chunk whose sizes and checksum are
	// zero. AutoCAD rejects system pages without it.
	var tail [20]byte
	binary.LittleEndian.PutUint32(tail[0:], typ)
	binary.LittleEndian.PutUint32(tail[12:], 2)
	return append(out, tail[:]...)
}

// Write serializes the container. Prefix page addresses (preview, summary info, app info...) are recomputed.
func (f *File) Write() []byte {
	if f.Version == "AC1021" {
		return f.writeR2007()
	}
	secs := append([]*Section(nil), f.Sections...)
	sort.SliceStable(secs, func(i, j int) bool { return streamRank(secs[i].Name) < streamRank(secs[j].Name) })

	type pageRef struct {
		num   int32
		size  int32
		start int64
	}
	out := make([]byte, 0x100, 0x100+64*1024)
	var pageMap [][2]int32
	secPages := map[*Section][]pageRef{}
	firstPageAddr := map[string]int64{}
	var pid int32
	for _, s := range secs {
		m := int(s.MaxSize)
		for start := 0; start < len(s.Data); start += m {
			chunk := make([]byte, m)
			copy(chunk, s.Data[start:])
			payload := chunk
			if s.Compressed == 2 {
				payload = Compress(chunk)
			}
			pid++
			addr := int64(len(out))
			if _, ok := firstPageAddr[s.Name]; !ok {
				firstPageAddr[s.Name] = addr
			}
			size := 32 + len(payload)
			size += (0x20 - size%0x20) % 0x20
			dcks := PageChecksum(0, payload)
			w := [8]uint32{pageTypeData, uint32(s.ID), uint32(len(payload)), uint32(size),
				uint32(start), uint32(int64(start) >> 32), 0, dcks}
			var hdr [32]byte
			for i, v := range w {
				binary.LittleEndian.PutUint32(hdr[4*i:], v)
			}
			w[6] = PageChecksum(dcks, hdr[:])
			mask := uint32(0x4164536B) ^ uint32(addr)
			for i, v := range w {
				binary.LittleEndian.PutUint32(hdr[4*i:], v^mask)
			}
			out = append(out, hdr[:]...)
			out = append(out, payload...)
			out = pad(out)
			pageMap = append(pageMap, [2]int32{pid, int32(int64(len(out)) - addr)})
			secPages[s] = append(secPages[s], pageRef{pid, int32(len(payload)), int64(start)})
		}
	}

	// Section map (§4.5), in the original section map order.
	sm := make([]byte, 20)
	binary.LittleEndian.PutUint32(sm[0:], uint32(len(f.Sections)))
	binary.LittleEndian.PutUint32(sm[4:], 2)
	binary.LittleEndian.PutUint32(sm[8:], 0x7400)
	binary.LittleEndian.PutUint32(sm[16:], uint32(len(f.Sections)))
	for _, s := range f.Sections {
		pl := secPages[s]
		sm = binary.LittleEndian.AppendUint64(sm, uint64(len(s.Data)))
		sm = binary.LittleEndian.AppendUint32(sm, uint32(len(pl)))
		sm = binary.LittleEndian.AppendUint32(sm, uint32(s.MaxSize))
		sm = binary.LittleEndian.AppendUint32(sm, s.Unknown)
		sm = binary.LittleEndian.AppendUint32(sm, uint32(s.Compressed))
		sm = binary.LittleEndian.AppendUint32(sm, uint32(s.ID))
		sm = binary.LittleEndian.AppendUint32(sm, uint32(s.Encrypted))
		var name [64]byte
		copy(name[:], s.Name)
		sm = append(sm, name[:]...)
		for _, r := range pl {
			sm = binary.LittleEndian.AppendUint32(sm, uint32(r.num))
			sm = binary.LittleEndian.AppendUint32(sm, uint32(r.size))
			sm = binary.LittleEndian.AppendUint64(sm, uint64(r.start))
		}
	}
	secMapID := pid + 2 // gap of one page number before the system pages (§4.4)
	secMapAddr := int64(len(out))
	out = pad(append(out, systemPage(pageTypeSectionMap, sm)...))
	pageMap = append(pageMap, [2]int32{secMapID, int32(int64(len(out)) - secMapAddr)})

	// Page map, which lists itself; iterate until its own size is stable.
	pmID := secMapID + 1
	pmAddr := int64(len(out))
	var page []byte
	pmSize := int32(0)
	for iter := 0; ; iter++ {
		buf := make([]byte, 0, 8*(len(pageMap)+1))
		for _, e := range append(pageMap, [2]int32{pmID, pmSize}) {
			buf = binary.LittleEndian.AppendUint32(buf, uint32(e[0]))
			buf = binary.LittleEndian.AppendUint32(buf, uint32(e[1]))
		}
		page = systemPage(pageTypePageMap, buf)
		// the page map extent also covers the second file header that directly follows its data
		sz := int32(len(page) + 0x6C)
		sz += (0x20 - sz%0x20) % 0x20
		if sz == pmSize || iter >= 4 && sz < pmSize {
			break // a page extent may be larger than its data (§4.6); the rest is padded below
		}
		if iter >= 4 {
			sz = max(sz, pmSize) // the compressed size can alternate between two values
		}
		pmSize = sz
	}
	out = append(out, page...)
	secondHdr := int64(len(out)) // as AutoCAD writes it: right after the page map, ending the file

	copy(out[0:0x80], f.Prefix[:])
	setAddr := func(off int, name string) {
		if a, ok := firstPageAddr[name]; ok {
			binary.LittleEndian.PutUint32(out[off:], uint32(a+0x20))
		} else {
			binary.LittleEndian.PutUint32(out[off:], 0)
		}
	}
	setAddr(0x0D, "AcDb:Preview")
	setAddr(0x20, "AcDb:SummaryInfo")
	setAddr(0x24, "AcDb:VBAProject")
	setAddr(0x2C, "AcDb:AppInfo")
	setAddr(0x30, "AcDb:AppInfoHistory")

	h := f.Header
	binary.LittleEndian.PutUint32(h[0x18:], 0)
	binary.LittleEndian.PutUint32(h[0x1C:], 0)
	binary.LittleEndian.PutUint32(h[0x20:], 0)
	binary.LittleEndian.PutUint32(h[0x28:], uint32(pmID))
	binary.LittleEndian.PutUint64(h[0x2C:], uint64(pmAddr+int64(pmSize)-0x100))
	binary.LittleEndian.PutUint64(h[0x34:], uint64(secondHdr))
	binary.LittleEndian.PutUint32(h[0x3C:], 0)
	binary.LittleEndian.PutUint32(h[0x40:], uint32(len(pageMap)+1))
	binary.LittleEndian.PutUint32(h[0x50:], uint32(pmID))
	binary.LittleEndian.PutUint64(h[0x54:], uint64(pmAddr-0x100))
	binary.LittleEndian.PutUint32(h[0x5C:], uint32(secMapID))
	binary.LittleEndian.PutUint32(h[0x60:], uint32(pmID))
	binary.LittleEndian.PutUint32(h[0x64:], 0)
	binary.LittleEndian.PutUint32(h[0x68:], 0)
	binary.LittleEndian.PutUint32(h[0x68:], crc32.ChecksumIEEE(h[:]))
	var enc [0x6C]byte
	for i := range enc {
		enc[i] = h[i] ^ magic[i]
	}
	copy(out[0x80:], enc[:])
	copy(out[0xEC:0x100], magic[0xEC:0x100])
	return append(out, enc[:]...)
}

