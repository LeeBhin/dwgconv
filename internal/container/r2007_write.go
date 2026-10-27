package container

import (
	"encoding/binary"
	"os"
	"unicode/utf16"
)

// Writing R2007 (AC1021) files (§5). Pages are stored uncompressed (allowed: a page is stored
// as is when compression does not make it shorter) and Reed-Solomon encoded, interleaved.

// r21StreamOrder is the physical order of data sections (§5.1).
var r21StreamOrder = []string{"AcDb:SummaryInfo", "AcDb:Preview", "AcDb:VBAProject", "AcDb:AppInfo",
	"AcDb:AppInfoHistory", "AcDb:FileDepList", "AcDb:RevHistory", "AcDb:Security", "AcDb:AcDbObjects",
	"AcDb:ObjFreeSpace", "AcDb:Template", "AcDb:Handles", "AcDb:Classes", "AcDb:AuxHeader", "AcDb:Header"}

// r21Hash is the section hash code written in the section map (§5.2 table; others from the sample).
var r21Hash = map[string]uint64{
	"AcDb:Security": 0x4a0204ea, "AcDb:FileDepList": 0x6c4205ca, "AcDb:VBAProject": 0x586e0544,
	"AcDb:AppInfo": 0x3fa0043e, "AcDb:Preview": 0x40aa0473, "AcDb:SummaryInfo": 0x717a060f,
	"AcDb:RevHistory": 0x60a205b3, "AcDb:AcDbObjects": 0x674c05a9, "AcDb:ObjFreeSpace": 0x77e2061f,
	"AcDb:Template": 0x4a1404ce, "AcDb:Handles": 0x3f6e0450, "AcDb:Classes": 0x3f54045f,
	"AcDb:AuxHeader": 0x54f0050a, "AcDb:Header": 0x32b803d9, "AcDb:AppInfoHistory": 0x96de0737,
}

// r21Plain are the sections AutoCAD stores uncompressed and non-interleaved (encoding 1, §5.2
// table); readers take e.g. the preview straight from its page.
var r21Plain = map[string]bool{"AcDb:SummaryInfo": true, "AcDb:Preview": true, "AcDb:AppInfo": true,
	"AcDb:AppInfoHistory": true, "AcDb:FileDepList": true, "AcDb:VBAProject": true}

// r21RandomBytes is the CRC random encoding table of §5.11 as little-endian bytes.
func r21RandomBytes(seed uint64) []byte {
	t := make([]uint32, 0x270)
	t[0] = uint32(seed)*0x343FD + 0x269EC3
	t[1] = uint32(seed>>32)*0x343FD + 0x269EC3
	v := t[1]
	for i := uint32(2); i < 0x270; i++ {
		v = (v>>0x1E^v)*0x6C078965 + i
		t[i] = v
	}
	out := make([]byte, 0, 4*len(t))
	for _, x := range t {
		out = binary.LittleEndian.AppendUint32(out, x)
	}
	return out
}

// randomFill returns n bytes of the random table from its start, repeating it if needed.
func randomFill(rnd []byte, n int) []byte {
	out := make([]byte, 0, n)
	for len(out) < n {
		out = append(out, rnd[:min(len(rnd), n-len(out))]...)
	}
	return out
}

// plainPageSize is the file size of a plain (encoding 1) page: room for max bytes of data in
// RS(255,251) blocks, the last one shortened, aligned to 0x20.
func plainPageSize(max int) int {
	full, rem := max/251, max%251
	size := full * 255
	if rem > 0 {
		size += rem + 4
	}
	return align(size, 0x20)
}

// rsEncodePlain writes a plain (encoding 1) page as AutoCAD does: the data (8-aligned), then the
// RS(255,251) parity of each data block — the last block is completed with the random table for
// the parity but those bytes are not stored — then the random table fills the page (observed).
func rsEncodePlain(data []byte, max int, rnd []byte) []byte {
	al := align(len(data), 8)
	out := make([]byte, al, plainPageSize(max))
	copy(out, data)
	for b := 0; b < al; b += 251 {
		blk := append([]byte(nil), out[b:min(b+251, al)]...)
		blk = append(blk, randomFill(rnd, 251-len(blk))...)
		out = append(out, rsData.encode(blk)...)
	}
	return append(out, randomFill(rnd, cap(out)-len(out))...)
}

