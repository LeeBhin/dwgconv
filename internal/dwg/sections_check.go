package dwg

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"dwgconv/internal/container"
)

// SectionCheck is the result of a decode/re-encode round trip of one section.
type SectionCheck struct {
	Name   string
	Result string
}

// fileText returns the version flags and text codec of a container.
func fileText(f *container.File) (Version, TextCodec) {
	return versionOf(f.Version), textCodecFor(int(binary.LittleEndian.Uint16(f.Prefix[0x13:])))
}

// DecodeFileHeader decodes the AcDb:Header section of a container.
func DecodeFileHeader(f *container.File) (*Header, error) {
	s := f.Section("AcDb:Header")
	if s == nil {
		return nil, fmt.Errorf("no AcDb:Header")
	}
	v, text := fileText(f)
	return DecodeHeader(s.Data, v, text, f.Prefix[0x0B])
}

// CheckSections round-trips every non-object section it knows and reports byte identity.
func CheckSections(f *container.File) []SectionCheck {
	v, text := fileText(f)
	var out []SectionCheck
	check := func(name string, enc func(data []byte) ([]byte, string, error)) {
		s := f.Section(name)
		if s == nil {
			out = append(out, SectionCheck{name, "absent"})
			return
		}
		got, note, err := enc(s.Data)
		res := ""
		switch {
		case err != nil:
			res = "ERROR " + err.Error()
		case bytes.Equal(got, s.Data):
			res = "identical"
		default:
			res = fmt.Sprintf("DIFFERENT (orig %d, enc %d, first diff @%d)", len(s.Data), len(got), firstDiff(got, s.Data))
		}
		if note != "" {
			res += " " + note
		}
		out = append(out, SectionCheck{name, res})
	}
	check("AcDb:Header", func(d []byte) ([]byte, string, error) {
		h, err := DecodeHeader(d, v, text, f.Prefix[0x0B])
		if err != nil {
			return nil, "", err
		}
		note := ""
		if h.LeftData != 0 || h.LeftHandles != 0 {
			note = fmt.Sprintf("[left data=%d handles=%d]", h.LeftData, h.LeftHandles)
		}
		return h.Encode(v, text), note, nil
	})
	check("AcDb:Classes", func(d []byte) ([]byte, string, error) {
		t, err := DecodeClassSection(d, v, text)
		if err != nil {
			return nil, "", err
		}
		return t.Encode(v, text), fmt.Sprintf("[%d classes]", len(t.Classes)), nil
	})
	check("AcDb:SummaryInfo", func(d []byte) ([]byte, string, error) {
		x, err := DecodeSummaryInfo(d, v, text)
		if err != nil {
			return nil, "", err
		}
		return x.Encode(v, text), fmt.Sprintf("[last saved by %q]", x.LastSavedBy), nil
	})
	check("AcDb:AppInfo", func(d []byte) ([]byte, string, error) {
		x, err := DecodeAppInfo(d, text)
		if err != nil {
			return nil, "", err
		}
		return x.Encode(text), fmt.Sprintf("[%d records]", len(x.Records)), nil
	})
	check("AcDb:RevHistory", func(d []byte) ([]byte, string, error) {
		x, err := DecodeRevHistory(d)
		if err != nil {
			return nil, "", err
		}
		return x.Encode(), fmt.Sprint(x.Values), nil
	})
	check("AcDb:Template", func(d []byte) ([]byte, string, error) {
		x, err := DecodeTemplate(d, v)
		if err != nil {
			return nil, "", err
		}
		return x.Encode(v), fmt.Sprintf("[MEASUREMENT %d]", x.Measurement), nil
	})
	check("AcDb:ObjFreeSpace", func(d []byte) ([]byte, string, error) {
		x, err := DecodeObjFreeSpace(d, v)
		if err != nil {
			return nil, "", err
		}
		return x.Encode(v), "", nil
	})
	check("AcDb:AuxHeader", func(d []byte) ([]byte, string, error) {
		x, err := DecodeAuxHeader(d, v)
		if err != nil {
			return nil, "", err
		}
		return x.Encode(v), fmt.Sprintf("[saves %d, handseed %x]", x.Saves, x.HandSeed), nil
	})
	check("AcDb:FileDepList", func(d []byte) ([]byte, string, error) {
		x, err := DecodeFileDepList(d, v, text)
		if err != nil {
			return nil, "", err
		}
		return x.Encode(v, text), fmt.Sprintf("[%d files, tail %d]", len(x.Files), len(x.Tail)), nil
	})
	check("AcDb:Preview", func(d []byte) ([]byte, string, error) {
		x, err := DecodePreview(d)
		if err != nil {
			return nil, "", err
		}
		codes := []byte{}
		for _, e := range x.Entries {
			codes = append(codes, e.Code)
		}
		return x.Encode(), fmt.Sprintf("[codes %v]", codes), nil
	})
	return out
}

func firstDiff(a, b []byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

// FileVersion returns the version flags of a container.
func FileVersion(f *container.File) (Version, error) { v, _ := fileText(f); return v, nil }

// FileText returns the text codec of a container.
func FileText(f *container.File) TextCodec { _, t := fileText(f); return t }
