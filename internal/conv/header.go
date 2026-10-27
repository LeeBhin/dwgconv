package conv

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
)

// headerRefTable is the table a name-written header reference lives in.
var headerRefTable = map[string]string{
	"TEXTSTYLE": "STYLE", "DIMTXSTY": "STYLE", "CLAYER": "LAYER", "CELTYPE": "LTYPE",
	"DIMLTYPE": "LTYPE", "DIMLTEX1": "LTYPE", "DIMLTEX2": "LTYPE", "DIMSTYLE": "DIMSTYLE",
	"DIMBLK": "BLOCK_RECORD", "DIMBLK1": "BLOCK_RECORD", "DIMBLK2": "BLOCK_RECORD", "DIMLDRBLK": "BLOCK_RECORD",
	"UCSBASE": "UCS", "UCSNAME": "UCS", "UCSORTHOREF": "UCS", "PUCSBASE": "UCS", "PUCSNAME": "UCS",
	"PUCSORTHOREF": "UCS", "CMLSTYLE": "MLINESTYLE",
}

// dxfMaintVer is the $ACADMAINTVER written for AC1032 DXF output (as ODA writes it).
const dxfMaintVer = 55

// headerInfo carries the header values that live outside AcDb:Header.
type headerInfo struct {
	codepage    string
	lastSavedBy string
	measurement int16
}

// exportHeader writes the HEADER section.
func (x *ctx) exportHeader(w *dxf.Writer, h *dwg.Header, info headerInfo) {
	hv := reflect.ValueOf(&h.Vars).Elem()
	w.Str(0, "SECTION")
	w.Str(2, "HEADER")
	for _, e := range dwg.HeaderDXFVars {
		var vals []string
		switch {
		case e.Special == "acadver":
			vals = []string{"AC1032"}
		case e.Special == "acadmaintver":
			vals = []string{strconv.Itoa(dxfMaintVer)}
		case e.Special == "codepage":
			vals = []string{info.codepage}
		case e.Special == "lastsavedby":
			vals = []string{info.lastSavedBy}
		case e.Special == "measurement":
			vals = []string{strconv.Itoa(int(info.measurement))}
		case strings.HasPrefix(e.Special, "local:"):
			jd := hv.FieldByName(e.Special[len("local:"):]).Interface().(dwg.JulianDate)
			vals = []string{dxf.FormatFloat(julianLocal(jd.Float()))}
		case strings.HasPrefix(e.Special, "flags:") || strings.HasPrefix(e.Special, "nflags:"):
			v := flagValue(h.Vars.Flags, e.Special)
			if e.Name == "$CELWEIGHT" {
				v = int64(lineweightToDXF(byte(v)))
			}
			vals = []string{strconv.FormatInt(v, 10)}
		case e.Field != "":
			vals = x.headerValue(hv.FieldByName(e.Field).Interface(), e)
		}
		if vals == nil {
			continue
		}
		w.Str(9, e.Name)
		for i, c := range e.Codes {
			if i >= len(vals) {
				break
			}
			writeTyped(w, c, vals[i])
		}
	}
	w.Str(0, "ENDSEC")
}

func flagValue(flags int32, special string) int64 {
	m, _ := strconv.ParseInt(special[strings.Index(special, ":")+3:], 16, 32)
	v := int64(flags) & m
	for m != 0 && m&1 == 0 {
		m >>= 1
		v >>= 1
	}
	if strings.HasPrefix(special, "n") {
		v ^= 1
	}
	return v
}

func (x *ctx) headerValue(v any, e dwg.HeaderDXFVar) []string {
	f := dxf.FormatFloat
	switch t := v.(type) {
	case bool:
		if t {
			return []string{"1"}
		}
		return []string{"0"}
	case int16:
		return []string{strconv.Itoa(int(t))}
	case int32:
		return []string{strconv.Itoa(int(t))}
	case byte:
		return []string{strconv.Itoa(int(t))}
	case uint64:
		return []string{strconv.FormatUint(t, 10)}
	case float64:
		return []string{f(t)}
	case [2]float64:
		return []string{f(t[0]), f(t[1])}
	case [3]float64:
		return []string{f(t[0]), f(t[1]), f(t[2])}
	case string:
		return []string{t}
	case dwg.JulianDate:
		return []string{f(t.Float())}
	case dwg.CmColor:
		return []string{strconv.Itoa(t.DXFIndex())}
	case dwg.Ref:
		c := e.Codes[0]
		if c == 5 || c >= 300 {
			return []string{strings.ToUpper(strconv.FormatUint(t.H, 16))}
		}
		return []string{x.names[t.H]}
	}
	return nil
}

