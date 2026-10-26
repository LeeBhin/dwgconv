package dwg

import (
	"bytes"
	"fmt"
)

// ---- AcDb:SummaryInfo (§13) ----

// SummaryInfo holds the drawing properties. Strings are UTF-16 for R2007+, codepage bytes before.
type SummaryInfo struct {
	Title, Subject, Author, Keywords, Comments, LastSavedBy, RevisionNumber, HyperlinkBase string
	EditTime                                                                               JulianDate // total editing time: days + milliseconds
	Created, Modified                                                                      JulianDate
	Props                                                                                  [][2]string
	Unknown1, Unknown2                                                                     int32
}

func (s *SummaryInfo) code(c *bcodec, wide bool) {
	for _, p := range []*string{&s.Title, &s.Subject, &s.Author, &s.Keywords, &s.Comments,
		&s.LastSavedBy, &s.RevisionNumber, &s.HyperlinkBase} {
		c.Str16(p, wide)
	}
	c.Julian(&s.EditTime)
	c.Julian(&s.Created)
	c.Julian(&s.Modified)
	n := c.Count(len(s.Props), 2)
	if !c.w {
		s.Props = make([][2]string, n)
	}
	for i := range s.Props {
		c.Str16(&s.Props[i][0], wide)
		c.Str16(&s.Props[i][1], wide)
	}
	c.I32(&s.Unknown1)
	c.I32(&s.Unknown2)
}

func DecodeSummaryInfo(data []byte, v Version, text TextCodec) (*SummaryInfo, error) {
	s := &SummaryInfo{}
	return s, decodeBytes(data, text, func(c *bcodec) { s.code(c, v.R2007) })
}

func (s *SummaryInfo) Encode(v Version, text TextCodec) []byte {
	return encodeBytes(text, func(c *bcodec) { s.code(c, v.R2007) })
}

// ---- AcDb:AppInfo (§16.2; R21+ layout, also used by R2004 files saved by newer AutoCAD) ----

// AppInfo is the writer application record: a name followed by checksum/text pairs
// (version, comment, product XML ...).
type AppInfo struct {
	Unknown int32 // 3 in AutoCAD files (ODA writes 2)
	Name    string
	Records []AppInfoRecord
}

// AppInfoRecord is a 16-byte checksum followed by a UTF-16 string.
type AppInfoRecord struct {
	Checksum []byte
	Text     string
}

func (a *AppInfo) code(c *bcodec) {
	c.I32(&a.Unknown)
	c.Str16(&a.Name, true)
	n := c.Count(len(a.Records), 4)
	if !c.w {
		a.Records = make([]AppInfoRecord, n)
	}
	for i := range a.Records {
		c.Bytes(&a.Records[i].Checksum, 16)
		c.Str16(&a.Records[i].Text, true)
	}
}

func DecodeAppInfo(data []byte, text TextCodec) (*AppInfo, error) {
	a := &AppInfo{}
	return a, decodeBytes(data, text, a.code)
}

func (a *AppInfo) Encode(text TextCodec) []byte { return encodeBytes(text, a.code) }

// ---- AcDb:RevHistory (§18) ----

// RevHistory is a list of unknown 32-bit values (AutoCAD writes 0, 0, 1, 0).
type RevHistory struct{ Values []uint32 }

func DecodeRevHistory(data []byte) (*RevHistory, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("revhistory: size %d", len(data))
	}
	r := &RevHistory{Values: make([]uint32, len(data)/4)}
	return r, decodeBytes(data, TextCodec{}, func(c *bcodec) {
		for i := range r.Values {
			c.U32(&r.Values[i])
		}
	})
}

func (r *RevHistory) Encode() []byte {
	return encodeBytes(TextCodec{}, func(c *bcodec) {
		for i := range r.Values {
			c.U32(&r.Values[i])
		}
	})
}

// ---- AcDb:Template (§22) ----

// Template holds the template description and MEASUREMENT (0 English, 1 metric).
// The description length counts bytes (R2004) or UTF-16 characters including the NUL (R2007+).
type Template struct {
	Description []byte // raw encoded description, as stored
	Measurement int16
}

func (t *Template) code(c *bcodec, v Version) {
	unit := 1
	if v.R2007 {
		unit = 2
	}
	n := c.Count(len(t.Description)/unit, 2)
	c.Bytes(&t.Description, n*unit)
	c.I16(&t.Measurement)
}

