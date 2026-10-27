package container

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// R2007 (AC1021) file organization (§5).

const r21Base = 0x480 // stream offsets in the R21 header are relative to this position

// R21Header is the decoded R2007 file header (§5.2), kept so a rewrite can reuse its constants.
type R21Header struct {
	Fields [0x110 / 8]uint64
}

// field offsets in R21Header.Fields (byte offset / 8)
const (
	r21PagesMapCrcCompressed   = 0x10 / 8
	r21PagesMapCorrection      = 0x18 / 8
	r21PagesMapCrcSeed         = 0x20 / 8
	r21PagesMap2Offset         = 0x28 / 8
	r21PagesMap2Id             = 0x30 / 8
	r21PagesMapOffset          = 0x38 / 8
	r21PagesMapId              = 0x40 / 8
	r21Header2Offset           = 0x48 / 8
	r21PagesMapSizeCompressed  = 0x50 / 8
	r21PagesMapSizeUncomp      = 0x58 / 8
	r21PagesAmount             = 0x60 / 8
	r21PagesMaxId              = 0x68 / 8
	r21PagesMapCrcUncompressed = 0x80 / 8
	r21SectionsAmount          = 0xA0 / 8
	r21SectionsMapCrcUncomp    = 0xA8 / 8
	r21SectionsMapSizeComp     = 0xB0 / 8
	r21SectionsMap2Id          = 0xB8 / 8
	r21SectionsMapId           = 0xC0 / 8
	r21SectionsMapSizeUncomp   = 0xC8 / 8
	r21SectionsMapCrcComp      = 0xD0 / 8
	r21SectionsMapCorrection   = 0xD8 / 8
	r21SectionsMapCrcSeed      = 0xE0 / 8
	r21StreamVersion           = 0xE8 / 8
	r21CrcSeed                 = 0xF0 / 8
	r21CrcSeedEncoded          = 0xF8 / 8
	r21RandomSeed              = 0x100 / 8
	r21HeaderCrc               = 0x108 / 8
)

// rsDeinterleave returns the data bytes of n RS(255,k) blocks stored interleaved.
func rsDeinterleave(buf []byte, n, k int) []byte {
	out := make([]byte, 0, n*k)
	for b := 0; b < n; b++ {
		for j := 0; j < k; j++ {
			out = append(out, buf[b+n*j])
		}
	}
	return out
}

// rsSequential returns the data bytes of n non-interleaved RS(255,k) blocks: all data blocks
// come first, followed by all parity blocks (§5.13.1).
func rsSequential(buf []byte, n, k int) []byte {
	return append([]byte(nil), buf[:n*k]...)
}

func readR21Header(raw []byte) (*R21Header, error) {
	if len(raw) < 0x80+0x400 {
		return nil, fmt.Errorf("r2007: file too short")
	}
	d := rsDeinterleave(raw[0x80:], 3, 239)
	cl := int32(binary.LittleEndian.Uint32(d[0x18:]))
	var hdr []byte
	if cl < 0 {
		hdr = d[0x20 : 0x20-int(cl)]
	} else {
		var err error
		hdr, err = Decompress21(d[0x20:0x20+int(cl)], 0x110)
		if err != nil {
			return nil, fmt.Errorf("r2007 header: %w", err)
		}
	}
	h := &R21Header{}
	for i := range h.Fields {
		h.Fields[i] = binary.LittleEndian.Uint64(hdr[8*i:])
	}
	return h, nil
}

// readR21SysPage reads a system page (page map or section map, §5.3).
func readR21SysPage(raw []byte, pos int64, comp, uncomp, factor uint64) ([]byte, error) {
	aligned := (comp + 7) &^ 7
	blocks := int((aligned*factor + 238) / 239)
	end := pos + int64(blocks)*255
	if end > int64(len(raw)) {
		return nil, fmt.Errorf("system page out of range")
	}
	d := rsDeinterleave(raw[pos:end], blocks, 239)[:comp]
	if comp >= uncomp {
		return d[:uncomp], nil
	}
	return Decompress21(d, int(uncomp))
}

