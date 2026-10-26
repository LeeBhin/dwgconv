package dwg

import "math"

// HeaderDefaults returns an R2018 header with metric (acadiso-like) defaults, for writing a DWG
// without an original. The caller must still set every handle reference to the real objects:
// HANDSEED, CLAYER, TEXTSTYLE, CELTYPE, CMATERIAL, DIMSTYLE, CMLSTYLE, DIMTXSTY, the nine table
// control objects, the dictionaries (group, mlinestyle, named objects, layouts, plot settings,
// plot styles, materials, colors, visual styles), *PAPER_SPACE/*MODEL_SPACE block records,
// BYLAYER/BYBLOCK/CONTINUOUS linetypes and INTERFEREOBJVS/INTERFEREVPVS. Unused references stay
// null. All references use code 5 (hard pointer) except the controls and the named object
// dictionary (code 3, hard owner); HANDSEED uses code 0. Use SetHeaderRef to set them.
func HeaderDefaults() *Header {
	h := &Header{HasSizeHi: true}
	v := &h.Vars
	v.Unknown1, v.Unknown2, v.Unknown3, v.Unknown4 = 412148564080.0, 1, 1, 1
	v.UnknownLong1 = 24
	v.DIMASO, v.DIMSHO = true, true
	v.REGENMODE, v.FILLMODE, v.PSLTSCALE = true, true, true
	v.USRTIMER, v.WORLDVIEW, v.TILEMODE, v.VISRETAIN = true, true, true, true
	v.PROXYGRAPHICS, v.TREEDEPTH = 1, 3020
	v.LUNITS, v.LUPREC, v.AUNITS, v.AUPREC = 2, 4, 0, 0
	v.ATTMODE = 1
	v.SPLINESEGS, v.SURFU, v.SURFV, v.SURFTYPE, v.SURFTAB1, v.SURFTAB2 = 8, 6, 6, 6, 6, 6
	v.SPLINETYPE, v.SHADEDGE, v.SHADEDIF = 6, 3, 70
	v.MAXACTVP, v.ISOLINES, v.TEXTQLTY = 64, 4, 50
	v.LTSCALE, v.TEXTSIZE, v.TRACEWID, v.SKETCHINC = 1, 2.5, 1, 1
	v.FACETRES, v.CMLSCALE, v.CELTSCALE = 0.5, 20, 1
	v.MENU = "acad"
	v.CECOLOR = ACIColor(256)
	v.PLIMMAX = [2]float64{420, 297}
	v.LIMMAX = [2]float64{420, 297}
	big := 1e20
	v.EXTMIN, v.EXTMAX = [3]float64{big, big, big}, [3]float64{-big, -big, -big}
	v.PEXTMIN, v.PEXTMAX = v.EXTMIN, v.EXTMAX
	v.UCSXDIR, v.UCSYDIR = [3]float64{1, 0, 0}, [3]float64{0, 1, 0}
	v.PUCSXDIR, v.PUCSYDIR = v.UCSXDIR, v.UCSYDIR
	// dimensions (ISO-25)
	v.DIMSCALE, v.DIMASZ, v.DIMEXO, v.DIMDLI, v.DIMEXE = 1, 2.5, 0.625, 3.75, 1.25
	v.DIMTXT, v.DIMCEN, v.DIMGAP = 2.5, 2.5, 0.625
	v.DIMALTF, v.DIMLFAC, v.DIMTFAC = 0.0393700787401575, 1, 1
	v.DIMTAD, v.DIMZIN, v.DIMDSEP, v.DIMATFIT = 1, 8, ',', 3
	v.DIMALTD, v.DIMALTU, v.DIMALTTD = 3, 2, 3
	v.DIMDEC, v.DIMTDEC, v.DIMLUNIT = 2, 2, 2
	v.DIMTOFL = true
	v.DIMLWD, v.DIMLWE = -2, -2
	v.DIMCLRD, v.DIMCLRE, v.DIMCLRT = ACIColor(0), ACIColor(0), ACIColor(0)
	v.DIMTZIN, v.DIMALTZ, v.DIMALTTZ, v.DIMAZIN = 8, 0, 0, 0
	v.TSTACKALIGN, v.TSTACKSIZE = 1, 70
	// Flags: CELWEIGHT -1 (index 29), LWDISPLAY off, EXTNAMES, PSTYLEMODE (color dependent)
	v.Flags = 29 | 0x200 | 0x800 | 0x2000
	v.INSUNITS = 4
	v.SORTENTS, v.INDEXCTL, v.HIDETEXT, v.XCLIPFRAME, v.DIMASSOC = 127, 0, 1, 2, 2
	v.OBSCOLOR, v.INTERSECTIONCOLOR = 257, 257
	SetR2007HeaderDefaults(h)
	v.DIMARCSYM = 0
	v.DGNFRAME = 0
	v.INTERFERECOLOR = ACIColor(1)
	v.UnknownDouble1 = 1
	v.UnknownLong10 = 10
	for _, name := range headerRefFields {
		SetHeaderRef(h, name, 0)
	}
	return h
}

