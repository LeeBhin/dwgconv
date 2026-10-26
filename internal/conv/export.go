package conv

import (
	"io"
	"reflect"

	"dwgconv/internal/dwg"
	"dwgconv/internal/dxf"
)

// ToDXF writes the drawing as an AC1032 ASCII DXF.
func ToDXF(d *dwg.Document, out io.Writer) error {
	x := newCtx(d)
	hdr, err := dwg.DecodeFileHeader(d.File)
	if err != nil {
		return err
	}
	if !d.Version.R2007 {
		dwg.SetR2007HeaderDefaults(hdr)
	}
	info := headerInfo{codepage: dwg.CodepageName(d.Codepage), measurement: 1}
	if s := d.File.Section("AcDb:SummaryInfo"); s != nil {
		if si, err := dwg.DecodeSummaryInfo(s.Data, d.Version, d.Text); err == nil {
			info.lastSavedBy = si.LastSavedBy
		}
	}
	if s := d.File.Section("AcDb:Template"); s != nil {
		if t, err := dwg.DecodeTemplate(s.Data, d.Version); err == nil {
			info.measurement = t.Measurement
		}
	}
	x.hdr = hdr
	if r := hdr.Vars.BlockRecordModelSpace.H; r != 0 {
		x.ms = r
	}
	if r := hdr.Vars.BlockRecordPaperSpace.H; r != 0 {
		x.ps = r
	}

	w := dxf.NewWriter(out)
	w.UnescapeU = !d.Version.R2007
	x.exportHeader(w, hdr, info)
	x.exportClasses(w)
	x.exportTables(w)
	x.exportBlocks(w)
	x.exportEntities(w)
	x.exportObjects(w)
	w.Str(0, "EOF")
	return w.Flush()
}

func (x *ctx) exportClasses(w *dxf.Writer) {
	w.Str(0, "SECTION")
	w.Str(2, "CLASSES")
	for _, k := range x.d.Classes.Classes {
		w.Str(0, "CLASS")
		w.Str(1, k.DXFName)
		w.Str(2, k.CppName)
		w.Str(3, k.AppName)
		w.Int(90, int64(k.ProxyFlags))
		w.Int(91, int64(k.Count))
		w.Bool(280, k.Zombie)
		w.Bool(281, k.ItemClass == 0x1F2)
	}
	w.Str(0, "ENDSEC")
}

// tableOrder is the TABLES section order with the header control field of each table.
var tableOrder = []string{"VportControl", "LtypeControl", "LayerControl", "StyleControl", "ViewControl",
	"UCSControl", "AppIDControl", "DimStyleControl", "BlockControl"}

func (x *ctx) exportTables(w *dxf.Writer) {
	w.Str(0, "SECTION")
	w.Str(2, "TABLES")
	hv := reflect.ValueOf(&x.hdr.Vars).Elem()
	for _, f := range tableOrder {
		ref := hv.FieldByName(f).Interface().(dwg.Ref)
		ctl := x.d.ByHandle[ref.H]
		if ctl == nil || !x.exportRecord(w, ctl) {
			continue
		}
		for _, h := range controlEntries(ctl) {
			if o := x.d.ByHandle[h.H]; o != nil {
				x.exportRecord(w, o)
			}
		}
		w.Str(0, "ENDTAB")
	}
	w.Str(0, "ENDSEC")
}

// controlEntries returns a table control's records, the special ones (*Model_Space, ByLayer...) first.
func controlEntries(ctl *dwg.Object) []dwg.Ref {
	tc, ok := ctl.Body.(*dwg.TableControl)
	if !ok {
		return nil
	}
	if ctl.Type == 0x30 { // BLOCK_CONTROL: *Model_Space, *Paper_Space, then the rest
		return append(append([]dwg.Ref(nil), tc.Special...), tc.Entries...)
	}
	return tc.Entries
}

func (x *ctx) blockHeaders() []*dwg.Object {
	ctl := x.d.ByHandle[x.hdr.Vars.BlockControl.H]
	if ctl == nil {
		return nil
	}
	var out []*dwg.Object
	for _, h := range controlEntries(ctl) {
		if o := x.d.ByHandle[h.H]; o != nil {
			if _, ok := o.Body.(*dwg.BlockHeader); ok {
				out = append(out, o)
			}
		}
	}
	return out
}

