package conv

import (
	"strings"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// Unverified against ODA: none of these types occur in the sample drawings.
// Layouts follow the AutoCAD DXF reference.

func init() {
	acisName := map[int]string{0x25: "REGION", 0x26: "3DSOLID", 0x27: "BODY"}
	for _, t := range []int{0x25, 0x27, 0x26} { // 3DSOLID last: it owns the Go-type lookup
		mp := register(acisName[t], t, cAcis)
		mp.name = func(o *dwg.Object) string { return acisName[o.Type] }
	}
	register("IMAGE", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Image) {
		cRaster(m, x, b)
	})
	register("WIPEOUT", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Wipeout) {
		cRaster(m, x, &b.Image)
		m.Subclass("AcDbWipeout")
	})
	register("ACAD_PROXY_ENTITY", 0x1F2, cProxyEntity)
}

// cAcis binds AcDbModelerGeometry: version 70 then the SAT text as 1 (with 3 continuations).
func cAcis(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Acis) {
	m.Subclass("AcDbModelerGeometry")
	ver := int16(1)
	m.Int16(70, &ver)
	if !m.Reading() {
		if !b.Empty && b.Version == 1 {
			for _, line := range strings.Split(strings.TrimRight(string(b.Data.SAT()), "\n"), "\n") {
				first := true
				for {
					n := min(255, len(line))
					code := 1
					if !first {
						code = 3
					}
					s := line[:n]
					m.Str(code, &s)
					line, first = line[n:], false
					if line == "" {
						break
					}
				}
			}
		}
	} else {
		var sat strings.Builder
		for _, g := range m.Rest() {
			switch g.Code {
			case 1:
				if sat.Len() > 0 {
					sat.WriteByte('\n')
				}
				sat.WriteString(g.Value)
			case 3:
				sat.WriteString(g.Value)
			}
		}
		b.Empty = sat.Len() == 0
		if !b.Empty {
			b.Version = 1
			b.Empty2 = true
			b.Data = dwg.AcisData{Blocks: [][]byte{cEncodeSAT(sat.String() + "\n")}}
		}
	}
	if o.Type == 0x26 && x.d.Version.R2007 {
		m.Subclass("AcDb3dSolid")
		x.HandleRef(m, 350, &b.HistoryID, 4)
	}
}

// cEncodeSAT applies the version 1 SAT character mapping (0x9F-c for 0x20..0x7E), its own inverse.
func cEncodeSAT(s string) []byte {
	out := []byte(s)
	for i, c := range out {
		if c >= 0x20 && c <= 0x7E {
			out[i] = 0x9F - c
		}
	}
	return out
}

// cRaster binds AcDbRasterImage (IMAGE, and WIPEOUT before its own subclass).
func cRaster(m *dxf.Mapper, x *ctx, b *dwg.Image) {
	m.Subclass("AcDbRasterImage")
	m.Int32(90, &b.ClassVersion)
	m.Point3(10, &b.Pt0)
	m.Point3(11, &b.UVec)
	m.Point3(12, &b.VVec)
	m.Point2(13, &b.Size)
	x.HandleRef(m, 340, &b.ImageDef, 5)
	m.Int16(70, &b.DisplayProps)
	m.Bool(280, &b.Clipping)
	m.Byte(281, &b.Brightness)
	m.Byte(282, &b.Contrast)
	m.Byte(283, &b.Fade)
	x.HandleRefOpt(m, 360, &b.ImageDefReactor, 3)
	m.Int16(71, &b.ClipBndType)
	n := m.Count(91, len(b.ClipVerts))
	if m.Reading() {
		b.ClipVerts = make([][2]float64, n)
	}
	for i := range b.ClipVerts {
		m.Point2(14, &b.ClipVerts[i])
	}
	if x.d.Version.R2010 {
		m.Bool(290, &b.ClipMode)
	}
}

// proxy handle group by reference code: soft pointer 330, hard pointer 340, soft owner 350, hard owner 360.
var cProxyRefGroup = map[byte]int{4: 330, 5: 340, 2: 350, 3: 360}

// cProxyEntity binds AcDbProxyEntity. Proxy graphics come from the common entity data.
func cProxyEntity(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.ProxyEntity) {
	m.Subclass("AcDbProxyEntity")
	id := int32(498)
	m.Int32(90, &id)
	m.Int32(91, &b.ClassID)
	g := o.Ent.Graphic
	gn := int32(len(g))
	m.Int32(92, &gn)
	if gn > 0 {
		m.Binary(310, &g)
	}
	nb := int32(b.NDataBits)
	m.Int32(93, &nb)
	data := b.DataBits
	if nb > 0 {
		m.Binary(310, &data)
	}
	if !m.Reading() {
		for i := range b.Handles {
			x.HandleRef(m, cProxyRefGroup[b.Handles[i].Code], &b.Handles[i], b.Handles[i].Code)
		}
	} else {
		for len(m.Rest()) > 0 {
			gr := m.Rest()[0]
			var code byte
			for k, v := range cProxyRefGroup {
				if v == gr.Code {
					code = k
				}
			}
			if code == 0 {
				break
			}
			m.Seek(m.Pos() + 1)
			b.Handles = append(b.Handles, dwg.Ref{Code: code, H: gr.Handle()})
		}
	}
	end := int32(0)
	m.Int32(94, &end)
	if x.d.Version.R2018 {
		m.Int32(71, &b.Version)
		m.Int32(97, &b.MaintVersion)
	} else {
		m.Int32(95, &b.DrawingFmt)
	}
	m.Bool(70, &b.OrigDXF)
	if m.Reading() {
		o.Ent.Graphic, o.Ent.HasGraphic = g, len(g) > 0
		b.DataBits, b.NDataBits = data, int(nb)
	}
}
