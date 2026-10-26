package dwg

import (
	"encoding/binary"
	"fmt"
	"sort"

	"dwgconv/internal/bits"
	"dwgconv/internal/container"
)

// Document is a decoded drawing.
type Document struct {
	File     *container.File
	Version  Version
	Codepage int
	Text     TextCodec
	Classes  *ClassTable
	Objects  []*Object // in file (offset) order
	ByHandle map[uint64]*Object
	// objPrefix holds the bytes before the first object (RL 0x0dca for R2004+).
	objPrefix []byte
}

// newBody returns an empty body for a fixed type, or nil if the type is not decoded.
var bodyFactory = map[int]func() Body{}

// classBodyFactory decodes class-based types by DXF name.
var classBodyFactory = map[string]func() Body{}

// entity types (fixed) — used to decide whether common entity data is present.
func isEntityType(t int, classes []Class) bool {
	if t >= 500 {
		i := t - 500
		return i < len(classes) && classes[i].ItemClass == 0x1F2
	}
	switch {
	case t >= 1 && t <= 41:
		return t != 9 // 9 is unused
	case t == 0x2B || t == 44 || t == 45 || t == 46 || t == 47 || t == 0x4A || t == 0x4D || t == 0x4E || t == 0x1F2:
		return true
	}
	return false
}

func Load(path string) (*Document, error) {
	f, err := container.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Decode(f)
}

func Decode(f *container.File) (*Document, error) {
	d := &Document{File: f, Version: versionOf(f.Version)}
	d.Codepage = int(binary.LittleEndian.Uint16(f.Prefix[0x13:]))
	d.Text = textCodecFor(d.Codepage)
	cs := f.Section("AcDb:Classes")
	if cs == nil {
		return nil, fmt.Errorf("no AcDb:Classes section")
	}
	ct, err := decodeClasses(cs.Data, d.Version, d.Text)
	if err != nil {
		return nil, err
	}
	d.Classes = ct
	hs, os_ := f.Section("AcDb:Handles"), f.Section("AcDb:AcDbObjects")
	if hs == nil || os_ == nil {
		return nil, fmt.Errorf("missing objects or handles section")
	}
	omap, err := decodeObjectMap(hs.Data)
	if err != nil {
		return nil, err
	}
	type loc struct {
		h   uint64
		off int
	}
	locs := make([]loc, 0, len(omap))
	for h, off := range omap {
		locs = append(locs, loc{h, int(off)})
	}
	sort.Slice(locs, func(i, j int) bool { return locs[i].off < locs[j].off })
	if len(locs) > 0 {
		d.objPrefix = append([]byte(nil), os_.Data[:locs[0].off]...)
	}
	d.ByHandle = make(map[uint64]*Object, len(locs))
	for _, l := range locs {
		o := d.decodeObject(os_.Data, l.off)
		if o.Handle == 0 {
			o.Handle = l.h
		}
		d.Objects = append(d.Objects, o)
		d.ByHandle[o.Handle] = o
	}
	return d, nil
}

func (d *Document) decodeObject(buf []byte, off int) *Object {
	o := &Object{}
	c, typ, end, err := splitObject(buf, off, d.Version)
	if err != nil {
		o.DecodeErr = err
		return o
	}
	c.text = d.Text
	o.Raw = buf[off : end+2]
	o.Type = typ
	o.IsEntity = isEntityType(typ, d.Classes.Classes)
	if o.IsEntity {
		o.Ent = &EntityCommon{}
	}
	var mk func() Body
	if typ >= 500 {
		if i := typ - 500; i < len(d.Classes.Classes) {
			mk = classBodyFactory[d.Classes.Classes[i].DXFName]
		}
	} else {
		mk = bodyFactory[typ]
	}
	err = catch(func() {
		o.decodeCommon(c)
		if mk == nil {
			mk = func() Body { return &RawBody{} }
		}
		b := mk()
		b.data(c, o)
		o.codeCommonHandles(c)
		b.handles(c, o)
		o.Body = b
		o.LeftData, o.LeftHandles = c.dat.Remaining(), c.hdl.Remaining()
		if o.LeftHandles > 0 && o.LeftHandles < 8 {
			o.padBits = o.LeftHandles
			o.pad = c.hdl.Bits(o.LeftHandles)
		}
	})
	if err != nil {
		o.Body = nil
		o.DecodeErr = err
	}
	o.StrFlag = d.Version.R2007 && c.str != c.dat
	return o
}

// decodeObjectMap parses AcDb:Handles (§23). Offsets reset at each chunk (observed; the spec text implies otherwise).
func decodeObjectMap(data []byte) (map[uint64]int64, error) {
	out := map[uint64]int64{}
	p := 0
	for p+2 <= len(data) {
		size := int(binary.BigEndian.Uint16(data[p:]))
		if size == 2 {
			return out, nil
		}
		end := p + size
		if end+2 > len(data) {
			return nil, fmt.Errorf("handles: chunk out of range")
		}
		q := p + 2
		var lastH, lastLoc int64
		for q < end {
			var dh, dl int64
			dh, q = bits.MC(data, q, false)
			dl, q = bits.MC(data, q, true)
			lastH += dh
			lastLoc += dl
			out[uint64(lastH)] = lastLoc
		}
		p = end + 2
	}
	return out, nil
}

func encodeObjectMap(m map[uint64]int64) []byte {
	hs := make([]uint64, 0, len(m))
	for h := range m {
		hs = append(hs, h)
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i] < hs[j] })
	var out []byte
	for i := 0; i < len(hs); {
		chunk := []byte{0, 0}
		var lastH uint64
		var lastLoc int64
		for i < len(hs) {
			pair := bits.PutMCUnsigned(nil, hs[i]-lastH)
			pair = bits.PutMCSigned(pair, m[hs[i]]-lastLoc)
			if len(chunk)+len(pair) > 2032 {
				break
			}
			chunk = append(chunk, pair...)
			lastH, lastLoc = hs[i], m[hs[i]]
			i++
		}
		binary.BigEndian.PutUint16(chunk, uint16(len(chunk)))
		out = append(out, chunk...)
		out = binary.BigEndian.AppendUint16(out, bits.CRC8(0xC0C1, chunk))
	}
	end := []byte{0, 2}
	out = append(out, end...)
	return binary.BigEndian.AppendUint16(out, bits.CRC8(0xC0C1, end))
}

// EncodeObjects re-serializes all objects and returns the AcDb:AcDbObjects and AcDb:Handles section data.
func (d *Document) EncodeObjects() (objects, handles []byte) {
	objects = append([]byte(nil), d.objPrefix...)
	m := make(map[uint64]int64, len(d.Objects))
	for _, o := range d.Objects {
		m[o.Handle] = int64(len(objects))
		objects = append(objects, o.Encode(d.Version, d.Text)...)
	}
	return objects, encodeObjectMap(m)
}

// NewDocument returns an empty document for building a drawing from scratch.
func NewDocument(version string, codepage int) *Document {
	return &Document{
		Version:   versionOf(version),
		Codepage:  codepage,
		Text:      textCodecFor(codepage),
		Classes:   DefaultClasses(),
		ByHandle:  map[uint64]*Object{},
		objPrefix: []byte{0xCA, 0x0D, 0, 0}, // RL 0x0dca at the start of AcDb:AcDbObjects (§20)
	}
}

// Add appends an object to the document.
func (d *Document) Add(o *Object) {
	d.Objects = append(d.Objects, o)
	d.ByHandle[o.Handle] = o
}