func DecodeTemplate(data []byte, v Version) (*Template, error) {
	t := &Template{}
	return t, decodeBytes(data, TextCodec{}, func(c *bcodec) { t.code(c, v) })
}

func (t *Template) Encode(v Version) []byte {
	return encodeBytes(TextCodec{}, func(c *bcodec) { t.code(c, v) })
}

// DefaultTemplate is what AutoCAD writes: an empty description (one NUL character for R2007+).
func DefaultTemplate(v Version, measurement int16) *Template {
	t := &Template{Measurement: measurement}
	if v.R2007 {
		t.Description = []byte{0, 0}
	}
	return t
}

// ---- AcDb:ObjFreeSpace (§21) ----

// ObjFreeSpace: R2004 has 32-bit fields and an objects-section offset (53 bytes); R2010+ widens
// the counters and the value list to 64 bits and drops the offset (89 bytes). Not in the spec.
type ObjFreeSpace struct {
	Unknown0   uint64
	NumHandles uint64 // approximate number of objects
	TDUPDATE   JulianDate
	Offset     uint32 // R2004 only
	Values     []uint64
}

func (o *ObjFreeSpace) code(c *bcodec, v Version) {
	if v.R2010 {
		c.U64(&o.Unknown0)
		c.U64(&o.NumHandles)
	} else {
		u0, nh := uint32(o.Unknown0), uint32(o.NumHandles)
		c.U32(&u0)
		c.U32(&nh)
		o.Unknown0, o.NumHandles = uint64(u0), uint64(nh)
	}
	c.Julian(&o.TDUPDATE)
	if !v.R2010 {
		c.U32(&o.Offset)
	}
	n := byte(len(o.Values) / 2)
	c.U8(&n)
	if !c.w {
		o.Values = make([]uint64, 2*int(n))
	}
	for i := range o.Values {
		if v.R2010 {
			c.U64(&o.Values[i])
		} else {
			x := uint32(o.Values[i])
			c.U32(&x)
			o.Values[i] = uint64(x)
		}
	}
}

func DecodeObjFreeSpace(data []byte, v Version) (*ObjFreeSpace, error) {
	o := &ObjFreeSpace{}
	return o, decodeBytes(data, TextCodec{}, func(c *bcodec) { o.code(c, v) })
}

func (o *ObjFreeSpace) Encode(v Version) []byte {
	return encodeBytes(TextCodec{}, func(c *bcodec) { o.code(c, v) })
}

// DefaultObjFreeSpace returns the values AutoCAD and the ODA write (§21).
func DefaultObjFreeSpace(numObjects uint64, tdupdate JulianDate) *ObjFreeSpace {
	return &ObjFreeSpace{NumHandles: numObjects, TDUPDATE: tdupdate,
		Values: []uint64{0x32, 0, 0x64, 0, 0x200, 0, 0xFFFFFFFF, 0}}
}

// ---- AcDb:AuxHeader (§27) ----

// AuxHeader is the auxiliary file header. AC1032 stores the three maintenance versions as RL
// (129 bytes); earlier versions as RS (123 bytes). The trailing block (dates, HANDSEED, save
// counts) that §27 marks "R2018+" is present in every R2004+ file.
type AuxHeader struct {
	Magic                  [3]byte // ff 77 01
	Version                int16
	Maint                  int32
	Saves                  int32
	MinusOne               int32
	SavesPart1, SavesPart2 int16
	Zero1                  int32
	Version2               int16
	Maint2                 int32
	Version3               int16
	Maint3                 int32
	Unknown                [6]int16 // spec: 0x0005, 0x0893, 0x0005, 0x0893, 0, 1
	Zero2                  [5]int32
	TDCREATE, TDUPDATE     JulianDate
	HandSeed               int32
	EduPlotStamp           int32
	Zero3                  int16
	SavesDiff              int16
	Zero4                  [3]int32
	Saves2                 int32
	Zero5                  [4]int32
}

