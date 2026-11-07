package dwg

import (
	"fmt"
	"testing"
)

// cRoundTrip encodes a synthetic entity and decodes it back, checking that the
// type's data()/handles() are symmetric. It does not prove spec conformance.
func cRoundTrip(t *testing.T, ver string, typ int, body Body, fresh func() Body) {
	t.Helper()
	v := versionOf(ver)
	o := &Object{Handle: 0x1234, Type: typ, IsEntity: true, XDicMiss: true,
		Ent:  &EntityCommon{EntMode: 2, LtypeScale: 1, Layer: Ref{5, 0x10}, Color: Color{Raw: 256}},
		Body: body}
	text := textCodecFor(30)
	raw := o.Encode(v, text)
	c, gotTyp, _, err := splitObject(raw, 0, v)
	if err != nil {
		t.Fatalf("%s: %v", ver, err)
	}
	c.text = text
	got := &Object{Type: gotTyp, IsEntity: true, Ent: &EntityCommon{}}
	b := fresh()
	if err := catch(func() {
		got.decodeCommon(c)
		b.data(c, got)
		got.codeCommonHandles(c)
		b.handles(c, got)
	}); err != nil {
		t.Fatalf("%s: decode: %v", ver, err)
	}
	if fmt.Sprintf("%+v", b) != fmt.Sprintf("%+v", body) { // nil and empty slices print alike
		t.Errorf("%s: mismatch\n got %+v\nwant %+v", ver, b, body)
	}
	if c.dat.Remaining() != 0 {
		t.Errorf("%s: %d data bits left", ver, c.dat.Remaining())
	}
}

