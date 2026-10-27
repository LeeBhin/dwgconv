package conv

import (
	"encoding/binary"
	"os"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"dwgconv/internal/container"
	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// Versions FromDXF can write, by the names users know.
var outVersions = map[string]string{"R2004": "AC1018", "R2007": "AC1021", "R2010": "AC1024", "R2013": "AC1027", "R2018": "AC1032"}

// DefaultOutVersion is the DWG version written when none is given (as the BricsCAD export did).
const DefaultOutVersion = "R2018"

// fileMaint is the maintenance release byte (file offset 0x0B) written per version, as the
// AutoCAD versions that introduced each format write it.
var fileMaint = map[string]byte{"AC1018": 104, "AC1021": 50, "AC1024": 6, "AC1027": 125, "AC1032": 0}

// rootDictEntries maps named-object dictionary entries to header handle fields.
var rootDictEntries = map[string]string{
	"ACAD_GROUP": "DictGroup", "ACAD_MLINESTYLE": "DictMLineStyle", "ACAD_LAYOUT": "DictLayouts",
	"ACAD_PLOTSETTINGS": "DictPlotSettings", "ACAD_PLOTSTYLENAME": "DictPlotStyles",
	"ACAD_MATERIAL": "DictMaterials", "ACAD_COLOR": "DictColors", "ACAD_VISUALSTYLE": "DictVisualStyle",
}

// controlField maps a DXF table name to the header field holding its control object.
var controlField = map[string]string{
	"VPORT": "VportControl", "LTYPE": "LtypeControl", "LAYER": "LayerControl", "STYLE": "StyleControl",
	"VIEW": "ViewControl", "UCS": "UCSControl", "APPID": "AppIDControl", "DIMSTYLE": "DimStyleControl",
	"BLOCK_RECORD": "BlockControl",
}

// FromDXF builds a DWG of the given version (R2004, R2010, R2013 or R2018) from a parsed DXF.
func FromDXF(f *dxf.File, version string) ([]byte, error) {
	ver, ok := outVersions[strings.ToUpper(version)]
	if !ok {
		return nil, fmt.Errorf("unsupported output version %q (R2004, R2007, R2010, R2013, R2018)", version)
	}
	cp := dwg.CodepageNumber(headerString(f, "$DWGCODEPAGE"))
	d := dwg.NewDocument(ver, cp)
	x := &ctx{d: d, names: map[uint64]string{}, byName: map[string]map[string]uint64{}, outVersion: ver}

	x.importClasses(f.Section("CLASSES"))
	x.registerNames(f)
	x.hdr = x.importHeader(f.Section("HEADER"))
	x.measurement = 1
	if m := headerString(f, "$MEASUREMENT"); m != "" {
		x.measurement = int16((dxf.Group{Value: m}).Int())
	}
	x.ms = x.byName["BLOCK_RECORD"]["*MODEL_SPACE"]
	x.ps = x.byName["BLOCK_RECORD"]["*PAPER_SPACE"]

	var warn []string
	skip := map[string]bool{}
	for _, t := range strings.Split(os.Getenv("DWGCONV_SKIP"), ",") {
		skip[t] = t != ""
	}
	add := func(r *dxf.Record, entity bool) *dwg.Object {
		if skip[r.Type] { // diagnostics: leave record types out
			return nil
		}
		o := x.importRecord(r, entity)
		if o == nil {
			warn = append(warn, r.Type)
			return nil
		}
		// types with a fixed number keep it (AutoCAD writes LAYOUT, LWPOLYLINE, XRECORD... that way);
		// the class number is used only for class-only types
		if o.Type == 0 {
			if t, ok := x.classType(r.Type); ok {
				o.Type = t
			}
		}
		if o.Type == 0 {
			warn = append(warn, r.Type+"(type)")
			return nil
		}
		d.Add(o)
		return o
	}

	// TABLES: control objects and their records
	hv := reflect.ValueOf(&x.hdr.Vars).Elem()
	if s := f.Section("TABLES"); s != nil {
		var ctl *dwg.Object
		var table string
		for _, r := range s.Records {
			switch r.Type {
			case "TABLE":
				table, _ = r.Value(2)
				ctl = add(r, false)
				if ctl != nil {
					if tc, ok := ctl.Body.(*dwg.TableControl); ok {
						tc.SetKind(ctl.Type) // which control (decides the special entries)
					}
					if fld, ok := controlField[table]; ok {
						hv.FieldByName(fld).Set(reflect.ValueOf(dwg.Ref{Code: 3, H: ctl.Handle}))
					}
				}
			case "ENDTAB":
				ctl = nil
			default:
				o := add(r, false)
				if o == nil || ctl == nil {
					continue
				}
				tc, ok := ctl.Body.(*dwg.TableControl)
				if !ok {
					continue
				}
				name, _ := r.Value(2)
				if table == "BLOCK_RECORD" {
					setNameField(o, dwgBlockName(name))
				}
				// special records: *MODEL_SPACE, *PAPER_SPACE and BYBLOCK, BYLAYER, in that order (as AutoCAD
				// writes them; AUDIT reports "Special Name Incorrect" otherwise)
				slot := -1
				switch strings.ToUpper(name) {
				case "*MODEL_SPACE", "BYBLOCK":
					slot = 0
				case "*PAPER_SPACE", "BYLAYER":
					slot = 1
				}
				switch {
				case slot >= 0 && slot < len(tc.Special) && tc.Special[slot].H == 0:
					tc.Special[slot] = dwg.Ref{Code: 3, H: o.Handle}
				default:
					tc.Entries = append(tc.Entries, dwg.Ref{Code: 2, H: o.Handle})
				}
			}
		}
	}

	// BLOCKS and ENTITIES: entities are attached to their owner (block record or parent entity)
	for _, sec := range []string{"BLOCKS", "ENTITIES"} {
		s := f.Section(sec)
		if s == nil {
			continue
		}
		for _, r := range s.Records {
			if r.Type == "ENDSEC" {
				continue
			}
			o := add(r, true)
			if o == nil {
				continue
			}
			x.attachEntity(o, r.Type)
		}
	}

	// OBJECTS
	if s := f.Section("OBJECTS"); s != nil {
		for i, r := range s.Records {
			o := add(r, false)
			if o == nil {
				continue
			}
			if i == 0 {
				x.hdr.Vars.DictNamedObjects = dwg.Ref{Code: 3, H: o.Handle}
				if dict, ok := o.Body.(*dwg.Dictionary); ok {
					for k, name := range dict.Names {
						if fld, ok := rootDictEntries[name]; ok {
							hv.FieldByName(fld).Set(reflect.ValueOf(dwg.Ref{Code: 5, H: dict.Items[k].H}))
						}
					}
				}
			}
		}
	}
	x.finishBlockHeaders()
	fixNullRefs(reflect.ValueOf(&x.hdr.Vars)) // header handles: no code-0 null references either
	// dictionary entries pointing to objects the DXF did not carry are dropped (AUDIT erases them)
	for _, o := range d.Objects {
		if dict, ok := o.Body.(*dwg.Dictionary); ok {
			names, items := dict.Names[:0:0], dict.Items[:0:0]
			for i, it := range dict.Items {
				if d.ByHandle[it.H] != nil {
					names, items = append(names, dict.Names[i]), append(items, it)
				}
			}
			dict.Names, dict.Items = names, items
		}
	}
	for _, o := range d.Objects {
		if bh, ok := o.Body.(*dwg.BlockHeader); ok {
			bh.InsertCounts = bh.InsertCounts[:0] // one non-zero byte per insert (as AutoCAD writes)
			for range bh.Inserts {
				bh.InsertCounts = append(bh.InsertCounts, 1)
			}
			if bh.Layout.H == 0 {
				bh.Layout.Code = 5
			}
		}
	}
	x.finishHeaderRefs()

	if len(warn) > 0 {
		counts := map[string]int{}
		for _, w := range warn {
			counts[w]++
		}
		var parts []string
		for k, n := range counts {
			parts = append(parts, fmt.Sprintf("%s×%d", k, n))
		}
		sort.Strings(parts)
		lastWarnings = parts
	}
	return x.encodeFile()
}

// lastWarnings lists record types FromDXF skipped (no mapping), for diagnostics.
var lastWarnings []string

// Warnings returns the record types skipped by the last FromDXF call.
func Warnings() []string { return lastWarnings }

func headerString(f *dxf.File, name string) string {
	if s := f.Section("HEADER"); s != nil {
		for _, r := range s.Records {
			if r.Type == name && len(r.Groups) > 0 {
				return r.Groups[0].Value
			}
		}
	}
	return ""
}

func (x *ctx) importClasses(s *dxf.Section) {
	ct := x.d.Classes
	if s == nil {
		return
	}
	for i, r := range s.Records {
		m := dxf.NewReadMapper(r)
		k := dwg.Class{Num: int16(500 + i), ItemClass: 0x1F3}
		var entity bool
		m.Str(1, &k.DXFName)
		m.Str(2, &k.CppName)
		m.Str(3, &k.AppName)
		var flags, count int32
		m.Int32(90, &flags)
		m.Int32(91, &count)
		m.Bool(280, &k.Zombie)
		m.Bool(281, &entity)
		k.ProxyFlags, k.Count = int16(flags), count
		if entity {
			k.ItemClass = 0x1F2
		}
		ct.Classes = append(ct.Classes, k)
	}
	ct.MaxNum = int16(499 + len(ct.Classes))
}

// classType returns the class-based type number for a DXF record name, if the drawing defines it.
func (x *ctx) classType(name string) (int, bool) {
	for i, k := range x.d.Classes.Classes {
		if k.DXFName == name {
			return 500 + i, true
		}
	}
	return 0, false
}

// registerNames records table entry and MLINESTYLE names before any reference is resolved.
func (x *ctx) registerNames(f *dxf.File) {
	if s := f.Section("TABLES"); s != nil {
		table := ""
		for _, r := range s.Records {
			switch r.Type {
			case "TABLE":
				table, _ = r.Value(2)
			case "ENDTAB":
			default:
				h := r.Handle(5)
				if h == 0 {
					h = r.Handle(105)
				}
				if name, ok := r.Value(2); ok && h != 0 {
					x.addName(table, name, h)
				}
			}
		}
	}
	if s := f.Section("OBJECTS"); s != nil {
		for _, r := range s.Records {
			if r.Type == "MLINESTYLE" {
				if name, ok := r.Value(2); ok {
					x.addName("MLINESTYLE", name, r.Handle(5))
				}
			}
		}
	}
}

// attachEntity links an imported entity into its owner's lists.
func (x *ctx) attachEntity(o *dwg.Object, typ string) {
	owner := x.d.ByHandle[o.Owner.H]
	if owner == nil {
		return
	}
	if bh, ok := owner.Body.(*dwg.BlockHeader); ok {
		switch typ {
		case "BLOCK":
			bh.Block = dwg.Ref{Code: 3, H: o.Handle}
			if n, ok := entryName(o); ok {
				setNameField(o, dwgBlockName(n))
			}
		case "ENDBLK":
			bh.EndBlk = dwg.Ref{Code: 3, H: o.Handle}
		default:
			bh.Entities = append(bh.Entities, dwg.Ref{Code: 3, H: o.Handle})
		}
		return
	}
	if owner.Body == nil {
		return
	}
	v := reflect.ValueOf(owner.Body).Elem()
	if typ == "SEQEND" {
		if f := v.FieldByName("SeqEnd"); f.IsValid() {
			f.Set(reflect.ValueOf(dwg.Ref{Code: 3, H: o.Handle}))
		}
		return
	}
	for _, name := range []string{"Attribs", "Vertices"} {
		if f := v.FieldByName(name); f.IsValid() {
			f.Set(reflect.Append(f, reflect.ValueOf(dwg.Ref{Code: 3, H: o.Handle})))
			return
		}
	}
}

// finishBlockHeaders fills the INSERT back references of each block record.
func (x *ctx) finishBlockHeaders() {
	for _, o := range x.d.Objects {
		if !o.IsEntity || o.Body == nil {
			continue
		}
		f := reflect.ValueOf(o.Body).Elem().FieldByName("BlockHdr")
		if !f.IsValid() {
			continue
		}
		if b := x.d.ByHandle[f.Interface().(dwg.Ref).H]; b != nil {
			if bh, ok := b.Body.(*dwg.BlockHeader); ok && !hasRef(bh.Inserts, o.Handle) {
				bh.Inserts = append(bh.Inserts, dwg.Ref{Code: 4, H: o.Handle})
			}
		}
	}
}

func hasRef(list []dwg.Ref, h uint64) bool {
	for _, r := range list {
		if r.H == h {
			return true
		}
	}
	return false
}

// finishHeaderRefs sets header references DXF does not carry directly.
func (x *ctx) finishHeaderRefs() {
	v := &x.hdr.Vars
	v.BlockRecordModelSpace = dwg.Ref{Code: 5, H: x.ms}
	v.BlockRecordPaperSpace = dwg.Ref{Code: 5, H: x.ps}
	lt := x.byName["LTYPE"]
	v.LtypeByLayer = dwg.Ref{Code: 5, H: lt["BYLAYER"]}
	v.LtypeByBlock = dwg.Ref{Code: 5, H: lt["BYBLOCK"]}
	v.LtypeContinuous = dwg.Ref{Code: 5, H: lt["CONTINUOUS"]}
	var max uint64
	for _, o := range x.d.Objects {
		if o.Handle > max {
			max = o.Handle
		}
	}
	if v.HANDSEED.H <= max {
		v.HANDSEED.H = max + 1
	}
	v.HANDSEED.Code = 0 // the handle seed is a plain value, not a reference (AutoCAD misreads the header otherwise)
}

// filePrefix builds the file header bytes 0x00-0x7F (§4.1) for a version; page addresses are
// filled in by the container writer. The writing application is recorded as that version.
func filePrefix(version string, maint byte) [0x80]byte {
	var p [0x80]byte
	copy(p[:], version)
	p[0x0B] = maint
	p[0x0C] = 3
	app := byte(dwg.VersionCode(version))
	p[0x11], p[0x12] = app, maint // application version / maintenance
	p[0x16], p[0x17] = app, maint // repeated, as AutoCAD writes it
	binary.LittleEndian.PutUint32(p[0x28:], 0x80)
	if version == "AC1032" {
		copy(p[0x34:], []byte{0x02, 0x02, 0, 0, 0xF7, 0, 0, 0, 0xC5, 0, 0, 0, 0xC5, 0, 0, 0})
	}
	return p
}

func (x *ctx) encodeFile() ([]byte, error) {
	d := x.d
	v := d.Version
	sort.SliceStable(d.Objects, func(i, j int) bool { return d.Objects[i].Handle < d.Objects[j].Handle })
	objs, handles := d.EncodeObjects()

	now := julianNow()
	hdr := x.hdr
	maint := fileMaint[x.outVersion]
	// R2018 always has the extra size RL; R2010/R2013 only with maintenance > 3 (§9)
	hdr.HasSizeHi = v.R2018 || (v.R2010 && maint > 3)
	if hdr.Vars.TDCREATE.Day == 0 {
		hdr.Vars.TDCREATE = now
	}
	hdr.Vars.TDUPDATE = now

	f := &container.File{Version: x.outVersion, Prefix: filePrefix(x.outVersion, maint)}
	binary.LittleEndian.PutUint16(f.Prefix[0x13:], uint16(d.Codepage))
	copy(f.Header[:], "AcFssFcAJMB\x00")
	binary.LittleEndian.PutUint32(f.Header[0x10:], 0x6C)
	binary.LittleEndian.PutUint32(f.Header[0x14:], 4)
	binary.LittleEndian.PutUint32(f.Header[0x24:], 1)
	binary.LittleEndian.PutUint32(f.Header[0x44:], 0x20)
	binary.LittleEndian.PutUint32(f.Header[0x48:], 0x80)
	binary.LittleEndian.PutUint32(f.Header[0x4C:], 0x40)

	type sec struct {
		name string
		max  int32
		comp int32
		data []byte
	}
	list := []sec{
		{"", 0x7400, 2, nil},
		{"AcDb:AppInfo", 0x300, 1, dwg.DefaultAppInfo("DELTAFlow", "", "DELTAFlow").Encode(d.Text)},
		{"AcDb:Preview", 0x400, 1, dwg.DefaultPreview().Encode()},
		{"AcDb:SummaryInfo", 0x100, 1, dwg.DefaultSummaryInfo(hdr.Vars.TDCREATE, now).Encode(v, d.Text)},
		{"AcDb:RevHistory", 0x7400, 2, dwg.DefaultRevHistory().Encode()},
		{"AcDb:AcDbObjects", 0x7400, 2, objs},
		{"AcDb:ObjFreeSpace", 0x7400, 2, dwg.DefaultObjFreeSpace(uint64(len(d.Objects)), now).Encode(v)},
		{"AcDb:Template", 0x7400, 2, dwg.DefaultTemplate(v, x.measurement).Encode(v)},
		{"AcDb:Handles", 0x7400, 2, handles},
		{"AcDb:Classes", 0x7400, 2, d.Classes.Encode(v, d.Text)},
		{"AcDb:AuxHeader", 0x7400, 2, dwg.DefaultAuxHeader(x.outVersion, int32(maint), hdr.Vars.TDCREATE, now, hdr.Vars.HANDSEED.H).Encode(v)},
		{"AcDb:Header", 0x7400, 2, hdr.Encode(v, d.Text)},
	}
	for i, s := range list {
		id := int32(0)
		if i > 0 {
			id = int32(len(list) - i)
		}
		f.Sections = append(f.Sections, &container.Section{Name: s.name, ID: id, MaxSize: s.max,
			Compressed: s.comp, Unknown: 1, Data: s.data})
	}
	return f.Write(), nil
}

func julianNow() dwg.JulianDate {
	t := time.Now().UTC()
	const unixEpochJD = 2440587.5
	jd := float64(t.UnixMilli())/86400000 + unixEpochJD
	day := int32(jd)
	return dwg.JulianDate{Day: day, Ms: int32((jd - float64(day)) * 86400000)}
}

// setNameField sets the exported Name field of a body, if it has one.
func setNameField(o *dwg.Object, name string) {
	if o.Body == nil {
		return
	}
	if f := reflect.ValueOf(o.Body).Elem().FieldByName("Name"); f.IsValid() && f.Kind() == reflect.String {
		f.SetString(name)
	}
}
