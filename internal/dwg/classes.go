package dwg

import (
	"encoding/binary"
	"fmt"

	"github.com/LeeBhin/dwgconv/internal/bits"
)

var (
	sentinelClassesStart = []byte{0x8D, 0xA1, 0xC4, 0xB8, 0xC4, 0xA9, 0xF8, 0xC5, 0xC0, 0xDC, 0xF4, 0x5F, 0xE7, 0xCF, 0xB6, 0x8A}
	sentinelClassesEnd   = []byte{0x72, 0x5E, 0x3B, 0x47, 0x3B, 0x56, 0x07, 0x3A, 0x3F, 0x23, 0x0B, 0xA0, 0x18, 0x30, 0x49, 0x75}
)

// Class is one custom class definition (§10.2, §5.8).
type Class struct {
	Num        int16
	ProxyFlags int16
	AppName    string
	CppName    string
	DXFName    string
	Zombie     bool
	ItemClass  int16 // 0x1F2 entity, 0x1F3 object
	Count      int32
	DwgVer     int16
	MaintVer   int16
	Unk1, Unk2 int32
}

// ClassTable is the AcDb:Classes section.
type ClassTable struct {
	MaxNum  int16
	Classes []Class
	Tail    []byte // bytes after the end sentinel, kept verbatim
}

func (t *ClassTable) code(c *codec) {
	var zero byte
	c.BS(&t.MaxNum)
	c.RC(&zero)
	c.RC(&zero)
	flag := true
	c.B(&flag)
	if c.w {
		for i := range t.Classes {
			t.Classes[i].code(c)
		}
		return
	}
	for c.dat.Remaining() > 8 { // a class needs far more than 8 bits; the rest is padding
		var k Class
		k.code(c)
		t.Classes = append(t.Classes, k)
	}
}

func (k *Class) code(c *codec) {
	c.BS(&k.Num)
	c.BS(&k.ProxyFlags)
	c.TV(&k.AppName)
	c.TV(&k.CppName)
	c.TV(&k.DXFName)
	c.B(&k.Zombie)
	c.BS(&k.ItemClass)
	c.BL(&k.Count)
	c.BS(&k.DwgVer)
	c.BS(&k.MaintVer)
	c.BL(&k.Unk1)
	c.BL(&k.Unk2)
}

func decodeClasses(data []byte, v Version, text TextCodec) (*ClassTable, error) {
	if len(data) < 0x20 || string(data[:16]) != string(sentinelClassesStart) {
		return nil, fmt.Errorf("classes: bad start sentinel")
	}
	size := int(binary.LittleEndian.Uint32(data[16:]))
	start := 20
	endbit := 0
	if v.R2010 {
		// The bit count is measured from offset 24, i.e. it includes its own RL field.
		bitsize := int(binary.LittleEndian.Uint32(data[24:]))
		start = 28
		endbit = 24*8 + bitsize
	} else if v.R2007 {
		// R2007 (§5.8): RL byte size, RL bit size counted from its own offset 20, data from 24
		bitsize := int(binary.LittleEndian.Uint32(data[20:]))
		start = 24
		endbit = 20*8 + bitsize
	} else {
		endbit = (20 + size) * 8
	}
	r := bits.NewReader(data)
	c := &codec{Version: v, text: text}
	c.dat = r.Sub(start*8, endbit)
	c.str = c.dat
	if v.R2007 {
		str, datEnd := stringStream(r, endbit)
		if str != nil {
			c.str = str
		}
		c.dat.End = datEnd
	}
	t := &ClassTable{}
	if err := catch(func() { t.code(c) }); err != nil {
		return nil, fmt.Errorf("classes: %w", err)
	}
	return t, nil
}

// stringStream locates the R2007+ string stream that ends just before endbit (§20.1).
func stringStream(r *bits.Reader, endbit int) (*bits.Reader, int) {
	flagPos := endbit - 1
	if r.Sub(flagPos, endbit).B() == 0 {
		return nil, flagPos
	}
	pos := flagPos - 16
	strSize := int(uint16(r.Sub(pos, flagPos).RS()))
	if strSize&0x8000 != 0 {
		hi := int(uint16(r.Sub(pos-16, pos).RS()))
		pos -= 16
		strSize = strSize&0x7FFF | hi<<15
	}
	return r.Sub(pos-strSize, pos), pos - strSize
}

// catch converts a panic in a decode function into an error.
func catch(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	f()
	return nil
}