func (x *ctx) exportBlocks(w *dxf.Writer) {
	w.Str(0, "SECTION")
	w.Str(2, "BLOCKS")
	for _, o := range x.blockHeaders() {
		bh := o.Body.(*dwg.BlockHeader)
		if b := x.d.ByHandle[bh.Block.H]; b != nil {
			x.exportRecord(w, b)
		}
		if o.Handle != x.ms && o.Handle != x.ps {
			x.exportEntityList(w, bh.Entities)
		}
		if e := x.d.ByHandle[bh.EndBlk.H]; e != nil {
			x.exportRecord(w, e)
		}
	}
	w.Str(0, "ENDSEC")
}

func (x *ctx) exportEntities(w *dxf.Writer) {
	w.Str(0, "SECTION")
	w.Str(2, "ENTITIES")
	for _, h := range []uint64{x.ms, x.ps} {
		if o := x.d.ByHandle[h]; o != nil {
			if bh, ok := o.Body.(*dwg.BlockHeader); ok {
				x.exportEntityList(w, bh.Entities)
			}
		}
	}
	w.Str(0, "ENDSEC")
}

// exportEntityList writes entities followed by their owned sub-entities (ATTRIB/VERTEX ... SEQEND).
func (x *ctx) exportEntityList(w *dxf.Writer, refs []dwg.Ref) {
	for _, r := range refs {
		o := x.d.ByHandle[r.H]
		if o == nil || !x.exportRecord(w, o) {
			continue
		}
		sub, seq := ownedSubEntities(o)
		for _, s := range sub {
			if so := x.d.ByHandle[s.H]; so != nil {
				x.exportRecord(w, so)
			}
		}
		if so := x.d.ByHandle[seq.H]; seq.H != 0 && so != nil {
			x.exportRecord(w, so)
		}
	}
}

// ownedSubEntities returns the ATTRIBs or VERTEXes and the SEQEND an entity owns.
func ownedSubEntities(o *dwg.Object) ([]dwg.Ref, dwg.Ref) {
	if o.Body == nil {
		return nil, dwg.Ref{}
	}
	v := reflect.ValueOf(o.Body).Elem()
	var list []dwg.Ref
	for _, name := range []string{"Attribs", "Vertices"} {
		if f := v.FieldByName(name); f.IsValid() {
			list = f.Interface().([]dwg.Ref)
		}
	}
	var seq dwg.Ref
	if f := v.FieldByName("SeqEnd"); f.IsValid() {
		seq = f.Interface().(dwg.Ref)
	}
	return list, seq
}

// exportObjects writes non-graphic objects: the named object dictionary tree first (depth first),
// then everything else in file order.
func (x *ctx) exportObjects(w *dxf.Writer) {
	w.Str(0, "SECTION")
	w.Str(2, "OBJECTS")
	done := map[uint64]bool{}
	var visit func(h uint64)
	visit = func(h uint64) {
		o := x.d.ByHandle[h]
		if o == nil || done[h] || !isObjectsSectionObject(x, o) {
			return
		}
		done[h] = true
		x.exportRecord(w, o)
		if !o.XDicMiss {
			visit(o.XDic.H)
		}
		if dict, ok := o.Body.(*dwg.Dictionary); ok {
			for _, it := range dict.Items {
				visit(it.H)
			}
		}
	}
	visit(x.hdr.Vars.DictNamedObjects.H)
	for _, o := range x.d.Objects {
		visit(o.Handle)
	}
	w.Str(0, "ENDSEC")
}

// isObjectsSectionObject reports whether o belongs in the OBJECTS section.
func isObjectsSectionObject(x *ctx, o *dwg.Object) bool {
	if o.IsEntity {
		return false
	}
	switch o.Body.(type) {
	case *dwg.TableControl, *dwg.BlockHeader:
		return false
	}
	n := o.TypeName(x.d.Classes.Classes)
	_, isTable := tableOf[n]
	return !isTable || n == "MLINESTYLE"
}
