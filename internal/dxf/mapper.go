package dxf

import (
	"math"
	"strconv"
	"strings"
)

// Mapper binds DXF groups to Go values in either direction, so each record layout is
// described once. Writing emits groups in call order. Reading consumes the record
// sequentially: each binding searches forward for its code, but never past the next
// subclass marker (100), so absent optional groups keep their default.
type Mapper struct {
	w   *Writer
	rec *Record
	pos int
}

func NewWriteMapper(w *Writer) *Mapper { return &Mapper{w: w} }
func NewReadMapper(r *Record) *Mapper  { return &Mapper{rec: r} }
func (m *Mapper) Reading() bool        { return m.w == nil }
func (m *Mapper) Writer() *Writer      { return m.w }
func (m *Mapper) Record() *Record      { return m.rec }
func (m *Mapper) Rest() []Group        { return m.rec.Groups[m.pos:] }
func (m *Mapper) Seek(pos int)         { m.pos = pos }
func (m *Mapper) Pos() int             { return m.pos }

// find returns the index of the next group with code, stopping at a subclass marker.
func (m *Mapper) find(code int) int {
	for i := m.pos; i < len(m.rec.Groups); i++ {
		c := m.rec.Groups[i].Code
		if c == code {
			return i
		}
		if c == 100 && code != 100 {
			return -1
		}
	}
	return -1
}

func (m *Mapper) take(code int) (Group, bool) {
	i := m.find(code)
	if i < 0 {
		return Group{}, false
	}
	m.pos = i + 1
	return m.rec.Groups[i], true
}

// Peek reports whether the next group (at the cursor) has the given code.
func (m *Mapper) Peek(code int) bool {
	return m.pos < len(m.rec.Groups) && m.rec.Groups[m.pos].Code == code
}

// Has reports whether a group with code follows before the next subclass marker.
func (m *Mapper) Has(code int) bool { return m.find(code) >= 0 }

// Subclass writes or skips to a subclass marker.
func (m *Mapper) Subclass(name string) {
	if !m.Reading() {
		m.w.Str(100, name)
		return
	}
	for i := m.pos; i < len(m.rec.Groups); i++ {
		g := m.rec.Groups[i]
		if g.Code == 100 && g.Value == name {
			m.pos = i + 1
			return
		}
	}
}

func (m *Mapper) Str(code int, v *string) {
	if !m.Reading() {
		m.w.Str(code, *v)
		return
	}
	if g, ok := m.take(code); ok {
		*v = g.Value
	}
}

// StrOpt writes only when v is not empty.
func (m *Mapper) StrOpt(code int, v *string) {
	if !m.Reading() && *v == "" {
		return
	}
	m.Str(code, v)
}

func (m *Mapper) Float(code int, v *float64) {
	if !m.Reading() {
		m.w.Float(code, *v)
		return
	}
	if g, ok := m.take(code); ok {
		*v = g.Float()
	}
}

// FloatDef writes only when v differs from def; reading an absent group yields def.
func (m *Mapper) FloatDef(code int, v *float64, def float64) {
	if !m.Reading() {
		if *v != def {
			m.w.Float(code, *v)
		}
		return
	}
	*v = def
	if g, ok := m.take(code); ok {
		*v = g.Float()
	}
}

// AngleDef binds an angle stored in radians in DWG and degrees in DXF; omitted on write when equal to def.
func (m *Mapper) AngleDef(code int, rad *float64, def float64) {
	if !m.Reading() {
		if *rad != def {
			m.w.raw(code, FormatFloat(roundDeg(*rad*180/math.Pi)))
		}
		return
	}
	*rad = def
	if g, ok := m.take(code); ok {
		*rad = g.Float() * math.Pi / 180
	}
}

// Angle binds a radian angle as DXF degrees, always written.
func (m *Mapper) Angle(code int, rad *float64) {
	if !m.Reading() {
		m.w.raw(code, FormatFloat(roundDeg(*rad*180/math.Pi)))
		return
	}
	if g, ok := m.take(code); ok {
		*rad = g.Float() * math.Pi / 180
	}
}

// roundDeg drops the last-bit noise of the radian to degree conversion (90.00000000000001 -> 90).
func roundDeg(v float64) float64 {
	r, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'g', 15, 64), 64)
	return r
}