// rsEncodeInterleaved encodes data (padded with zeros to whole blocks) into n interleaved
// RS(255,k) codewords: byte j of block b goes to position b + n*j.
func rsEncodeInterleaved(data []byte, rs *rsCode) []byte {
	k := rs.k
	n := (len(data) + k - 1) / k
	if n == 0 {
		n = 1
	}
	buf := make([]byte, n*k)
	copy(buf, data)
	out := make([]byte, n*255)
	for b := 0; b < n; b++ {
		blk := buf[b*k : (b+1)*k]
		par := rs.encode(blk)
		for j := 0; j < k; j++ {
			out[b+n*j] = blk[j]
		}
		for j := 0; j < 255-k; j++ {
			out[b+n*(k+j)] = par[j]
		}
	}
	return out
}

func align(n, a int) int { return (n + a - 1) / a * a }

// systemPageSize is GetSystemPageSize of §5.3.1.
func systemPageSize(dataSize int) int {
	aligned := align(dataSize, 8)
	size := (aligned*2 + 238) / 239 * 255
	if size < 0x400 {
		return 0x400
	}
	return align(size, 0x20)
}

// encodeSystemPage compresses data (kept as is when that is not shorter), repeats it to fill the
// page and RS(255,239) encodes it interleaved. It returns the stored (compressed) bytes too.
func encodeSystemPage(plain []byte, size int) (page []byte, factor int, data []byte) {
	data = Compress21(plain)
	if len(data) >= len(plain) {
		data = plain
	}
	if size == 0 {
		size = systemPageSize(len(plain))
	}
	aligned := align(len(data), 8)
	maxBlocks := size / 255
	factor = maxBlocks * 239 / aligned
	if factor < 1 {
		factor = 1
	}
	rep := make([]byte, 0, aligned*factor)
	for i := 0; i < factor; i++ {
		rep = append(rep, data...)
		rep = append(rep, make([]byte, aligned-len(data))...)
	}
	enc := rsEncodeInterleaved(rep, rsSystem)
	page = make([]byte, size)
	copy(page, enc)
	return page, factor, data
}

type r21PageOut struct {
	id    int64
	bytes []byte
}

type r21SectionPage struct {
	start, size, id, usize, csize int64
	checksum                      uint32
	crc                           uint64
}

