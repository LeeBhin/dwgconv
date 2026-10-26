// Package dxf reads and writes ASCII DXF as a sequence of group code/value pairs,
// grouped into sections and records (one record per 0-group).
package dxf

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// Group is one code/value pair. Value is kept as text; typed accessors parse it.
type Group struct {
	Code  int
	Value string
}

func (g Group) Float() float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(g.Value), 64)
	return v
}

func (g Group) Int() int64 {
	s := strings.TrimSpace(g.Value)
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		f, _ := strconv.ParseFloat(s, 64)
		return int64(f)
	}
	return v
}

func (g Group) Handle() uint64 {
	v, _ := strconv.ParseUint(strings.TrimSpace(g.Value), 16, 64)
	return v
}

// Record is a 0-group and the groups that follow it up to the next 0-group.
type Record struct {
	Type   string
	Groups []Group
}

// Section is a DXF section (HEADER, CLASSES, TABLES, BLOCKS, ENTITIES, OBJECTS, ...).
type Section struct {
	Name    string
	Records []*Record
}

// File is a parsed DXF.
type File struct {
	Sections []*Section
}

func (f *File) Section(name string) *Section {
	for _, s := range f.Sections {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// Parse reads an ASCII DXF (UTF-8). For the HEADER section, each $VARIABLE becomes a record
// whose Type is the variable name.
func Parse(r io.Reader) (*File, error) {
	br := bufio.NewReaderSize(r, 1<<20)
	line := 0
	next := func() (Group, error) {
		c, err := br.ReadString('\n')
		if err != nil {
			return Group{}, err
		}
		v, err := br.ReadString('\n')
		if err != nil && !(err == io.EOF && v != "") {
			return Group{}, fmt.Errorf("dxf: line %d: missing value", line+2)
		}
		line += 2
		code, err := strconv.Atoi(strings.TrimSpace(c))
		if err != nil {
			return Group{}, fmt.Errorf("dxf: line %d: bad group code %q", line-1, c)
		}
		return Group{code, strings.TrimRight(v, "\r\n")}, nil
	}
	f := &File{}
	var sec *Section
	var rec *Record
	header := false
	for {
		g, err := next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch {
		case g.Code == 0 && g.Value == "SECTION":
			n, err := next()
			if err != nil {
				return nil, err
			}
			sec = &Section{Name: n.Value}
			f.Sections = append(f.Sections, sec)
			rec = nil
			header = n.Value == "HEADER"
		case g.Code == 0 && g.Value == "ENDSEC":
			sec, rec, header = nil, nil, false
		case g.Code == 0 && g.Value == "EOF":
			return f, nil
		case sec == nil:
			// stray groups outside sections are ignored
		case header && g.Code == 9:
			rec = &Record{Type: g.Value}
			sec.Records = append(sec.Records, rec)
		case !header && g.Code == 0:
			rec = &Record{Type: g.Value}
			sec.Records = append(sec.Records, rec)
		case rec != nil:
			rec.Groups = append(rec.Groups, g)
		}
	}
	return f, nil
}

// Writer emits DXF text.
type Writer struct {
	w   *bufio.Writer
	err error
}

func NewWriter(w io.Writer) *Writer { return &Writer{w: bufio.NewWriterSize(w, 1<<20)} }

func (w *Writer) Flush() error {
	if w.err != nil {
		return w.err
	}
	return w.w.Flush()
}

func (w *Writer) raw(code int, value string) {
	if w.err != nil {
		return
	}
	_, w.err = fmt.Fprintf(w.w, "%3d\r\n%s\r\n", code, value)
}

func (w *Writer) Str(code int, s string) { w.raw(code, s) }

func (w *Writer) Int(code int, v int64) {
	switch {
	case code >= 60 && code <= 79, code >= 170 && code <= 179, code >= 270 && code <= 299,
		code >= 370 && code <= 389, code >= 400 && code <= 409, code >= 1060 && code <= 1070:
		w.raw(code, fmt.Sprintf("%6d", v))
	case code >= 160 && code <= 169:
		w.raw(code, fmt.Sprintf("%18d", v))
	default:
		w.raw(code, fmt.Sprintf("%9d", v))
	}
}

func (w *Writer) Float(code int, v float64) { w.raw(code, FormatFloat(v)) }

func (w *Writer) Handle(code int, h uint64) { w.raw(code, strings.ToUpper(strconv.FormatUint(h, 16))) }

func (w *Writer) Bool(code int, b bool) {
	if b {
		w.Int(code, 1)
	} else {
		w.Int(code, 0)
	}
}

// Binary writes data as hex lines of at most 127 bytes (codes 310-319, 1004).
func (w *Writer) Binary(code int, data []byte) {
	const chunk = 127
	for len(data) > 0 {
		n := min(chunk, len(data))
		w.raw(code, strings.ToUpper(fmt.Sprintf("%x", data[:n])))
		data = data[n:]
	}
}

func (w *Writer) Group(g Group) { w.raw(g.Code, g.Value) }

// FormatFloat formats like AutoCAD: shortest round-trip text, always with a decimal point.
func FormatFloat(v float64) string {
	if v == 0 {
		return "0.0"
	}
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return "0.0"
	}
	s := strconv.FormatFloat(v, 'g', -1, 64)
	if strings.ContainsAny(s, "e") {
		// AutoCAD writes exponents as E+NN
		m, e, _ := strings.Cut(s, "e")
		if !strings.Contains(m, ".") {
			m += ".0"
		}
		ei, _ := strconv.Atoi(e)
		sign := "+"
		if ei < 0 {
			sign = "-"
			ei = -ei
		}
		return fmt.Sprintf("%sE%s%02d", m, sign, ei)
	}
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// Write serializes a File as-is.
func (f *File) Write(out io.Writer) error {
	w := NewWriter(out)
	for _, s := range f.Sections {
		w.Str(0, "SECTION")
		w.Str(2, s.Name)
		for _, r := range s.Records {
			if s.Name == "HEADER" {
				w.Str(9, r.Type)
			} else {
				w.Str(0, r.Type)
			}
			for _, g := range r.Groups {
				w.Group(g)
			}
		}
		w.Str(0, "ENDSEC")
	}
	w.Str(0, "EOF")
	return w.Flush()
}

// DecodeBinary parses hex text of a binary group.
func DecodeBinary(s string) []byte {
	s = strings.TrimSpace(s)
	out := make([]byte, 0, len(s)/2)
	for i := 0; i+1 < len(s); i += 2 {
		v, err := strconv.ParseUint(s[i:i+2], 16, 8)
		if err != nil {
			break
		}
		out = append(out, byte(v))
	}
	return out
}

