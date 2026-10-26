package dwg

import (
	"bytes"
	"testing"
)

func TestHeaderDefaultsRoundTrip(t *testing.T) {
	for _, ver := range []string{"AC1018", "AC1027", "AC1032"} {
		v := versionOf(ver)
		h := HeaderDefaults()
		h.HasSizeHi = headerHasSizeHi(v, 8)
		SetHeaderRef(h, "CLAYER", 0x10)
		data := h.Encode(v, textCodecFor(40))
		g, err := DecodeHeader(data, v, textCodecFor(40), 8)
		if err != nil {
			t.Fatalf("%s: %v", ver, err)
		}
		if g.Vars.CLAYER != (Ref{5, 0x10}) || g.Vars.DIMTXT != 2.5 || g.Vars.MENU != "acad" {
			t.Errorf("%s: values lost: %+v %v %q", ver, g.Vars.CLAYER, g.Vars.DIMTXT, g.Vars.MENU)
		}
		if again := g.Encode(v, textCodecFor(40)); !bytes.Equal(again, data) {
			t.Errorf("%s: re-encode differs", ver)
		}
	}
}

func TestSmallSectionDefaults(t *testing.T) {
	v := versionOf("AC1032")
	text := textCodecFor(40)
	jd := JulianDate{Day: 2461340}
	if _, err := DecodeSummaryInfo(DefaultSummaryInfo(jd, jd).Encode(v, text), v, text); err != nil {
		t.Error(err)
	}
	if _, err := DecodeAppInfo(DefaultAppInfo("1", "", "x").Encode(text), text); err != nil {
		t.Error(err)
	}
	if a, err := DecodeAuxHeader(DefaultAuxHeader("AC1032", 0, jd, jd, 0x2EDC).Encode(v), v); err != nil || a.HandSeed != 0x2EDC {
		t.Error(err)
	}
	if p, err := DecodePreview(DefaultPreview().Encode()); err != nil || len(p.Entries) != 0 {
		t.Error(err)
	}
}