func (a *AuxHeader) code(c *bcodec, v Version) {
	maint := func(p *int32) {
		if v.R2018 {
			c.I32(p)
			return
		}
		x := int16(*p)
		c.I16(&x)
		*p = int32(x)
	}
	for i := range a.Magic {
		c.U8(&a.Magic[i])
	}
	c.I16(&a.Version)
	maint(&a.Maint)
	c.I32(&a.Saves)
	c.I32(&a.MinusOne)
	c.I16(&a.SavesPart1)
	c.I16(&a.SavesPart2)
	c.I32(&a.Zero1)
	c.I16(&a.Version2)
	maint(&a.Maint2)
	c.I16(&a.Version3)
	maint(&a.Maint3)
	for i := range a.Unknown {
		c.I16(&a.Unknown[i])
	}
	for i := range a.Zero2 {
		c.I32(&a.Zero2[i])
	}
	c.Julian(&a.TDCREATE)
	c.Julian(&a.TDUPDATE)
	c.I32(&a.HandSeed)
	c.I32(&a.EduPlotStamp)
	c.I16(&a.Zero3)
	c.I16(&a.SavesDiff)
	for i := range a.Zero4 {
		c.I32(&a.Zero4[i])
	}
	c.I32(&a.Saves2)
	for i := range a.Zero5 {
		c.I32(&a.Zero5[i])
	}
}

func DecodeAuxHeader(data []byte, v Version) (*AuxHeader, error) {
	a := &AuxHeader{}
	return a, decodeBytes(data, TextCodec{}, func(c *bcodec) { a.code(c, v) })
}

func (a *AuxHeader) Encode(v Version) []byte {
	return encodeBytes(TextCodec{}, func(c *bcodec) { a.code(c, v) })
}

// auxVersionCodes is the §27 DWG version number per version string.
var auxVersionCodes = map[string]int16{"AC1018": 25, "AC1021": 27, "AC1024": 29, "AC1027": 31, "AC1032": 33}

// DefaultAuxHeader builds the auxiliary header for a newly written file (one save).
func DefaultAuxHeader(version string, maint int32, created, updated JulianDate, handSeed uint64) *AuxHeader {
	vc := auxVersionCodes[version]
	hs := int32(-1)
	if handSeed < 0x7FFFFFFF {
		hs = int32(handSeed)
	}
	return &AuxHeader{
		Magic: [3]byte{0xFF, 0x77, 0x01}, Version: vc, Maint: maint, Saves: 1, MinusOne: -1,
		SavesPart1: 1, Version2: vc, Maint2: maint, Version3: vc, Maint3: maint,
		Unknown:  [6]int16{0x0005, 0x0893, 0x0005, 0x0893, 0, 1},
		TDCREATE: created, TDUPDATE: updated, HandSeed: hs, SavesDiff: 1, Saves2: 1,
	}
}

// ---- AcDb:FileDepList (§17) ----

// FileDepList lists external files (fonts, plot configs, xrefs, images). Strings have an Int32
// byte length and no terminator: UTF-16LE for R2007+ (the spec only shows the 8-bit form).
type FileDepList struct {
	Features []string
	Files    []FileDep
	Tail     []byte // page padding after the last entry, kept verbatim
}

type FileDep struct {
	FullName, FoundPath, Fingerprint, VersionGUID string
	Feature, Timestamp, Size                      int32
	AffectsGraphics                               int16
	RefCount                                      int32
}

func (f *FileDepList) code(c *bcodec, wide bool) {
	n := c.Count(len(f.Features), 4)
	if !c.w {
		f.Features = make([]string, n)
	}
	for i := range f.Features {
		c.Str32(&f.Features[i], wide)
	}
	n = c.Count(len(f.Files), 4)
	if !c.w {
		f.Files = make([]FileDep, n)
	}
	for i := range f.Files {
		d := &f.Files[i]
		c.Str32(&d.FullName, wide)
		c.Str32(&d.FoundPath, wide)
		c.Str32(&d.Fingerprint, wide)
		c.Str32(&d.VersionGUID, wide)
		c.I32(&d.Feature)
		c.I32(&d.Timestamp)
		c.I32(&d.Size)
		c.I16(&d.AffectsGraphics)
		c.I32(&d.RefCount)
	}
	c.Rest(&f.Tail)
}

func DecodeFileDepList(data []byte, v Version, text TextCodec) (*FileDepList, error) {
	f := &FileDepList{}
	return f, decodeBytes(data, text, func(c *bcodec) { f.code(c, v.R2007) })
}

func (f *FileDepList) Encode(v Version, text TextCodec) []byte {
	return encodeBytes(text, func(c *bcodec) { f.code(c, v.R2007) })
}

// ---- AcDb:Preview (§14.2) ----