// writeTyped writes a value already formatted as text with the integer padding of its code.
func writeTyped(w *dxf.Writer, code int, val string) {
	if isIntCode(code) {
		n, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		if err == nil {
			w.Int(code, n)
			return
		}
	}
	w.Str(code, val)
}

func isIntCode(c int) bool {
	return c >= 60 && c <= 99 || c >= 160 && c <= 179 || c >= 270 && c <= 299 || c >= 370 && c <= 389 ||
		c >= 400 && c <= 409 || c >= 420 && c <= 429 || c >= 440 && c <= 459 || c >= 1060 && c <= 1071
}

// julianLocal converts a UTC Julian date to the local time zone (DXF $TDCREATE/$TDUPDATE).
func julianLocal(jd float64) float64 {
	if jd == 0 {
		return 0
	}
	const unixEpochJD = 2440587.5
	t := time.UnixMilli(int64((jd - unixEpochJD) * 86400000))
	_, off := t.In(time.Local).Zone()
	return jd + float64(off)/86400
}

// importHeader fills header variables from the DXF HEADER section, starting from defaults.
func (x *ctx) importHeader(sec *dxf.Section) *dwg.Header {
	h := dwg.HeaderDefaults()
	hv := reflect.ValueOf(&h.Vars).Elem()
	byName := map[string]*dxf.Record{}
	if sec != nil {
		for _, r := range sec.Records {
			byName[r.Type] = r
		}
	}
	for _, e := range dwg.HeaderDXFVars {
		r := byName[e.Name]
		if r == nil {
			continue
		}
		vals := make([]dxf.Group, 0, len(e.Codes))
		for _, c := range e.Codes {
			for _, g := range r.Groups {
				if g.Code == c {
					vals = append(vals, g)
					break
				}
			}
		}
		if len(vals) == 0 {
			continue
		}
		switch {
		case strings.HasPrefix(e.Special, "flags:") || strings.HasPrefix(e.Special, "nflags:"):
			v := vals[0].Int()
			if e.Name == "$CELWEIGHT" {
				v = int64(lineweightFromDXF(int16(v)))
			}
			setFlag(&h.Vars.Flags, e.Special, v)
		case e.Field != "":
			x.setHeaderField(hv.FieldByName(e.Field), e, vals)
		}
	}
	return h
}

func setFlag(flags *int32, special string, v int64) {
	m, _ := strconv.ParseInt(special[strings.Index(special, ":")+3:], 16, 32)
	if strings.HasPrefix(special, "n") {
		v ^= 1
	}
	shift := 0
	for mm := m; mm != 0 && mm&1 == 0; mm >>= 1 {
		shift++
	}
	*flags = int32(int64(*flags)&^m | (v<<shift)&m)
}

func (x *ctx) setHeaderField(f reflect.Value, e dwg.HeaderDXFVar, g []dxf.Group) {
	switch f.Interface().(type) {
	case bool:
		f.SetBool(g[0].Int() != 0)
	case int16, int32:
		f.SetInt(g[0].Int())
	case byte, uint64:
		f.SetUint(uint64(g[0].Int()))
	case float64:
		f.SetFloat(g[0].Float())
	case [2]float64:
		if len(g) >= 2 {
			f.Set(reflect.ValueOf([2]float64{g[0].Float(), g[1].Float()}))
		}
	case [3]float64:
		if len(g) >= 3 {
			f.Set(reflect.ValueOf([3]float64{g[0].Float(), g[1].Float(), g[2].Float()}))
		}
	case string:
		f.SetString(g[0].Value)
	case dwg.JulianDate:
		v := g[0].Float()
		day := int32(v)
		f.Set(reflect.ValueOf(dwg.JulianDate{Day: day, Ms: int32(math.Round((v - float64(day)) * 86400000))}))
	case dwg.CmColor:
		f.Set(reflect.ValueOf(dwg.ACIColor(int(g[0].Int()))))
	case dwg.Ref:
		c := e.Codes[0]
		if c == 5 || c >= 300 {
			f.Set(reflect.ValueOf(dwg.Ref{Code: 5, H: g[0].Handle()}))
			return
		}
		if t, ok := headerRefTable[e.Field]; ok {
			if hd := x.byName[t][nameKey(g[0].Value)]; hd != 0 {
				f.Set(reflect.ValueOf(dwg.Ref{Code: 5, H: hd}))
			}
		}
	}
}
