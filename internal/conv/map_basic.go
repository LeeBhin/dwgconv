package conv

import (
	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

func init() {
	register("LINE", 0x13, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Line) {
		m.Subclass("AcDbLine")
		m.FloatDef(39, &b.Thickness, 0)
		m.Point3(10, &b.Start)
		m.Point3(11, &b.End)
		m.Extrusion(&b.Extrusion)
		if m.Reading() {
			b.ZZero = b.Start[2] == 0 && b.End[2] == 0
		}
	})
	register("TEXT", 0x01, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Text) {
		textFields(m, x, &b.TextCommon, "AcDbText")
		m.Subclass("AcDbText")
		m.Int16Def(73, &b.VAlign, 0)
		if m.Reading() {
			b.DataFlags = textDataFlags(&b.TextCommon)
		}
	})
}

// textFields binds the AcDbText part shared by TEXT, ATTRIB and ATTDEF (vertical alignment excluded,
// since ATTRIB/ATTDEF place it in their own subclass).
func textFields(m *dxf.Mapper, x *ctx, t *dwg.TextCommon, subclass string) {
	m.Subclass(subclass)
	m.FloatDef(39, &t.Thickness, 0)
	ins := [3]float64{t.Insertion[0], t.Insertion[1], t.Elevation}
	m.Point3(10, &ins)
	m.Float(40, &t.Height)
	m.Str(1, &t.Value)
	m.AngleDef(50, &t.Rotation, 0)
	m.FloatDef(41, &t.WidthFact, 1)
	m.AngleDef(51, &t.Oblique, 0)
	x.NameRefDef(m, 7, &t.Style, "STYLE", 5, "Standard")
	m.Int16Def(71, &t.Generation, 0)
	m.Int16Def(72, &t.HAlign, 0)
	al := [3]float64{t.Alignment[0], t.Alignment[1], t.Elevation}
	if !m.Reading() {
		// written when stored, and whenever the text is aligned (ODA writes 0,0 then)
		if t.DataFlags&0x02 == 0 || t.HAlign != 0 || t.VAlign != 0 {
			m.Point3(11, &al)
		}
	} else if m.Has(11) {
		m.Point3(11, &al)
	}
	m.Extrusion(&t.Extrusion)
	if m.Reading() {
		t.Insertion = [2]float64{ins[0], ins[1]}
		t.Elevation = ins[2]
		t.Alignment = [2]float64{al[0], al[1]}
	}
}

// textDataFlags derives the R2000+ TEXT DataFlags (§20.4.3) from the values: a set bit means "default, not stored".
func textDataFlags(t *dwg.TextCommon) byte {
	var f byte
	if t.Elevation == 0 {
		f |= 0x01
	}
	if t.HAlign == 0 && t.VAlign == 0 {
		f |= 0x02
	}
	if t.Oblique == 0 {
		f |= 0x04
	}
	if t.Rotation == 0 {
		f |= 0x08
	}
	if t.WidthFact == 1 {
		f |= 0x10
	}
	if t.Generation == 0 {
		f |= 0x20
	}
	if t.HAlign == 0 {
		f |= 0x40
	}
	if t.VAlign == 0 {
		f |= 0x80
	}
	return f
}
