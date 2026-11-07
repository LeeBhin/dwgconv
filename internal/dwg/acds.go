package dwg

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
)

// AcDs data store (§24, section "AcDb:AcDsPrototype_1b"). From R2013 the SAB stream of REGION,
// 3DSOLID and BODY lives here under the entity handle (schema "AcDb3DSolid_ASM_Data") instead of
// inside the object. The layout written here follows an ODA-written file: file header, data
// segment with the records, an empty data segment, data index, schema data, schema index, search
// data and the segment index. The two schema segments do not depend on the drawing and are
// embedded verbatim (they include their own segment headers, indexes 5 and 6).

//go:embed acds_schdat.bin
var acdsSchdat []byte

//go:embed acds_schidx.bin
var acdsSchidx []byte

const (
	acdsSignature = 0x6472616A // "jard"
	acdsSegSig    = 0xD5AC
	acdsSegHdr    = 0x30 // 2 signature + 6 name + 8 Int32 + 8 alignment bytes (0x55)
	acdsBlobPages = 0xBB106BB1
)

// AcDsRecord is one data record: the SAB bytes of an entity.
type AcDsRecord struct {
	Handle uint64
	Data   []byte
}

// ParseAcDs returns the data records by entity handle. Records paged into blob01 segments
// (over 0x40000 bytes) are left out.
func ParseAcDs(b []byte) (map[uint64][]byte, error) {
	le := binary.LittleEndian
	if len(b) < 0x80 || le.Uint32(b) != acdsSignature {
		return nil, fmt.Errorf("acds: bad file header")
	}
	segidxOff, nseg, datidxSeg := int(le.Uint32(b[24:])), int(le.Uint32(b[32:])), int(le.Uint32(b[40:]))
	type seg struct{ off, size int }
	if segidxOff+acdsSegHdr+12*nseg > len(b) {
		return nil, fmt.Errorf("acds: segment index out of range")
	}
	segs := make([]seg, nseg)
	for i := range segs {
		p := segidxOff + acdsSegHdr + 12*i
		segs[i] = seg{int(le.Uint64(b[p:])), int(le.Uint32(b[p+8:]))}
	}
	if datidxSeg <= 0 || datidxSeg >= nseg {
		return nil, fmt.Errorf("acds: no data index segment")
	}
	di := segs[datidxSeg].off + acdsSegHdr
	if di+8 > len(b) {
		return nil, fmt.Errorf("acds: data index out of range")
	}
	n := int(le.Uint32(b[di:]))
	out := map[uint64][]byte{}
	for k := 0; k < n; k++ {
		e := di + 8 + 12*k
		if e+12 > len(b) {
			return out, fmt.Errorf("acds: data index entry %d out of range", k)
		}
		segIdx, local := int(le.Uint32(b[e:])), int(le.Uint32(b[e+4:]))
		if segIdx <= 0 || segIdx >= nseg {
			continue // stub entry
		}
		s := segs[segIdx]
		hdr := s.off + acdsSegHdr + local // record header: size, 1, handle, blob offset
		if hdr+20 > len(b) {
			return out, fmt.Errorf("acds: record header out of range")
		}
		handle := le.Uint64(b[hdr+8:])
		dataStart := s.off + int(le.Uint32(b[s.off+36:]))<<4 // object data alignment offset
		rs := dataStart + int(le.Uint32(b[hdr+16:]))
		if rs+4 > len(b) {
			return out, fmt.Errorf("acds: record out of range")
		}
		size := int(le.Uint32(b[rs:]))
		if uint32(size) == acdsBlobPages || rs+4+size > len(b) {
			continue
		}
		rec := b[rs+4 : rs+4+size]
		// only the SAB records (schema AcDb3DSolid_ASM_Data); AutoCAD stores thumbnails and other
		// data here too, so the record is recognized by its header rather than by schema index
		if !bytes.HasPrefix(rec, []byte("ASM BinaryFile")) && !bytes.HasPrefix(rec, []byte("ACIS BinaryFile")) {
			continue
		}
		out[handle] = append([]byte(nil), rec...)
	}
	return out, nil
}

// acdsSegHeader is the 0x30-byte segment header (§24.2.1).
func acdsSegHeader(name string, idx, size, sysAlign, objAlign uint32) []byte {
	var h [acdsSegHdr]byte
	le := binary.LittleEndian
	le.PutUint16(h[0:], acdsSegSig)
	copy(h[2:8], name)
	for i, v := range []uint32{idx, 0, size, 0, 1, 0, sysAlign, objAlign} {
		le.PutUint32(h[8+4*i:], v)
	}
	for i := 0x28; i < acdsSegHdr; i++ {
		h[i] = 0x55
	}
	return h[:]
}

