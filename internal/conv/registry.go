package conv

import (
	"reflect"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// mapping binds one DWG body type to its DXF record. fields describes the type-specific
// groups once (via the bidirectional Mapper): it writes them on export and fills b on import.
// Common entity/object groups are handled by the caller.
type mapping struct {
	dxfName string
	goType  reflect.Type
	newBody func() dwg.Body
	fields  func(m *dxf.Mapper, x *ctx, o *dwg.Object, b dwg.Body)
	// dwgType returns the DWG object type for a newly imported object (fixed types); 0 = class-based.
	dwgType int
	// name, when set, picks the DXF record name from the object (e.g. POLYLINE variants share a Go type).
	name func(o *dwg.Object) string
	// match, when set, selects this mapping on import among those sharing a DXF name.
	match func(r *dxf.Record) bool
	// handleCode is the group code of the record handle (5, or 105 for DIMSTYLE).
	handleCode int
}

var (
	byGoType  = map[reflect.Type]*mapping{}
	byDXFName = map[string][]*mapping{}
)

// register adds a mapping. dwgType is the fixed DWG type number, or 0 for class-based types.
func register[T dwg.Body](dxfName string, dwgType int, f func(m *dxf.Mapper, x *ctx, o *dwg.Object, b T)) *mapping {
	var zero T
	t := reflect.TypeOf(zero)
	mp := &mapping{
		dxfName:    dxfName,
		goType:     t,
		dwgType:    dwgType,
		handleCode: 5,
		newBody:    func() dwg.Body { return reflect.New(t.Elem()).Interface().(dwg.Body) },
		fields:     func(m *dxf.Mapper, x *ctx, o *dwg.Object, b dwg.Body) { f(m, x, o, b.(T)) },
	}
	byGoType[t] = mp
	byDXFName[dxfName] = append(byDXFName[dxfName], mp)
	return mp
}

func mappingFor(o *dwg.Object) *mapping {
	if o.Body == nil {
		return nil
	}
	return byGoType[reflect.TypeOf(o.Body)]
}