// SetR2007HeaderDefaults fills the variables an R2004 file does not store, with the values the
// ODA converter reports for such files. Needed when an R2004 header is written as R2007+ or DXF.
// CMATERIAL is left null: the converter sets it to the MATERIAL named "ByLayer" in the
// materials dictionary, which needs the object table.
func SetR2007HeaderDefaults(h *Header) {
	v := &h.Vars
	v.DIMFXL, v.DIMJOGANG, v.DIMARCSYM = 1, math.Pi/4, 1
	v.DIMTFILLCLR = ACIColor(0)
	v.LENSLENGTH, v.STEPSPERSEC, v.STEPSIZE, v.DWFPREC3D = 50, 2, 6, 2
	v.PSOLWIDTH, v.PSOLHEIGHT = 0.25, 4
	v.LOFTANG1, v.LOFTANG2, v.LOFTPARAM, v.LOFTNORMALS = math.Pi/2, math.Pi/2, 7, 1
	v.LATITUDE, v.LONGITUDE, v.TIMEZONE = 37.795, -122.394, -8000
	v.LIGHTGLYPHDISPLAY, v.TILEMODELIGHTSYNCH, v.SHOWHIST = 1, 1, 1
	v.DWFFRAME, v.DGNFRAME = 2, 2
	v.REALWORLDSCALE = true
	v.INTERFERECOLOR = ACIColor(256)
	for _, name := range []string{"CMATERIAL", "DIMLTYPE", "DIMLTEX1", "DIMLTEX2", "DictVisualStyle",
		"UnknownHandle1", "INTERFEREOBJVS", "INTERFEREVPVS", "DRAGVS"} {
		r := headerRef(h, name)
		if r.Code == 0 {
			r.Code = 5
		}
	}
}

// headerRefFields lists the handle fields in stream order (excluding HANDSEED and CPSNID).
var headerRefFields = []string{"CLAYER", "TEXTSTYLE", "CELTYPE", "CMATERIAL", "DIMSTYLE", "CMLSTYLE",
	"PUCSNAME", "PUCSORTHOREF", "PUCSBASE", "UCSNAME", "UCSORTHOREF", "UCSBASE",
	"DIMTXSTY", "DIMLDRBLK", "DIMBLK", "DIMBLK1", "DIMBLK2", "DIMLTYPE", "DIMLTEX1", "DIMLTEX2",
	"BlockControl", "LayerControl", "StyleControl", "LtypeControl", "ViewControl", "UCSControl",
	"VportControl", "AppIDControl", "DimStyleControl", "DictGroup", "DictMLineStyle", "DictNamedObjects",
	"DictLayouts", "DictPlotSettings", "DictPlotStyles", "DictMaterials", "DictColors", "DictVisualStyle",
	"UnknownHandle1", "BlockRecordPaperSpace", "BlockRecordModelSpace", "LtypeByLayer", "LtypeByBlock",
	"LtypeContinuous", "INTERFEREOBJVS", "INTERFEREVPVS", "DRAGVS"}

