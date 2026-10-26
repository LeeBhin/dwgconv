package conv

import (
	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// Objects of classes whose layout is not decoded (RawBody) are written as ACAD_PROXY_OBJECT /
// ACAD_PROXY_ENTITY carrying their original bits, as ODA writes them. On import, a proxy whose
// data is in the output format (71 = AC1032) and whose class is in the CLASSES section is
// restored as the original object.

// proxyHandleGroup maps a handle reference code to its DXF group (§2.13).
func proxyHandleGroup(code byte) int {
	switch code {
	case 2:
		return 350
	case 3:
		return 360
	case 5:
		return 340
	}
	return 330
}

func proxyHandleCode(group int) byte {
	switch group {
	case 350:
		return 2
	case 360:
		return 3
	case 340:
		return 5
	}
	return 4
}

func init() {
	mp := register("ACAD_PROXY_OBJECT", 0, func(m *dxf.Mapper, x *ctx, o *dwg.Object, b *dwg.RawBody) {
		sub, id := "AcDbProxyObject", int32(499)
		if o.IsEntity {
			sub, id = "AcDbProxyEntity", 498
		}
		m.Subclass(sub)
		m.Int32(90, &id)
		cls := int32(o.Type)
		m.Int32(91, &cls)
		ver, maint := int16(x.auxVersion), int32(x.auxMaint)
		m.Int16(71, &ver)
		m.Int32(97, &maint)
		var dxfFormat int16
		m.Int16(70, &dxfFormat)
		if !m.Reading() {
			w := m.Writer()
			if b.StrBits > 0 {
				w.Int(162, int64(b.StrBits))
				w.Binary(311, b.Str)
			}
			w.Int(161, int64(b.DataBits))
			w.Binary(310, b.Data)
			for _, h := range b.Handles {
				w.Handle(proxyHandleGroup(h.Code), h.H)
			}
			w.Int(94, 0)
			return
		}
		o.Type = int(cls)
		var n32 int32
		if m.Has(162) {
			m.Int32(162, &n32)
			b.StrBits = int(n32)
			m.Binary(311, &b.Str)
		}
		m.Int32(161, &n32)
		b.DataBits = int(n32)
		m.Binary(310, &b.Data)
		for len(m.Rest()) > 0 {
			g := m.Rest()[0]
			if g.Code == 94 {
				m.Seek(m.Pos() + 1)
				break
			}
			if g.Code >= 330 && g.Code <= 369 {
				b.Handles = append(b.Handles, dwg.Ref{Code: proxyHandleCode(g.Code / 10 * 10), H: g.Handle()})
			}
			m.Seek(m.Pos() + 1)
		}
	})
	mp.name = func(o *dwg.Object) string {
		if o.IsEntity {
			return "ACAD_PROXY_ENTITY"
		}
		return "ACAD_PROXY_OBJECT"
	}
	mp.match = func(r *dxf.Record) bool {
		v, _ := r.Value(71)
		c, _ := r.Value(91)
		return (dxf.Group{Value: v}).Int() == 33 && (dxf.Group{Value: c}).Int() >= 500
	}
}
