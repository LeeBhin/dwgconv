package conv

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/LeeBhin/dwgconv/internal/dwg"
	"github.com/LeeBhin/dwgconv/internal/dxf"
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

// cAcis binds AcDbModelerGeometry: SAT text (version 1) as 70 then 1 (with 3 continuations);
// binary SAB (version 2) as 290 + GUID with the bytes in the ACDSDATA section (see acds.go).
func cAcis(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.Acis) {
	m.Subclass("AcDbModelerGeometry")
	if !m.Reading() && !b.Empty && b.Version == 2 && len(b.Data.File) > 0 {
		w := m.Writer()
		has := b.GUID != [16]byte{}
		w.Int(290, map[bool]int64{false: 0, true: 1}[has])
		w.Str(2, formatGUID(b.GUID))
		x.asmOut = append(x.asmOut, asmRecord{o.Handle, b.Data.File})
	} else if !m.Reading() {
		ver := int16(1)
		m.Int16(70, &ver)
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
		} else if data := x.asm[o.Handle]; len(data) > 0 {
			b.Empty, b.Version, b.Empty2 = false, 2, true
			b.Data = dwg.AcisData{File: data}
			if x.d.Version.R2013 {
				// R2013+ object: revision GUID from the DXF (a new one when absent) and the two values
				// ODA writes around it; the SAB itself goes to the AcDs section (see encodeFile)
				var gs string
				m.Str(2, &gs)
				if g, ok := parseGUID(gs); ok && g != [16]byte{} {
					b.GUID = g
				} else {
					b.GUID = newGUID()
				}
				b.Unknown, b.Flags2013, b.Tail2013 = true, 0x40|b.GUID[0]&0x0F, 2
			}
		}
		if !b.Empty {
			// AutoCAD always writes the wireframe block (no point, $ISOLINES, empty wire/silhouette
			// lists); with "wireframe present" or "isolines present" false AutoCAD discards the
			// object (eDwgObjectImproperlyRead) and ODA warns.
			b.Wireframe, b.PointPresent, b.IsolinesPresent = true, false, true
			b.NumIsolines = 4
			if x.hdr != nil && x.hdr.Vars.ISOLINES > 0 {
				b.NumIsolines = int32(x.hdr.Vars.ISOLINES)
			}
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

// formatGUID writes the 16 stored bytes (mixed-endian: Data1-3 little-endian) as {8-4-4-4-12}.
func formatGUID(g [16]byte) string {
	return fmt.Sprintf("{%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x}",
		g[3], g[2], g[1], g[0], g[5], g[4], g[7], g[6], g[8], g[9], g[10], g[11], g[12], g[13], g[14], g[15])
}

// parseGUID is the inverse of formatGUID; braces are optional.
func parseGUID(s string) ([16]byte, bool) {
	var g [16]byte
	s = strings.Trim(strings.TrimSpace(s), "{}")
	var hex [16]byte
	n, err := fmt.Sscanf(s, "%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		&hex[0], &hex[1], &hex[2], &hex[3], &hex[4], &hex[5], &hex[6], &hex[7], &hex[8], &hex[9], &hex[10], &hex[11], &hex[12], &hex[13], &hex[14], &hex[15])
	if err != nil || n != 16 {
		return g, false
	}
	g = [16]byte{hex[3], hex[2], hex[1], hex[0], hex[5], hex[4], hex[7], hex[6], hex[8], hex[9], hex[10], hex[11], hex[12], hex[13], hex[14], hex[15]}
	return g, true
}

// newGUID returns a random version-4 GUID in the stored byte order.
func newGUID() [16]byte {
	var g [16]byte
	_, _ = rand.Read(g[:])
	g[7] = g[7]&0x0F | 0x40 // version 4 (Data3 high byte in little-endian order)
	g[8] = g[8]&0x3F | 0x80 // variant
	return g
}
