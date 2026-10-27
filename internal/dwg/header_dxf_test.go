package dwg

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/LeeBhin/dwgconv/internal/container"
)

// refDXF pairs a sample drawing (.work/orig/sN.dwg) with its ODA-converted DXF (.work/oda/sN.dxf).
var refDXF = map[string]string{"s2": "s2.dxf", "s3": "s3.dxf", "s7": "s7.dxf"}

const refDir = "../../.work/oda/"

func readDXFHeader(path string) (map[string][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	out := map[string][]string{}
	cur := ""
	for sc.Scan() {
		code := strings.TrimSpace(sc.Text())
		if !sc.Scan() {
			break
		}
		val := strings.TrimRight(sc.Text(), "\r")
		if code == "0" && val == "ENDSEC" {
			break
		}
		if code == "9" {
			cur = val
			continue
		}
		if cur != "" {
			out[cur] = append(out[cur], strings.TrimSpace(val))
		}
	}
	return out, nil
}

// odaChanges are variables the converter itself updates when it saves the DXF.
var odaChanges = map[string]bool{"$TDUPDATE": true, "$TDUUPDATE": true, "$HANDSEED": true}

func TestHeaderValuesAgainstODA(t *testing.T) {
	for sample, dxf := range refDXF {
		f, err := container.ReadFile("../../.work/orig/" + sample + ".dwg")
		if err != nil {
			t.Skip(err)
		}
		ref, err := readDXFHeader(refDir + dxf)
		if err != nil {
			t.Skip(err)
		}
		h, err := DecodeFileHeader(f)
		if err != nil {
			t.Fatal(err)
		}
		if !versionOf(f.Version).R2007 {
			SetR2007HeaderDefaults(h)
		}
		hv := reflect.ValueOf(h.Vars)
		checked, bad := 0, 0
		for _, e := range HeaderDXFVars {
			want, ok := ref[e.Name]
			// R2004 has no CMATERIAL; the converter points it at the "ByLayer" MATERIAL object.
			if !ok || odaChanges[e.Name] || (e.Name == "$CMATERIAL" && !versionOf(f.Version).R2007) {
				continue
			}
			var got []string
			switch {
			case strings.HasPrefix(e.Special, "flags:") || strings.HasPrefix(e.Special, "nflags:"):
				m, _ := strconv.ParseInt(e.Special[strings.Index(e.Special, ":")+3:], 16, 32)
				v := int64(h.Vars.Flags) & m
				for m&1 == 0 {
					m >>= 1
					v >>= 1
				}
				if strings.HasPrefix(e.Special, "n") {
					v ^= 1
				}
				if e.Name == "$CELWEIGHT" {
					v = int64(lineweightFromIndex(int(v)))
				}
				got = []string{fmt.Sprint(v)}
			case e.Field != "":
				fv := hv.FieldByName(e.Field)
				got = dxfStrings(fv.Interface(), e.Codes)
				if got == nil {
					continue // handle written as a name: needs the object table
				}
			default:
				continue
			}
			checked++
			if !sameValues(got, want) {
				bad++
				t.Errorf("%s %s: got %v want %v", sample, e.Name, got, want)
			}
		}
		t.Logf("%s: %d variables checked, %d mismatches", sample, checked, bad)
	}
}

// lineweightFromIndex maps the 5-bit CELWEIGHT index to the DXF lineweight (§9 Flags).
func lineweightFromIndex(i int) int {
	table := []int{0, 5, 9, 13, 15, 18, 20, 25, 30, 35, 40, 50, 53, 60, 70, 80, 90, 100, 106, 120, 140, 158, 200, 211}
	switch {
	case i < len(table):
		return table[i]
	case i == 29:
		return -1
	case i == 30:
		return -2
	case i == 31:
		return -3
	}
	return 0
}

func dxfStrings(v any, codes []int) []string {
	switch x := v.(type) {
	case bool:
		if x {
			return []string{"1"}
		}
		return []string{"0"}
	case int16, int32, byte, uint64:
		return []string{fmt.Sprint(x)}
	case float64:
		return []string{fmt.Sprint(x)}
	case [3]float64:
		return []string{fmt.Sprint(x[0]), fmt.Sprint(x[1]), fmt.Sprint(x[2])}
	case [2]float64:
		return []string{fmt.Sprint(x[0]), fmt.Sprint(x[1])}
	case string:
		return []string{x}
	case JulianDate:
		return []string{fmt.Sprint(x.Float())}
	case CmColor:
		return []string{fmt.Sprint(x.DXFIndex())}
	case Ref:
		if codes[0] == 5 || codes[0] >= 300 {
			return []string{strings.ToUpper(strconv.FormatUint(x.H, 16))}
		}
		return nil
	}
	return []string{fmt.Sprint(v)}
}

func sameValues(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] == want[i] {
			continue
		}
		a, e1 := strconv.ParseFloat(got[i], 64)
		b, e2 := strconv.ParseFloat(want[i], 64)
		if e1 != nil || e2 != nil || math.Abs(a-b) > 1e-9*math.Max(1, math.Abs(b)) {
			return false
		}
	}
	return true
}