func TestCTypesRoundTrip(t *testing.T) {
	for _, ver := range []string{"AC1018", "AC1024", "AC1032"} {
		v := versionOf(ver)
		dc := DimCommon{Extrusion: [3]float64{0, 0, 1}, TextMidpt: [2]float64{1, 2}, UserText: "abc",
			InsScale: [3]float64{1, 1, 1}, Attachment: 5, LSpaceStyle: 1, LSpaceFactor: 1, ActualMeasure: 12.5,
			Pt12: [2]float64{3, 4}, DimStyle: Ref{5, 0x20}, Block: Ref{5, 0x21}}
		if v.R2007 {
			dc.FlipArrow1 = true
		}
		cRoundTrip(t, ver, 0x19, &DimRadius{DimCommon: dc, Pt10: [3]float64{1, 1, 0}, Pt15: [3]float64{2, 2, 0}, LeaderLen: 3},
			func() Body { return &DimRadius{} })
		cRoundTrip(t, ver, 0x1F2, &DimArc{DimCommon: dc, IsPartial: true, StartAngle: 0.5, HasLeader: true},
			func() Body { return &DimArc{} })

		img := &Image{ClassVersion: 0, Pt0: [3]float64{1, 2, 3}, UVec: [3]float64{1, 0, 0}, VVec: [3]float64{0, 1, 0},
			Size: [2]float64{640, 480}, DisplayProps: 7, Brightness: 50, Contrast: 50, ClipBndType: 2,
			ClipVerts: [][2]float64{{0, 0}, {1, 0}, {1, 1}}, ImageDef: Ref{5, 0x30}, ImageDefReactor: Ref{3, 0x31}}
		cRoundTrip(t, ver, 0x1F3, img, func() Body { return &Image{} })

		ac := &Acis{Version: 1, Data: AcisData{Blocks: [][]byte{[]byte("abc"), []byte("def")}},
			Wireframe: true, PointPresent: true, Point: [3]float64{1, 1, 1}, NumIsolines: 4, IsolinesPresent: true,
			Wires:  []AcisWire{{Type: 1, Color: 256, Points: [][3]float64{{0, 0, 0}, {1, 1, 1}}, HasTransform: true, Scale: 1}},
			Empty2: true}
		if v.R2007 {
			ac.HistoryID = Ref{5, 0x40}
		}
		if v.R2013 { // R2013+: only the revision GUID is in the object, the SAB is in the AcDs section
			ac = &Acis{Unknown: true, Flags2013: 0x4A, GUID: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Tail2013: 2}
		}
		cRoundTrip(t, ver, 0x26, ac, func() Body { return &Acis{} })

		px := &ProxyEntity{ClassID: 499, OrigDXF: false, DataBits: []byte{0xAB, 0xC0}, NDataBits: 10,
			Handles: []Ref{{5, 0x50}, {4, 0x51}}}
		if v.R2018 {
			px.Version, px.MaintVersion = 32, 2
		} else {
			px.DrawingFmt = 0x1F0003
		}
		cRoundTrip(t, ver, 0x1F2, px, func() Body { return &ProxyEntity{} })

		ml := &MLeader{Version: 2, Ctx: MLeaderContext{
			Roots: []MLeaderRoot{{ContentValid: true, Lines: []MLeaderLine{{Points: [][3]float64{{0, 0, 0}, {5, 5, 0}},
				Breaks: []MLeaderLineBreak{{SegmentIndex: 0, Pairs: []MLeaderBreak{{}}}}, Ltype: Ref{5, 1}, Arrow: Ref{5, 2}}}}},
			Scale: 1, HasText: true, TextLabel: "note", TextStyle: Ref{5, 3}, ColumnSizes: []float64{1, 2}},
			Style: Ref{5, 4}, Arrows: []MLeaderArrow{{true, Ref{5, 6}}}, Labels: []MLeaderLabel{{Ref{4, 7}, "x", 1, 2}}}
		if !v.R2010 {
			for i := range ml.Ctx.Roots[0].Lines {
				ml.Ctx.Roots[0].Lines[i].Ltype, ml.Ctx.Roots[0].Lines[i].Arrow = Ref{}, Ref{}
			}
		} else {
			ml.Arrows = nil // entity-level arrow overrides exist only in the pre-R2010 layout
		}
		cRoundTrip(t, ver, 0x1F2, ml, func() Body { return &MLeader{} })

		vp := &ViewportEnt{Width: 10, Height: 5, FrozenLayers: []Ref{{5, 8}}, StyleSheet: "a.ctb", ClipBoundary: Ref{4, 9}}
		cRoundTrip(t, ver, 0x22, vp, func() Body { return &ViewportEnt{} })

		if !v.R2010 {
			tb := &Table{Scale: [3]float64{2, 2, 2}, HasAttribs: false, Flags: 6,
				ColWidths: []float64{10, 20}, RowHeights: []float64{5},
				Cells: []TableCell{{Type: 1, Text: "a", HasOverride: true, FlagOverride: 0x10 | 0x40, EdgeColor: [4]CmColor{{Index: 1}}, TextStyle: Ref{5, 0xA}},
					{Type: 2, BlockScale: 1, HasAttrs: true, Attrs: []TableAttr{{1, "v", Ref{4, 0xB}}}}},
				BlockHeader: Ref{5, 0xC}, TableStyle: Ref{5, 0xD},
				HasTableOverrides: true, TableOverride: 0x01 | 0x20 | 0x20000, TitleSuppressed: true, RowTextStyle: [3]Ref{{5, 0xE}}}
			cRoundTrip(t, ver, 0x1F2, tb, func() Body { return &Table{} })
		} else {
			// R2010+ layout: the embedded content with a styled cell, a string value, geometry and a merge
			tb := &Table{Scale: [3]float64{2, 2, 2}, HorDir: [3]float64{1, 0, 0}, BlockHeader: Ref{5, 0xC}, UnknownH: Ref{4, 0},
				Unknown2010B: !v.R2013, Unknown38: 38, Ranges: []TableRowRange{{EndRow: 1}},
				Content: TableContent{Name: "t", TableStyle: Ref{5, 0xD}, Style: CellStyle{Type: 4},
					Columns: []TableColumn{{Style: CellStyle{Type: 3}, Width: 10}},
					Rows: []ContentRow{{Style: CellStyle{Type: 2}, StyleID: 1, Height: 5, Cells: []ContentCell{{
						Style: CellStyle{Type: 1, DataFlags: 1, PropOverride: 0x10, Content: ContentFormat{PropOverride: 0x10, CellAlignment: 5, TextStyle: Ref{5, 0xA}},
							Borders: []CellBorder{{EdgeFlags: 1, Override: 0x8, Type: 1, Color: CmColor{Index: 1}, Ltype: Ref{5, 0}}}},
						Contents: []CellContent{{Type: 1, Value: TableValue{Flags: 6, DataType: 4, Str: "a", Bytes: []byte{0x61, 0, 0, 0}, ValueStr: "a"},
							HasFormat: 1, Format: ContentFormat{TextStyle: Ref{5, 0xA}}}},
						HasGeom: 1, GeomFlags: 1, UnknownH: Ref{4, 0}, Geometry: CellGeometry{Width: 2}}}}},
					Merged: [][4]int32{{0, 0, 0, 0}}}}
			cRoundTrip(t, ver, 0x1F2, tb, func() Body { return &Table{} })
		}
	}
}