func (f *File) writeR2007() []byte {
	byName := map[string]*Section{}
	for _, s := range f.Sections {
		byName[s.Name] = s
	}
	var order []*Section
	for _, n := range r21StreamOrder {
		if s := byName[n]; s != nil {
			order = append(order, s)
		}
	}
	// Plan the pages first: the page map size depends on the page count, and the final address of
	// the preview page must be known before it is encoded.
	type plan struct {
		s           *Section
		start, max  int
		chunk, comp []byte
	}
	var plans []plan
	for _, s := range order {
		max := int(s.MaxSize)
		if max <= 0 {
			max = 0xF800
		}
		if os.Getenv("DWGCONV_R21_ORIG") != "" && len(s.R21Pages) > 0 && !r21Plain[s.Name] {
			for _, rp := range s.R21Pages {
				plans = append(plans, plan{s, rp.Start, max, s.Data[rp.Start : rp.Start+rp.USize], rp.Comp})
			}
			continue
		}
		for start := 0; start < len(s.Data); {
			chunk := s.Data[start:min(start+max, len(s.Data))]
			comp := chunk
			if !r21Plain[s.Name] {
				comp = Compress21(chunk)
				if len(comp) >= len(chunk) {
					// Stored as is. A raw page of several interleaved blocks is read differently by
					// different readers (AutoCAD only writes single-block raw pages), so raw pages are
					// cut to one RS block (248 bytes, 8-aligned <= 251).
					if len(chunk) > 248 {
						chunk = chunk[:248]
					}
					comp = chunk
				}
			}
			plans = append(plans, plan{s, start, max, chunk, comp})
			start += len(chunk)
		}
	}
	gap := 0 // AutoCAD leaves a gap page after the page map; readers do not need it
	nEntries := len(plans) + 4
	pmBase := int64(r21Base + r21PageMapPageSize(nEntries) + gap)
	seed := uint64(r21DefaultRandomSeed)
	if f.R21 != nil {
		seed = f.R21.Fields[r21RandomSeed]
	}
	rnd := r21RandomBytes(seed)
	var dataPages []r21PageOut
	secPages := map[*Section][]r21SectionPage{}
	firstPagePos := map[string]int64{}
	var id int64
	stream := int64(0) // position after the page map page
	for _, pl := range plans {
		s, chunk, comp := pl.s, pl.chunk, pl.comp
		id++
		var enc []byte
		if r21Plain[s.Name] {
			if s.Name == "AcDb:Preview" && pl.start == 0 {
				chunk = rebasePreview(chunk, uint32(pmBase+stream))
				comp = chunk
			}
			enc = rsEncodePlain(chunk, pl.max, rnd)
		} else {
			stored := make([]byte, align(len(comp), 8))
			copy(stored, comp)
			enc = rsEncodeInterleaved(stored, rsData)
		}
		pageBytes := make([]byte, align(len(enc), 0x20))
		copy(pageBytes, enc)
		dataPages = append(dataPages, r21PageOut{id, pageBytes})
		if _, ok := firstPagePos[s.Name]; !ok {
			firstPagePos[s.Name] = stream
		}
		stream += int64(len(pageBytes))
		secPages[s] = append(secPages[s], r21SectionPage{
			// the page size field is the page's own size: AutoCAD rejects a section max here
			start: int64(pl.start), size: int64(len(chunk)), id: id, usize: int64(len(chunk)),
			csize: int64(len(comp)), checksum: pageChecksum21(0, chunk),
			crc: crc64Mirrored(updateSeed1(0, uint32(len(comp))), comp),
		})
	}

	// section map (§5.2), in the original section map order, ending with the empty section
	var sm []byte
	put := func(v uint64) { sm = binary.LittleEndian.AppendUint64(sm, v) }
	sections := 0
	for _, s := range f.Sections {
		if s.Name == "" {
			continue
		}
		sections++
		name := utf16.Encode([]rune(s.Name))
		pages := secPages[s]
		put(uint64(len(s.Data)))
		put(uint64(s.MaxSize))
		put(0) // not encrypted
		h := s.Hash
		if h == 0 {
			h = r21Hash[s.Name]
		}
		put(h)
		put(uint64(len(name)*2 + 2))
		put(0)
		if r21Plain[s.Name] {
			put(1) // plain RS encoding
		} else {
			put(4) // interleaved RS encoding
		}
		put(uint64(len(pages)))
		for _, c := range name {
			sm = binary.LittleEndian.AppendUint16(sm, c)
		}
		sm = append(sm, 0, 0)
		for _, p := range pages {
			put(uint64(p.start))
			put(uint64(p.size))
			put(uint64(p.id))
			put(uint64(p.usize))
			put(uint64(p.csize))
			put(uint64(p.checksum))
			put(p.crc)
		}
	}
	// the empty section that closes the map
	put(0)
	put(0xF800)
	put(0)
	put(0)
	put(0)
	put(0)
	put(4)
	put(0)
	sections++

	smID, sm2ID := id+1, id+2
	pmID, pm2ID := id+3, id+4
	smPage, smFactor, smStored := encodeSystemPage(sm, 0)

	// page map: [page map][data pages][section map][section map copy][page map copy]
	type entry struct{ size, id int64 }
	var entries []entry
	pmSize := int64(r21PageMapPageSize(nEntries))
	entries = append(entries, entry{pmSize, pmID})
	if gap > 0 {
		entries = append(entries, entry{int64(gap), -(pm2ID + 1)})
	}
	for _, p := range dataPages {
		entries = append(entries, entry{int64(len(p.bytes)), p.id})
	}
	entries = append(entries, entry{int64(len(smPage)), smID}, entry{int64(len(smPage)), sm2ID})
	entries = append(entries, entry{pmSize, pm2ID})
	var pm []byte
	for _, e := range entries {
		pm = binary.LittleEndian.AppendUint64(pm, uint64(e.size))
		pm = binary.LittleEndian.AppendUint64(pm, uint64(e.id))
	}
	pmPage, pmFactor, pmStored := encodeSystemPage(pm, int(pmSize))

	// assemble the stream after the 0x480 byte prefix
	body := append([]byte(nil), pmPage...)
	body = append(body, make([]byte, gap)...)
	for _, p := range dataPages {
		body = append(body, p.bytes...)
	}
	smOffset := int64(len(body))
	body = append(body, smPage...)
	body = append(body, smPage...)
	pm2Offset := int64(len(body))
	body = append(body, pmPage...)
	header2Offset := int64(len(body))

	// file header (§5.2)
	var F [0x110 / 8]uint64
	check, first := r21DefaultCheck, r21DefaultCheckSeq
	if f.R21 != nil {
		F = f.R21.Fields
		if len(f.R21Check) == 0x28 {
			check = f.R21Check
		}
		if f.R21CheckSeq != 0 {
			first = f.R21CheckSeq
		}
	} else {
		F[0x70/8], F[0x78/8], F[0x88/8], F[0x90/8], F[0x98/8] = 0x20, 0x40, 0xF800, 4, 1
		F[r21StreamVersion] = 0x60100
		// The encoded seeds and check data depend only on the random seed, not on the content:
		// a consistent set taken from an AutoCAD 2007 file.
		F[r21PagesMapCrcSeed] = 0xa54735f19d696204
		F[r21SectionsMapCrcSeed] = 0x224b3db15918d418
		F[r21CrcSeed] = 0
		F[r21CrcSeedEncoded] = 0x378b39e75100c248
		F[r21RandomSeed] = r21DefaultRandomSeed
	}
	F[0] = 0x70
	F[1] = uint64(0x480 + len(body) + 0x400)
	F[r21PagesMapCrcCompressed] = crc64Mirrored(updateSeed1(0, uint32(len(pmStored))), pmStored)
	F[r21PagesMapCorrection] = uint64(pmFactor)
	F[r21PagesMap2Offset] = uint64(pm2Offset)
	F[r21PagesMap2Id] = uint64(pm2ID)
	F[r21PagesMapOffset] = 0
	F[r21PagesMapId] = uint64(pmID)
	F[r21Header2Offset] = uint64(header2Offset)
	F[r21PagesMapSizeCompressed] = uint64(len(pmStored))
	F[r21PagesMapSizeUncomp] = uint64(len(pm))
	F[r21PagesAmount] = uint64(len(entries))
	F[r21PagesMaxId] = uint64(pm2ID)
	if gap > 0 {
		F[r21PagesMaxId] = uint64(pm2ID + 1)
	}
	F[r21PagesMapCrcUncompressed] = crc64Mirrored(updateSeed1(0, uint32(len(pm))), pm)
	F[r21SectionsAmount] = uint64(sections)
	F[r21SectionsMapCrcUncomp] = crc64Mirrored(updateSeed1(0, uint32(len(sm))), sm)
	F[r21SectionsMapSizeComp] = uint64(len(smStored))
	F[r21SectionsMap2Id] = uint64(sm2ID)
	F[r21SectionsMapId] = uint64(smID)
	F[r21SectionsMapSizeUncomp] = uint64(len(sm))
	F[r21SectionsMapCrcComp] = crc64Mirrored(updateSeed1(0, uint32(len(smStored))), smStored)
	F[r21SectionsMapCorrection] = uint64(smFactor)
	_ = smOffset
	hp := buildR21HeaderPage(F, check, first)

	out := make([]byte, 0x80, 0x480+len(body)+0x400)
	copy(out, f.Prefix[:])
	setAddr := func(off int, name string) {
		v := uint32(0)
		if p, ok := firstPagePos[name]; ok {
			v = uint32(p + int64(len(pmPage)+gap) + r21Base)
		}
		binary.LittleEndian.PutUint32(out[off:], v)
	}
	setAddr(0x0D, "AcDb:Preview")
	setAddr(0x20, "AcDb:SummaryInfo")
	setAddr(0x24, "AcDb:VBAProject")
	setAddr(0x2C, "AcDb:AppInfo")
	setAddr(0x30, "AcDb:AppInfoHistory")
	out = append(out, hp...)
	out = append(out, body...)
	return append(out, hp...)
}