var (
	sentinelPreviewStart = []byte{0x1F, 0x25, 0x6D, 0x07, 0xD4, 0x36, 0x28, 0x28, 0x9D, 0x57, 0xCA, 0x3F, 0x9D, 0x44, 0x10, 0x2B}
	sentinelPreviewEnd   = []byte{0xE0, 0xDA, 0x92, 0xF8, 0x2B, 0xC9, 0xD7, 0xD7, 0x62, 0xA8, 0x35, 0xC0, 0x62, 0xBB, 0xEF, 0xD4}
)

// Preview is the thumbnail section. Entry codes: 1 header data, 2 BMP, 3 WMF, 6 PNG (6 is not in
// the spec). Entry starts are absolute file addresses: Base is the file address of the section's
// first data byte (file header offset 0x0D minus 0x20 + 0x20, i.e. the value stored at 0x0D).
type Preview struct {
	Base    uint32
	Entries []PreviewEntry
}

type PreviewEntry struct {
	Code byte
	Data []byte
}

func previewHeaderLen(n int) int { return 16 + 4 + 1 + 9*n }

func DecodePreview(data []byte) (*Preview, error) {
	if len(data) < 16+4+1+16 || !bytes.Equal(data[:16], sentinelPreviewStart) {
		return nil, fmt.Errorf("preview: bad start sentinel")
	}
	p := &Preview{}
	err := catch(func() {
		c := &bcodec{buf: data, p: 16}
		var size int32
		c.I32(&size)
		var n byte
		c.U8(&n)
		type ent struct {
			code        byte
			start, size int32
		}
		ents := make([]ent, n)
		minStart := int64(-1)
		for i := range ents {
			c.U8(&ents[i].code)
			c.I32(&ents[i].start)
			c.I32(&ents[i].size)
			if minStart < 0 || int64(ents[i].start) < minStart {
				minStart = int64(ents[i].start)
			}
		}
		hl := previewHeaderLen(int(n))
		if n > 0 {
			p.Base = uint32(minStart) - uint32(hl)
		}
		pos := hl
		for _, e := range ents {
			off := int(uint32(e.start) - p.Base)
			if off != pos {
				panic(fmt.Errorf("preview: entry data not contiguous (%d != %d)", off, pos))
			}
			p.Entries = append(p.Entries, PreviewEntry{Code: e.code, Data: append([]byte(nil), data[off:off+int(e.size)]...)})
			pos += int(e.size)
		}
		if int(size) != pos-20 || !bytes.Equal(data[pos:pos+16], sentinelPreviewEnd) || pos+16 != len(data) {
			panic(fmt.Errorf("preview: unexpected layout (size %d, data end %d, len %d)", size, pos, len(data)))
		}
	})
	return p, err
}

// Encode serializes the section with entry addresses relative to p.Base. A writer that moves the
// section must set Base to the new file address (the value it stores at file offset 0x0D) first.
func (p *Preview) Encode() []byte {
	hl := previewHeaderLen(len(p.Entries))
	total := hl
	for _, e := range p.Entries {
		total += len(e.Data)
	}
	c := &bcodec{w: true, buf: append([]byte(nil), sentinelPreviewStart...)}
	size := int32(total - 20)
	c.I32(&size)
	n := byte(len(p.Entries))
	c.U8(&n)
	pos := hl
	for i := range p.Entries {
		e := &p.Entries[i]
		c.U8(&e.Code)
		start, sz := int32(p.Base+uint32(pos)), int32(len(e.Data))
		c.I32(&start)
		c.I32(&sz)
		pos += len(e.Data)
	}
	for _, e := range p.Entries {
		c.buf = append(c.buf, e.Data...)
	}
	return append(c.buf, sentinelPreviewEnd...)
}

// DefaultPreview is a preview with no images.
func DefaultPreview() *Preview { return &Preview{} }

// DefaultAppInfo returns an AppInfo in the R21+ layout AutoCAD uses (zero checksums, as the ODA writes).
func DefaultAppInfo(version, comment, product string) *AppInfo {
	z := make([]byte, 16)
	return &AppInfo{Unknown: 3, Name: "AppInfoDataList", Records: []AppInfoRecord{
		{Checksum: z, Text: version}, {Checksum: z, Text: comment}, {Checksum: z, Text: product}}}
}

// DefaultSummaryInfo returns empty drawing properties with the given dates.
func DefaultSummaryInfo(created, modified JulianDate) *SummaryInfo {
	return &SummaryInfo{Created: created, Modified: modified}
}

// DefaultRevHistory is what AutoCAD writes (§18.2).
func DefaultRevHistory() *RevHistory { return &RevHistory{Values: []uint32{0, 0, 1, 0}} }