type r21Page struct {
	offset int64
	size   int64
}

func readR2007(raw []byte, f *File) error {
	h, err := readR21Header(raw)
	if err != nil {
		return err
	}
	f.R21 = h
	F := h.Fields
	pm, err := readR21SysPage(raw, int64(F[r21PagesMapOffset])+r21Base, F[r21PagesMapSizeCompressed],
		F[r21PagesMapSizeUncomp], F[r21PagesMapCorrection])
	if err != nil {
		return fmt.Errorf("r2007 page map: %w", err)
	}
	pages := map[int64]r21Page{}
	var off int64
	for p := 0; p+16 <= len(pm); p += 16 {
		size := int64(binary.LittleEndian.Uint64(pm[p:]))
		id := int64(binary.LittleEndian.Uint64(pm[p+8:]))
		if id < 0 {
			id = -id
		}
		pages[id] = r21Page{off, size}
		off += size
	}
	smPage, ok := pages[int64(F[r21SectionsMapId])]
	if !ok {
		return fmt.Errorf("r2007: section map page missing")
	}
	sm, err := readR21SysPage(raw, smPage.offset+r21Base, F[r21SectionsMapSizeComp],
		F[r21SectionsMapSizeUncomp], F[r21SectionsMapCorrection])
	if err != nil {
		return fmt.Errorf("r2007 section map: %w", err)
	}
	u64 := func(p int) uint64 { return binary.LittleEndian.Uint64(sm[p:]) }
	for p := 0; p+0x40 <= len(sm); {
		s := &Section{}
		size := u64(p)
		s.MaxSize = int32(u64(p + 8))
		s.Encrypted = int32(u64(p + 0x10))
		s.Hash = u64(p + 0x18)
		nameLen := int(u64(p + 0x20))
		s.Unknown = uint32(u64(p + 0x28))
		s.Encoding = int32(u64(p + 0x30))
		numPages := int(u64(p + 0x38))
		p += 0x40
		if nameLen > 0 { // byte length including the UTF-16 terminator (observed)
			u := make([]uint16, 0, nameLen/2)
			for i := 0; i+1 < nameLen; i += 2 {
				c := binary.LittleEndian.Uint16(sm[p+i:])
				if c == 0 {
					break
				}
				u = append(u, c)
			}
			s.Name = string(utf16.Decode(u))
			p += nameLen
		}
		s.Data = make([]byte, size)
		for k := 0; k < numPages; k++ {
			start := int64(u64(p))
			id := int64(u64(p + 0x10))
			usize := int64(u64(p + 0x18))
			csize := int64(u64(p + 0x20))
			p += 0x38
			pg, ok := pages[id]
			if !ok {
				return fmt.Errorf("r2007 section %s: page %d missing", s.Name, id)
			}
			plain, err := readR21DataPage(raw, pg.offset+r21Base, csize, usize, s.Encoding)
			if err != nil {
				return fmt.Errorf("r2007 section %s page %d: %w", s.Name, id, err)
			}
			if start < int64(len(s.Data)) {
				copy(s.Data[start:], plain)
			}
		}
		f.Sections = append(f.Sections, s)
	}
	return nil
}

// readR21DataPage reads one data page: RS(255,251) blocks, interleaved when encoding is 4.
func readR21DataPage(raw []byte, pos, comp, uncomp int64, encoding int32) ([]byte, error) {
	aligned := (comp + 7) &^ 7
	blocks := int((aligned + 250) / 251)
	end := pos + int64(blocks)*255
	if end > int64(len(raw)) {
		return nil, fmt.Errorf("data page out of range")
	}
	var d []byte
	if encoding == 4 {
		d = rsDeinterleave(raw[pos:end], blocks, 251)
	} else {
		d = rsSequential(raw[pos:end], blocks, 251)
	}
	d = d[:comp]
	if comp >= uncomp {
		return d[:uncomp], nil
	}
	return Decompress21(d, int(uncomp))
}