// rotl21 is the check data Encode function of §5.2.1.1.5: rotate left by control & 0x1f.
func rotl21(v, control uint64) uint64 {
	if n := control & 0x1F; n != 0 {
		v = v<<n | v>>(64-n)
	}
	return v
}

// Header page check data and check sequence value from an AutoCAD 2007 file (they depend only on
// the random seed written with them, see above).
var (
	r21DefaultCheck = []byte{0xd1, 0x81, 0xee, 0xf6, 0xaa, 0xfa, 0xbe, 0xe3, 0xba, 0xdc, 0x84, 0xc2, 0x8f, 0x1e,
		0xe1, 0xb0, 0x5a, 0xbc, 0xf4, 0x3c, 0x1c, 0x38, 0xb9, 0xd8, 0x0b, 0x15, 0x4f, 0x65, 0xa2, 0x7a, 0xd0, 0x23,
		0x07, 0x05, 0x08, 0x08, 0x14, 0x28, 0x9b, 0x15}
	r21DefaultCheckSeq uint64 = 0x1944b8f351bc0406
	// r21DefaultRandomSeed is the RandomSeed (header 0x100) written with the values above.
	r21DefaultRandomSeed uint64 = 0x6f341ff1e4380a31
)

// buildR21HeaderPage encodes the 0x400 byte file header page (§5.2.1) for the header fields F.
func buildR21HeaderPage(F [0x110 / 8]uint64, check []byte, first uint64) []byte {
	hdr := make([]byte, 0x110)
	for i, v := range F {
		binary.LittleEndian.PutUint64(hdr[8*i:], v)
	}
	binary.LittleEndian.PutUint64(hdr[0x108:], 0)
	binary.LittleEndian.PutUint64(hdr[0x108:], crc64Normal(updateSeed2(0, 0x110), hdr))

	// [check seq crc][check seq value][data crc][len int64][compressed data, padded to 8], repeated,
	// RS(255,239) x3 interleaved. (Readers reject the uncompressed, negative length form.)
	comp := Compress21(hdr)
	var seq []byte
	seq = binary.LittleEndian.AppendUint64(seq, first)
	seq = binary.LittleEndian.AppendUint64(seq, rotl21(first, first))
	var blk []byte
	blk = binary.LittleEndian.AppendUint64(blk, crc64Normal(updateSeed1(0, 16), seq))
	blk = binary.LittleEndian.AppendUint64(blk, first)
	blk = binary.LittleEndian.AppendUint64(blk, crc64Normal(updateSeed2(0, uint32(len(comp))), comp))
	blk = binary.LittleEndian.AppendUint64(blk, uint64(len(comp)))
	blk = append(blk, comp...)
	blk = append(blk, make([]byte, align(len(blk), 8)-len(blk))...)
	pre := make([]byte, 3*239)
	for p := 0; p+len(blk) <= len(pre); p += len(blk) {
		copy(pre[p:], blk)
	}
	hp := make([]byte, 0x400)
	copy(hp, rsEncodeInterleaved(pre, rsSystem))
	copy(hp[0x3D8:], check)
	return hp
}

