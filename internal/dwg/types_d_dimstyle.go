package dwg

// DimStyle §20.4.68 (R2000+ layout).
type DimStyle struct {
	dTableEntry
	DIMPOST, DIMAPOST                                         string  // 3, 4
	DIMSCALE, DIMASZ, DIMEXO, DIMDLI, DIMEXE, DIMRND, DIMDLE  float64 // 40-46
	DIMTP, DIMTM                                              float64 // 47, 48
	DIMFXL, DIMJOGANG                                         float64 // 49, 50 (R2007+)
	DIMTFILL                                                  int16   // 69 (R2007+)
	DIMTFILLCLR                                               CmColor // 70 (R2007+)
	DIMTOL, DIMLIM, DIMTIH, DIMTOH, DIMSE1, DIMSE2            bool    // 71-76
	DIMTAD, DIMZIN, DIMAZIN                                   int16   // 77, 78, 79
	DIMARCSYM                                                 int16   // 90 (R2007+)
	DIMTXT, DIMCEN, DIMTSZ, DIMALTF, DIMLFAC, DIMTVP, DIMTFAC float64 // 140-146
	DIMGAP, DIMALTRND                                         float64 // 147, 148
	DIMALT                                                    bool    // 170
	DIMALTD                                                   int16   // 171
	DIMTOFL, DIMSAH, DIMTIX, DIMSOXD                          bool    // 172-175
	DIMCLRD, DIMCLRE, DIMCLRT                                 CmColor // 176-178
	DIMADEC, DIMDEC, DIMTDEC, DIMALTU, DIMALTTD, DIMAUNIT     int16   // 179, 271-275
	DIMFRAC, DIMLUNIT, DIMDSEP, DIMTMOVE, DIMJUST             int16   // 276-280
	DIMSD1, DIMSD2                                            bool    // 281, 282
	DIMTOLJ, DIMTZIN, DIMALTZ, DIMALTTZ                       int16   // 283-286
	DIMUPT                                                    bool    // 288
	DIMATFIT                                                  int16   // 289
	DIMFXLON                                                  bool    // 290 (R2007+)
	DIMTXTDIRECTION                                           bool    // 295 (R2010+)
	DIMALTMZF                                                 float64 // R2010+
	DIMALTMZS                                                 string  // R2010+
	DIMMZF                                                    float64 // R2010+
	DIMMZS                                                    string  // R2010+
	DIMLWD, DIMLWE                                            int16   // 371, 372
	Flag0                                                     bool    // 70 bit 1
	DIMTXSTY, DIMLDRBLK, DIMBLK, DIMBLK1, DIMBLK2             Ref     // 340-344
	DIMLTYPE, DIMLTEX1, DIMLTEX2                              Ref     // 345-347 (R2007+)
}

func (b *DimStyle) data(c *codec, o *Object) {
	b.dData(c)
	c.TV(&b.DIMPOST)
	c.TV(&b.DIMAPOST)
	for _, p := range []*float64{&b.DIMSCALE, &b.DIMASZ, &b.DIMEXO, &b.DIMDLI, &b.DIMEXE, &b.DIMRND, &b.DIMDLE, &b.DIMTP, &b.DIMTM} {
		c.BD(p)
	}
	if c.R2007 {
		c.BD(&b.DIMFXL)
		c.BD(&b.DIMJOGANG)
		c.BS(&b.DIMTFILL)
		c.CMC(&b.DIMTFILLCLR)
	}
	for _, p := range []*bool{&b.DIMTOL, &b.DIMLIM, &b.DIMTIH, &b.DIMTOH, &b.DIMSE1, &b.DIMSE2} {
		c.B(p)
	}
	c.BS(&b.DIMTAD)
	c.BS(&b.DIMZIN)
	c.BS(&b.DIMAZIN)
	if c.R2007 {
		c.BS(&b.DIMARCSYM)
	}
	for _, p := range []*float64{&b.DIMTXT, &b.DIMCEN, &b.DIMTSZ, &b.DIMALTF, &b.DIMLFAC, &b.DIMTVP, &b.DIMTFAC, &b.DIMGAP, &b.DIMALTRND} {
		c.BD(p)
	}
	c.B(&b.DIMALT)
	c.BS(&b.DIMALTD)
	for _, p := range []*bool{&b.DIMTOFL, &b.DIMSAH, &b.DIMTIX, &b.DIMSOXD} {
		c.B(p)
	}
	c.CMC(&b.DIMCLRD)
	c.CMC(&b.DIMCLRE)
	c.CMC(&b.DIMCLRT)
	for _, p := range []*int16{&b.DIMADEC, &b.DIMDEC, &b.DIMTDEC, &b.DIMALTU, &b.DIMALTTD, &b.DIMAUNIT, &b.DIMFRAC, &b.DIMLUNIT, &b.DIMDSEP, &b.DIMTMOVE, &b.DIMJUST} {
		c.BS(p)
	}
	c.B(&b.DIMSD1)
	c.B(&b.DIMSD2)
	for _, p := range []*int16{&b.DIMTOLJ, &b.DIMTZIN, &b.DIMALTZ, &b.DIMALTTZ} {
		c.BS(p)
	}
	c.B(&b.DIMUPT)
	c.BS(&b.DIMATFIT)
	if c.R2007 {
		c.B(&b.DIMFXLON)
	}
	if c.R2010 {
		c.B(&b.DIMTXTDIRECTION)
		c.BD(&b.DIMALTMZF)
		c.TV(&b.DIMALTMZS)
		c.BD(&b.DIMMZF)
		c.TV(&b.DIMMZS)
	}
	c.BS(&b.DIMLWD)
	c.BS(&b.DIMLWE)
	c.B(&b.Flag0)
}

func (b *DimStyle) handles(c *codec, o *Object) {
	c.H(&b.XrefBlock)
	for _, p := range []*Ref{&b.DIMTXSTY, &b.DIMLDRBLK, &b.DIMBLK, &b.DIMBLK1, &b.DIMBLK2} {
		c.H(p)
	}
	if c.R2007 {
		c.H(&b.DIMLTYPE)
		c.H(&b.DIMLTEX1)
		c.H(&b.DIMLTEX2)
	}
}