// acdsPad pads a segment to a multiple of 0x40 bytes with 0x70, as AutoCAD and ODA write it.
func acdsPad(seg []byte) []byte {
	for len(seg)%0x40 != 0 {
		seg = append(seg, 0x70)
	}
	return seg
}

// BuildAcDs writes the section for the given records (ASM_Data schema, index 1).
func BuildAcDs(recs []AcDsRecord) []byte {
	le := binary.LittleEndian
	u32 := func(w *bytes.Buffer, v uint32) { _ = binary.Write(w, le, v) }
	u64 := func(w *bytes.Buffer, v uint64) { _ = binary.Write(w, le, v) }

	// data segment (2): record headers, filler to a 16-byte boundary, then the records
	var entries, blobs bytes.Buffer
	for i, r := range recs {
		_ = i
		u32(&entries, 0x14)
		u32(&entries, 1)
		u64(&entries, r.Handle)
		u32(&entries, uint32(blobs.Len()))
		u32(&blobs, uint32(len(r.Data)))
		blobs.Write(r.Data)
	}
	dataStart := (acdsSegHdr + entries.Len() + 15) &^ 15
	var data bytes.Buffer
	data.Write(acdsSegHeader("_data_", 2, 0, 0, uint32(dataStart>>4)))
	data.Write(entries.Bytes())
	for data.Len() < dataStart {
		data.WriteByte(0x62)
	}
	data.Write(blobs.Bytes())
	dataSeg := acdsPad(data.Bytes())
	le.PutUint32(dataSeg[16:], uint32(len(dataSeg)))

	// empty data segment (3), as ODA writes it
	empty := append(acdsSegHeader("_data_", 3, 0x40, 0, 4), bytes.Repeat([]byte{0x62}, 16)...)

	// data index (4): one entry per record pointing at its header in segment 2
	var di bytes.Buffer
	di.Write(acdsSegHeader("datidx", 4, 0, 0, 0))
	u32(&di, uint32(len(recs)))
	u32(&di, 0)
	for i := range recs {
		u32(&di, 2)
		u32(&di, uint32(0x14*i))
		u32(&di, 1)
	}
	datidx := acdsPad(di.Bytes())
	le.PutUint32(datidx[16:], uint32(len(datidx)))

	// search (7): schema 1 lists every record by sorted index and handle; schema 0 is empty
	var se bytes.Buffer
	se.Write(acdsSegHeader("search", 7, 0, 0, 0))
	u32(&se, 2)
	u32(&se, 1)
	u64(&se, uint64(len(recs)))
	for i := range recs {
		u64(&se, uint64(i))
	}
	u32(&se, 1)
	u32(&se, 0)
	u32(&se, uint32(len(recs)))
	for i, r := range recs {
		u64(&se, r.Handle)
		u64(&se, 1)
		u64(&se, uint64(i))
	}
	u32(&se, 0)
	u64(&se, 0)
	u32(&se, 1)
	u32(&se, 0)
	u32(&se, 0)
	search := acdsPad(se.Bytes())
	le.PutUint32(search[16:], uint32(len(search)))

	// segment index (1): 8 entries, index 0 unused
	segidxSize := uint32((acdsSegHdr + 12*8 + 0x3F) &^ 0x3F)
	offs := make([]uint64, 8)
	sizes := make([]uint32, 8)
	pos := uint64(0x80)
	place := func(idx int, size uint32) {
		offs[idx], sizes[idx] = pos, size
		pos += uint64(size)
	}
	place(2, uint32(len(dataSeg)))
	place(3, uint32(len(empty)))
	place(4, uint32(len(datidx)))
	place(5, uint32(len(acdsSchdat)))
	place(6, uint32(len(acdsSchidx)))
	place(7, uint32(len(search)))
	place(1, segidxSize)
	var si bytes.Buffer
	si.Write(acdsSegHeader("segidx", 1, segidxSize, 0, 0))
	for i := 0; i < 8; i++ {
		u64(&si, offs[i])
		u32(&si, sizes[i])
	}
	segidx := acdsPad(si.Bytes())

	// file header (§24.1)
	var out bytes.Buffer
	for _, v := range []uint32{acdsSignature, 0x80, 2, 2, 0, 1, uint32(offs[1]), 0, 8, 6, 4, 7, 0, uint32(pos)} {
		u32(&out, v)
	}
	out.Write(make([]byte, 0x80-out.Len()))
	out.Write(dataSeg)
	out.Write(empty)
	out.Write(datidx)
	out.Write(acdsSchdat)
	out.Write(acdsSchidx)
	out.Write(search)
	out.Write(segidx)
	return out.Bytes()
}