// rebasePreview shifts the absolute entry addresses of an AcDb:Preview section (§14: sentinel,
// RL size, RC count, count x (RC code, RL start, RL size)) so they point into a section that now
// starts at file address base. The section header is 16+4+1+9*count bytes long.
func rebasePreview(data []byte, base uint32) []byte {
	if len(data) < 21 {
		return data
	}
	n := int(data[20])
	hl := 21 + 9*n
	if len(data) < hl || n == 0 {
		return data
	}
	minStart := ^uint32(0)
	for i := 0; i < n; i++ {
		if st := binary.LittleEndian.Uint32(data[21+9*i+1:]); st < minStart {
			minStart = st
		}
	}
	oldBase := minStart - uint32(hl)
	out := append([]byte(nil), data...)
	for i := 0; i < n; i++ {
		p := 21 + 9*i + 1
		binary.LittleEndian.PutUint32(out[p:], binary.LittleEndian.Uint32(out[p:])-oldBase+base)
	}
	return out
}

// r21PageMapPageSize is the size AutoCAD gives the page map page: room for (entries + 5) * 16
// bytes (§5.2 "((dataSectionPageCount + 5) * 16)"), not the plain system page rule.
func r21PageMapPageSize(entries int) int { return systemPageSize((entries + 5) * 16) }
