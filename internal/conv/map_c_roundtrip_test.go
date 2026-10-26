package conv

import (
	"bytes"
	"testing"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// TestCMappingsRoundTrip exports synthetic group C objects to DXF and imports them back.
// It checks the mappings are symmetric; it does not prove agreement with AutoCAD/ODA.
func TestCMappingsRoundTrip(t *testing.T) {
	for _, v := range []dwg.Version{{}, {R2007: true, R2010: true, R2013: true, R2018: true}} {
		d := &dwg.Document{Version: v, Classes: &dwg.ClassTable{}, ByHandle: map[uint64]*dwg.Object{}}
		x := newCtx(d)
		ent := func() *dwg.EntityCommon {
			return &dwg.EntityCommon{EntMode: 2, LtypeScale: 1, Lineweight: 29, Color: dwg.Color{Raw: 256}}
		}
		dc := dwg.DimCommon{Extrusion: [3]float64{0, 0, 1}, TextMidpt: [2]float64{1, 2}, Flags1: 0x0B, UserText: "abc",
			InsScale: [3]float64{1, 1, 1}, Attachment: 5, LSpaceStyle: 1, LSpaceFactor: 1, ActualMeasure: 12.5}
		cases := []*dwg.Object{
			{Type: 0x19, Body: &dwg.DimRadius{DimCommon: dc, Pt10: [3]float64{1, 1, 0}, Pt15: [3]float64{2, 2, 0}, LeaderLen: 3}},
			{Type: 0x17, Body: &dwg.DimAng3Pt{DimCommon: dc, Pt13: [3]float64{1, 0, 0}}},
			{Type: 0x14, Body: &dwg.DimOrdinate{DimCommon: dc, Pt13: [3]float64{1, 0, 0}, Flags2: 1}},
			{Body: &dwg.DimArc{DimCommon: dc, IsPartial: true, StartAngle: 0.5, HasLeader: true}},
			{Body: &dwg.Image{Pt0: [3]float64{1, 2, 3}, UVec: [3]float64{1, 0, 0}, VVec: [3]float64{0, 1, 0}, Size: [2]float64{640, 480},
				DisplayProps: 7, Brightness: 50, Contrast: 50, ClipBndType: 2, ClipVerts: [][2]float64{{0, 0}, {1, 0}, {1, 1}},
				ImageDef: dwg.Ref{Code: 5, H: 0x30}, ImageDefReactor: dwg.Ref{Code: 3, H: 0x31}}},
			{Body: &dwg.Wipeout{Image: dwg.Image{ClipBndType: 1, ClipVerts: [][2]float64{{0, 0}, {1, 1}}}}},
			{Type: 0x26, Body: &dwg.Acis{Version: 1, Empty2: true, Data: dwg.AcisData{Blocks: [][]byte{cEncodeSAT("400 0 1 0\nbody $-1 $1 $-1 $-1 #\n")}}}},
			{Type: 0x1F2, Body: &dwg.ProxyEntity{ClassID: 499, DataBits: []byte{0xAB, 0xC0}, NDataBits: 10,
				Handles: []dwg.Ref{{Code: 5, H: 0x50}, {Code: 4, H: 0x51}}}},
			{Body: &dwg.MLeader{Ctx: dwg.MLeaderContext{
				Roots: []dwg.MLeaderRoot{{ContentValid: true, Lines: []dwg.MLeaderLine{{Points: [][3]float64{{0, 0, 0}, {5, 5, 0}},
					Breaks: []dwg.MLeaderLineBreak{{SegmentIndex: 0, Pairs: []dwg.MLeaderBreak{{End: [3]float64{1, 1, 1}}}}}}}}},
				Scale: 1, HasText: true, TextLabel: "note", TextStyle: dwg.Ref{Code: 5, H: 3}, ColumnSizes: []float64{1, 2}},
				Style: dwg.Ref{Code: 5, H: 4}, Arrows: []dwg.MLeaderArrow{{IsDefault: true, Arrow: dwg.Ref{Code: 5, H: 6}}},
				Labels: []dwg.MLeaderLabel{{AttDef: dwg.Ref{Code: 4, H: 7}, Text: "x", UIIndex: 1, Width: 2}}}},
			{Body: &dwg.Table{Scale: [3]float64{1, 1, 1}, Extrusion: [3]float64{0, 0, 1}, Flags: 6, ColWidths: []float64{10, 20}, RowHeights: []float64{5},
				Cells: []dwg.TableCell{{Type: 1, Text: "a"}, {Type: 2, BlockScale: 1, HasAttrs: true,
					Attrs: []dwg.TableAttr{{Text: "v", AttDef: dwg.Ref{Code: 4, H: 0xB}}}}},
				BlockHeader: dwg.Ref{Code: 5, H: 0xC}, TableStyle: dwg.Ref{Code: 5, H: 0xD}}},
		}
		for i, o := range cases {
			o.Handle, o.IsEntity, o.XDicMiss, o.Ent = uint64(0x100+i), true, true, ent()
			if p, ok := o.Body.(*dwg.ProxyEntity); ok {
				o.Ent.HasGraphic, o.Ent.Graphic = true, []byte{1, 2, 3}
				if v.R2018 {
					p.Version, p.MaintVersion = 32, 2
				}
			}
			var buf bytes.Buffer
			w := dxf.NewWriter(&buf)
			if !x.exportRecord(w, o) {
				t.Fatalf("%T: no mapping", o.Body)
			}
			w.Flush()
			gs := parseGroups(buf.String())
			r := &dxf.Record{Type: gs[0].Value, Groups: gs[1:]}
			got := x.importRecord(r, true)
			if got == nil {
				t.Fatalf("%T: import failed", o.Body)
			}
			if diff := objectDiff(o, got); diff != "" {
				t.Errorf("%+v %T: %s", v, o.Body, diff)
			}
		}
	}
}