// SetHeaderRef sets a handle field with the reference code AutoCAD uses for it.
func SetHeaderRef(h *Header, field string, handle uint64) {
	r := headerRef(h, field)
	if r == nil {
		return
	}
	code := byte(5)
	switch field {
	case "HANDSEED":
		code = 0
	case "BlockControl", "LayerControl", "StyleControl", "LtypeControl", "ViewControl", "UCSControl",
		"VportControl", "AppIDControl", "DimStyleControl", "DictNamedObjects":
		code = 3
	}
	*r = Ref{Code: code, H: handle}
}

func headerRef(h *Header, field string) *Ref {
	v := &h.Vars
	m := map[string]*Ref{
		"HANDSEED": &v.HANDSEED, "CLAYER": &v.CLAYER, "TEXTSTYLE": &v.TEXTSTYLE, "CELTYPE": &v.CELTYPE,
		"CMATERIAL": &v.CMATERIAL, "DIMSTYLE": &v.DIMSTYLE, "CMLSTYLE": &v.CMLSTYLE,
		"PUCSNAME": &v.PUCSNAME, "PUCSORTHOREF": &v.PUCSORTHOREF, "PUCSBASE": &v.PUCSBASE,
		"UCSNAME": &v.UCSNAME, "UCSORTHOREF": &v.UCSORTHOREF, "UCSBASE": &v.UCSBASE,
		"DIMTXSTY": &v.DIMTXSTY, "DIMLDRBLK": &v.DIMLDRBLK, "DIMBLK": &v.DIMBLK, "DIMBLK1": &v.DIMBLK1,
		"DIMBLK2": &v.DIMBLK2, "DIMLTYPE": &v.DIMLTYPE, "DIMLTEX1": &v.DIMLTEX1, "DIMLTEX2": &v.DIMLTEX2,
		"BlockControl": &v.BlockControl, "LayerControl": &v.LayerControl, "StyleControl": &v.StyleControl,
		"LtypeControl": &v.LtypeControl, "ViewControl": &v.ViewControl, "UCSControl": &v.UCSControl,
		"VportControl": &v.VportControl, "AppIDControl": &v.AppIDControl, "DimStyleControl": &v.DimStyleControl,
		"DictGroup": &v.DictGroup, "DictMLineStyle": &v.DictMLineStyle, "DictNamedObjects": &v.DictNamedObjects,
		"DictLayouts": &v.DictLayouts, "DictPlotSettings": &v.DictPlotSettings, "DictPlotStyles": &v.DictPlotStyles,
		"DictMaterials": &v.DictMaterials, "DictColors": &v.DictColors, "DictVisualStyle": &v.DictVisualStyle,
		"UnknownHandle1": &v.UnknownHandle1, "CPSNID": &v.CPSNID,
		"BlockRecordPaperSpace": &v.BlockRecordPaperSpace, "BlockRecordModelSpace": &v.BlockRecordModelSpace,
		"LtypeByLayer": &v.LtypeByLayer, "LtypeByBlock": &v.LtypeByBlock, "LtypeContinuous": &v.LtypeContinuous,
		"INTERFEREOBJVS": &v.INTERFEREOBJVS, "INTERFEREVPVS": &v.INTERFEREVPVS, "DRAGVS": &v.DRAGVS,
	}
	return m[field]
}

// CopyHeaderRefs copies every handle reference (and HANDSEED, CEPSNTYPE/CPSNID) from src to dst.
func CopyHeaderRefs(dst, src *Header) {
	for _, name := range append([]string{"HANDSEED", "CPSNID"}, headerRefFields...) {
		*headerRef(dst, name) = *headerRef(src, name)
	}
	dst.Vars.CEPSNTYPE = src.Vars.CEPSNTYPE
}