func (m *Mapper) Int16(code int, v *int16) {
	if !m.Reading() {
		m.w.Int(code, int64(*v))
		return
	}
	if g, ok := m.take(code); ok {
		*v = int16(g.Int())
	}
}

func (m *Mapper) Int16Def(code int, v *int16, def int16) {
	if !m.Reading() {
		if *v != def {
			m.w.Int(code, int64(*v))
		}
		return
	}
	*v = def
	if g, ok := m.take(code); ok {
		*v = int16(g.Int())
	}
}

func (m *Mapper) Int32(code int, v *int32) {
	if !m.Reading() {
		m.w.Int(code, int64(*v))
		return
	}
	if g, ok := m.take(code); ok {
		*v = int32(g.Int())
	}
}

func (m *Mapper) Int32Def(code int, v *int32, def int32) {
	if !m.Reading() {
		if *v != def {
			m.w.Int(code, int64(*v))
		}
		return
	}
	*v = def
	if g, ok := m.take(code); ok {
		*v = int32(g.Int())
	}
}

func (m *Mapper) Byte(code int, v *byte) {
	x := int16(*v)
	m.Int16(code, &x)
	*v = byte(x)
}

func (m *Mapper) Bool(code int, v *bool) {
	if !m.Reading() {
		m.w.Bool(code, *v)
		return
	}
	if g, ok := m.take(code); ok {
		*v = g.Int() != 0
	}
}

// Point2 binds code and code+10.
func (m *Mapper) Point2(code int, v *[2]float64) {
	m.Float(code, &v[0])
	m.Float(code+10, &v[1])
}

// Point3 binds code, code+10, code+20.
func (m *Mapper) Point3(code int, v *[3]float64) {
	m.Float(code, &v[0])
	m.Float(code+10, &v[1])
	m.Float(code+20, &v[2])
}

// Point3Def omits the whole point on write when it equals def.
func (m *Mapper) Point3Def(code int, v *[3]float64, def [3]float64) {
	if !m.Reading() {
		if *v != def {
			m.Point3(code, v)
		}
		return
	}
	*v = def
	if m.Has(code) {
		m.Point3(code, v)
	}
}

// Extrusion binds 210/220/230, omitted when (0,0,1).
func (m *Mapper) Extrusion(v *[3]float64) { m.Point3Def(210, v, [3]float64{0, 0, 1}) }

func (m *Mapper) Handle(code int, v *uint64) {
	if !m.Reading() {
		m.w.Handle(code, *v)
		return
	}
	if g, ok := m.take(code); ok {
		*v = g.Handle()
	}
}

// HandleOpt writes only non-zero handles.
func (m *Mapper) HandleOpt(code int, v *uint64) {
	if !m.Reading() && *v == 0 {
		return
	}
	m.Handle(code, v)
}

func (m *Mapper) Binary(code int, v *[]byte) {
	if !m.Reading() {
		m.w.Binary(code, *v)
		return
	}
	var out []byte
	for m.pos < len(m.rec.Groups) && m.rec.Groups[m.pos].Code != code {
		if i := m.find(code); i >= 0 {
			m.pos = i
		} else {
			return
		}
	}
	for m.pos < len(m.rec.Groups) && m.rec.Groups[m.pos].Code == code {
		out = append(out, DecodeBinary(m.rec.Groups[m.pos].Value)...)
		m.pos++
	}
	*v = out
}

// Count binds a count group: writing emits n, reading returns the stored value.
func (m *Mapper) Count(code int, n int) int {
	v := int64(n)
	if !m.Reading() {
		m.w.Int(code, v)
		return n
	}
	if g, ok := m.take(code); ok {
		return int(g.Int())
	}
	return 0
}

// Value of the first group with code anywhere in the record (read helper).
func (r *Record) Value(code int) (string, bool) {
	for _, g := range r.Groups {
		if g.Code == code {
			return g.Value, true
		}
	}
	return "", false
}

func (r *Record) Handle(code int) uint64 {
	v, _ := r.Value(code)
	h, _ := strconv.ParseUint(strings.TrimSpace(v), 16, 64)
	return h
}
