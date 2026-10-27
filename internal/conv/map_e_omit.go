package conv

import (

	"dwgconv/internal/dwg"
)

// Objects ODA leaves out of its DXF output (observed on the sample drawings). ODA folds round-trip
// records back into their owners, drops the AcDs data-store tree, SPATIAL_INDEX and MATERIAL tile
// records, drops dictionary entries whose target is missing, and then omits extension dictionaries
// (dictionaries owned by a non-dictionary object) that are left without entries; originally empty
// extension dictionaries are omitted only for some owner types.

// eRoundTripNames are dictionary entry names whose subtree ODA does not write.
var eRoundTripNames = map[string]bool{
	"ACAD_XREC_ROUNDTRIP":    true,
	"ACAD_MTEXT_2008_RT":     true,
	"ACAD_LAYOUTSELFREF":     true,
	"ACAD_2007HEADERVARS_RT": true,
	"AcDsDecomposeData":      true,
	"ACDB_RECOMPOSE_DATA":    true, // ODA writes a regenerated record under a new handle
	// MATERIAL texture tile records
	"BUMPTILE": true, "DIFFUSETILE": true, "OPACITYTILE": true,
	"REFLECTIONTILE": true, "REFRACTIONTILE": true, "SPECULARTILE": true,
}

// eEmptyXDicDropped are the owner types whose originally empty extension dictionary ODA omits
// (empty ones on LAYOUT, VPORT_CONTROL, INSERT... are kept).
var eEmptyXDicDropped = map[string]bool{"LAYER": true, "BLOCK_HEADER": true, "TABLESTYLE": true}

// eEmptiedXDicKept are the owner types whose extension dictionary ODA keeps (empty) after
// dropping its round-trip entries.
var eEmptiedXDicKept = map[string]bool{"ACDBSECTIONVIEWSTYLE": true, "ACDBDETAILVIEWSTYLE": true}

// eODAOmits reports whether ODA leaves the object with handle h out of the DXF. The set is computed
// once per conversion and kept in the ctx (not globally), so concurrent conversions stay separate.
func eODAOmits(x *ctx, h uint64) bool {
	if x.omitted == nil {
		x.omitted = eComputeOmitted(x.d)
	}
	if x.d.ByHandle[h] == nil {
		return true
	}
	return x.omitted[h]
}

func eDict(o *dwg.Object) *dwg.Dictionary {
	switch b := o.Body.(type) {
	case *dwg.Dictionary:
		return b
	case *dwg.DictionaryWithDefault:
		return &b.Dictionary
	}
	return nil
}

func eComputeOmitted(d *dwg.Document) map[uint64]bool {
	set := map[uint64]bool{}
	var subtree func(h uint64)
	subtree = func(h uint64) {
		o := d.ByHandle[h]
		if o == nil || set[h] {
			return
		}
		set[h] = true
		if dd := eDict(o); dd != nil {
			for _, it := range dd.Items {
				if t := d.ByHandle[it.H]; t != nil && t.Owner.H == h {
					subtree(it.H)
				}
			}
		}
	}
	isDict := func(o *dwg.Object) bool { return o != nil && eDict(o) != nil }
	for _, o := range d.Objects {
		name := o.TypeName(d.Classes.Classes)
		if name == "SPATIAL_INDEX" {
			set[o.Handle] = true
		}
		if dd := eDict(o); dd != nil {
			for i, it := range dd.Items {
				// R2004: the 2008 cell style map is round-trip data of the pre-2008 TABLESTYLE.
				rt := eRoundTripNames[dd.Names[i]] ||
					(!d.Version.R2007 && dd.Names[i] == "ACAD_ROUNDTRIP_2008_TABLESTYLE_CELLSTYLEMAP")
				if rt {
					subtree(it.H)
				}
			}
		}
		if (name == "XRECORD" || isDict(o)) && o.Owner.H != 0 && d.ByHandle[o.Owner.H] == nil {
			subtree(o.Handle)
		}
	}
	// extension dictionaries left without entries
	for changed := true; changed; {
		changed = false
		for _, o := range d.Objects {
			dd := eDict(o)
			if dd == nil || set[o.Handle] {
				continue
			}
			ow := d.ByHandle[o.Owner.H]
			if ow == nil || isDict(ow) {
				continue
			}
			on := ow.TypeName(d.Classes.Classes)
			if len(dd.Items) == 0 && !eEmptyXDicDropped[on] || eEmptiedXDicKept[on] {
				continue
			}
			live := false
			for _, it := range dd.Items {
				if d.ByHandle[it.H] != nil && !set[it.H] {
					live = true
					break
				}
			}
			if !live {
				set[o.Handle] = true
				changed = true
			}
		}
	}
	return set
}
