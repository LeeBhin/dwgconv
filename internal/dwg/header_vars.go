// Code generated from the §9 field list (Open Design Specification); field order is the stream order.

package dwg

// HeaderVars are the AcDb:Header variables (§9). Fields named in upper case match the DXF $VARIABLE
// name (DWFPREC3D is $3DDWFPREC); mixed-case fields have no DXF variable of their own.
type HeaderVars struct {
	REQUIREDVERSIONS      uint64
	Unknown1              float64
	Unknown2              float64
	Unknown3              float64
	Unknown4              float64
	UnknownText1          string
	UnknownText2          string
	UnknownText3          string
	UnknownText4          string
	UnknownLong1          int32
	UnknownLong2          int32
	DIMASO                bool
	DIMSHO                bool
	PLINEGEN              bool
	ORTHOMODE             bool
	REGENMODE             bool
	FILLMODE              bool
	QTEXTMODE             bool
	PSLTSCALE             bool
	LIMCHECK              bool
	UnknownBit1           bool
	USRTIMER              bool
	SKPOLY                bool
	ANGDIR                bool
	SPLFRAME              bool
	MIRRTEXT              bool
	WORLDVIEW             bool
	TILEMODE              bool
	PLIMCHECK             bool
	VISRETAIN             bool
	DISPSILH              bool
	PELLIPSE              bool
	PROXYGRAPHICS         int16
	TREEDEPTH             int16
	LUNITS                int16
	LUPREC                int16
	AUNITS                int16
	AUPREC                int16
	ATTMODE               int16
	PDMODE                int16
	UnknownLong3          int32
	UnknownLong4          int32
	UnknownLong5          int32
	USERI1                int16
	USERI2                int16
	USERI3                int16
	USERI4                int16
	USERI5                int16
	SPLINESEGS            int16
	SURFU                 int16
	SURFV                 int16
	SURFTYPE              int16
	SURFTAB1              int16
	SURFTAB2              int16
	SPLINETYPE            int16
	SHADEDGE              int16
	SHADEDIF              int16
	UNITMODE              int16
	MAXACTVP              int16
	ISOLINES              int16
	CMLJUST               int16
	TEXTQLTY              int16
	LTSCALE               float64
	TEXTSIZE              float64
	TRACEWID              float64
	SKETCHINC             float64
	FILLETRAD             float64
	THICKNESS             float64
	ANGBASE               float64
	PDSIZE                float64
	PLINEWID              float64
	USERR1                float64
	USERR2                float64
	USERR3                float64
	USERR4                float64
	USERR5                float64
	CHAMFERA              float64
	CHAMFERB              float64
	CHAMFERC              float64
	CHAMFERD              float64
	FACETRES              float64
	CMLSCALE              float64
	CELTSCALE             float64
	MENU                  string
	TDCREATE              JulianDate
	TDUPDATE              JulianDate
	UnknownLong6          int32
	UnknownLong7          int32
	UnknownLong8          int32
	TDINDWG               JulianDate
	TDUSRTIMER            JulianDate
	CECOLOR               CmColor
	HANDSEED              Ref
	CLAYER                Ref
	TEXTSTYLE             Ref
	CELTYPE               Ref
	CMATERIAL             Ref
	DIMSTYLE              Ref
	CMLSTYLE              Ref
	PSVPSCALE             float64
	PINSBASE              [3]float64
	PEXTMIN               [3]float64
	PEXTMAX               [3]float64
	PLIMMIN               [2]float64
	PLIMMAX               [2]float64
	PELEVATION            float64
	PUCSORG               [3]float64
	PUCSXDIR              [3]float64
	PUCSYDIR              [3]float64
	PUCSNAME              Ref
	PUCSORTHOREF          Ref
	PUCSORTHOVIEW         int16
	PUCSBASE              Ref
	PUCSORGTOP            [3]float64
	PUCSORGBOTTOM         [3]float64
	PUCSORGLEFT           [3]float64
	PUCSORGRIGHT          [3]float64
	PUCSORGFRONT          [3]float64
	PUCSORGBACK           [3]float64
	INSBASE               [3]float64
	EXTMIN                [3]float64
	EXTMAX                [3]float64
	LIMMIN                [2]float64
	LIMMAX                [2]float64
	ELEVATION             float64
	UCSORG                [3]float64
	UCSXDIR               [3]float64
	UCSYDIR               [3]float64
	UCSNAME               Ref
	UCSORTHOREF           Ref
	UCSORTHOVIEW          int16
	UCSBASE               Ref
	UCSORGTOP             [3]float64
	UCSORGBOTTOM          [3]float64
	UCSORGLEFT            [3]float64
	UCSORGRIGHT           [3]float64
	UCSORGFRONT           [3]float64
	UCSORGBACK            [3]float64
	DIMPOST               string
	DIMAPOST              string
	DIMSCALE              float64
	DIMASZ                float64
	DIMEXO                float64
	DIMDLI                float64
	DIMEXE                float64
	DIMRND                float64
	DIMDLE                float64
	DIMTP                 float64
	DIMTM                 float64
	DIMFXL                float64
	DIMJOGANG             float64
	DIMTFILL              int16
	DIMTFILLCLR           CmColor
	DIMTOL                bool
	DIMLIM                bool
	DIMTIH                bool
	DIMTOH                bool
	DIMSE1                bool
	DIMSE2                bool
	DIMTAD                int16
	DIMZIN                int16
	DIMAZIN               int16
	DIMARCSYM             int16
	DIMTXT                float64
	DIMCEN                float64
	DIMTSZ                float64
	DIMALTF               float64
	DIMLFAC               float64
	DIMTVP                float64
	DIMTFAC               float64
	DIMGAP                float64
	DIMALTRND             float64
	DIMALT                bool
	DIMALTD               int16
	DIMTOFL               bool
	DIMSAH                bool
	DIMTIX                bool
	DIMSOXD               bool
	DIMCLRD               CmColor
	DIMCLRE               CmColor
	DIMCLRT               CmColor
	DIMADEC               int16
	DIMDEC                int16
	DIMTDEC               int16
	DIMALTU               int16
	DIMALTTD              int16
	DIMAUNIT              int16
	DIMFRAC               int16
	DIMLUNIT              int16
	DIMDSEP               int16
	DIMTMOVE              int16
	DIMJUST               int16
	DIMSD1                bool
	DIMSD2                bool
	DIMTOLJ               int16
	DIMTZIN               int16
	DIMALTZ               int16
	DIMALTTZ              int16
	DIMUPT                bool
	DIMATFIT              int16
	DIMFXLON              bool
	DIMTXTDIRECTION       bool
	DIMALTMZF             float64
	DIMALTMZS             string
	DIMMZF                float64
	DIMMZS                string
	DIMTXSTY              Ref
	DIMLDRBLK             Ref
	DIMBLK                Ref
	DIMBLK1               Ref
	DIMBLK2               Ref
	DIMLTYPE              Ref
	DIMLTEX1              Ref
	DIMLTEX2              Ref
	DIMLWD                int16
	DIMLWE                int16
	BlockControl          Ref
	LayerControl          Ref
	StyleControl          Ref
	LtypeControl          Ref
	ViewControl           Ref
	UCSControl            Ref
	VportControl          Ref
	AppIDControl          Ref
	DimStyleControl       Ref
	DictGroup             Ref
	DictMLineStyle        Ref
	DictNamedObjects      Ref
	TSTACKALIGN           int16
	TSTACKSIZE            int16
	HYPERLINKBASE         string
	STYLESHEET            string
	DictLayouts           Ref
	DictPlotSettings      Ref
	DictPlotStyles        Ref
	DictMaterials         Ref
	DictColors            Ref
	DictVisualStyle       Ref
	UnknownHandle1        Ref
	Flags                 int32
	INSUNITS              int16
	CEPSNTYPE             int16
	CPSNID                Ref
	FINGERPRINTGUID       string
	VERSIONGUID           string
	SORTENTS              byte
	INDEXCTL              byte
	HIDETEXT              byte
	XCLIPFRAME            byte
	DIMASSOC              byte
	HALOGAP               byte
	OBSCOLOR              int16
	INTERSECTIONCOLOR     int16
	OBSLTYPE              byte
	INTERSECTIONDISPLAY   byte
	PROJECTNAME           string
	BlockRecordPaperSpace Ref
	BlockRecordModelSpace Ref
	LtypeByLayer          Ref
	LtypeByBlock          Ref
	LtypeContinuous       Ref
	CAMERADISPLAY         bool
	UnknownLong9          int32
	UnknownLong10         int32
	UnknownDouble1        float64
	STEPSPERSEC           float64
	STEPSIZE              float64
	DWFPREC3D             float64
	LENSLENGTH            float64
	CAMERAHEIGHT          float64
	SOLIDHIST             byte
	SHOWHIST              byte
	PSOLWIDTH             float64
	PSOLHEIGHT            float64
	LOFTANG1              float64
	LOFTANG2              float64
	LOFTMAG1              float64
	LOFTMAG2              float64
	LOFTPARAM             int16
	LOFTNORMALS           byte
	LATITUDE              float64
	LONGITUDE             float64
	NORTHDIRECTION        float64
	TIMEZONE              int32
	LIGHTGLYPHDISPLAY     byte
	TILEMODELIGHTSYNCH    byte
	DWFFRAME              byte
	DGNFRAME              byte
	REALWORLDSCALE        bool
	INTERFERECOLOR        CmColor
	INTERFEREOBJVS        Ref
	INTERFEREVPVS         Ref
	DRAGVS                Ref
	CSHADOW               byte
	SHADOWPLANELOCATION   float64
}

func (h *HeaderVars) code(c *codec) {
	if c.R2013 {
		codeBLL3(c, &h.REQUIREDVERSIONS)
	}
	c.BD(&h.Unknown1)
	c.BD(&h.Unknown2)
	c.BD(&h.Unknown3)
	c.BD(&h.Unknown4)
	c.TV(&h.UnknownText1)
	c.TV(&h.UnknownText2)
	c.TV(&h.UnknownText3)
	c.TV(&h.UnknownText4)
	c.BL(&h.UnknownLong1)
	c.BL(&h.UnknownLong2)
	c.B(&h.DIMASO)
	c.B(&h.DIMSHO)
	c.B(&h.PLINEGEN)
	c.B(&h.ORTHOMODE)
	c.B(&h.REGENMODE)
	c.B(&h.FILLMODE)
	c.B(&h.QTEXTMODE)
	c.B(&h.PSLTSCALE)
	c.B(&h.LIMCHECK)
	c.B(&h.UnknownBit1)
	c.B(&h.USRTIMER)
	c.B(&h.SKPOLY)
	c.B(&h.ANGDIR)
	c.B(&h.SPLFRAME)
	c.B(&h.MIRRTEXT)
	c.B(&h.WORLDVIEW)
	c.B(&h.TILEMODE)
	c.B(&h.PLIMCHECK)
	c.B(&h.VISRETAIN)
	c.B(&h.DISPSILH)
	c.B(&h.PELLIPSE)
	c.BS(&h.PROXYGRAPHICS)
	c.BS(&h.TREEDEPTH)
	c.BS(&h.LUNITS)
	c.BS(&h.LUPREC)
	c.BS(&h.AUNITS)
	c.BS(&h.AUPREC)
	c.BS(&h.ATTMODE)
	c.BS(&h.PDMODE)
	c.BL(&h.UnknownLong3)
	c.BL(&h.UnknownLong4)
	c.BL(&h.UnknownLong5)
	c.BS(&h.USERI1)
	c.BS(&h.USERI2)
	c.BS(&h.USERI3)
	c.BS(&h.USERI4)
	c.BS(&h.USERI5)
	c.BS(&h.SPLINESEGS)
	c.BS(&h.SURFU)
	c.BS(&h.SURFV)
	c.BS(&h.SURFTYPE)
	c.BS(&h.SURFTAB1)
	c.BS(&h.SURFTAB2)
	c.BS(&h.SPLINETYPE)
	c.BS(&h.SHADEDGE)
	c.BS(&h.SHADEDIF)
	c.BS(&h.UNITMODE)
	c.BS(&h.MAXACTVP)
	c.BS(&h.ISOLINES)
	c.BS(&h.CMLJUST)
	c.BS(&h.TEXTQLTY)
	c.BD(&h.LTSCALE)
	c.BD(&h.TEXTSIZE)
	c.BD(&h.TRACEWID)
	c.BD(&h.SKETCHINC)
	c.BD(&h.FILLETRAD)
	c.BD(&h.THICKNESS)
	c.BD(&h.ANGBASE)
	c.BD(&h.PDSIZE)
	c.BD(&h.PLINEWID)
	c.BD(&h.USERR1)
	c.BD(&h.USERR2)
	c.BD(&h.USERR3)
	c.BD(&h.USERR4)
	c.BD(&h.USERR5)
	c.BD(&h.CHAMFERA)
	c.BD(&h.CHAMFERB)
	c.BD(&h.CHAMFERC)
	c.BD(&h.CHAMFERD)
	c.BD(&h.FACETRES)
	c.BD(&h.CMLSCALE)
	c.BD(&h.CELTSCALE)
	c.TV(&h.MENU)
	c.BL(&h.TDCREATE.Day)
	c.BL(&h.TDCREATE.Ms)
	c.BL(&h.TDUPDATE.Day)
	c.BL(&h.TDUPDATE.Ms)
	c.BL(&h.UnknownLong6)
	c.BL(&h.UnknownLong7)
	c.BL(&h.UnknownLong8)
	c.BL(&h.TDINDWG.Day)
	c.BL(&h.TDINDWG.Ms)
	c.BL(&h.TDUSRTIMER.Day)
	c.BL(&h.TDUSRTIMER.Ms)
	c.CMC(&h.CECOLOR)
	c.HData(&h.HANDSEED)
	c.H(&h.CLAYER)
	c.H(&h.TEXTSTYLE)
	c.H(&h.CELTYPE)
	if c.R2007 {
		c.H(&h.CMATERIAL)
	}
	c.H(&h.DIMSTYLE)
	c.H(&h.CMLSTYLE)
	c.BD(&h.PSVPSCALE)
	c.BD3(&h.PINSBASE)
	c.BD3(&h.PEXTMIN)
	c.BD3(&h.PEXTMAX)
	c.RD2(&h.PLIMMIN)
	c.RD2(&h.PLIMMAX)
	c.BD(&h.PELEVATION)
	c.BD3(&h.PUCSORG)
	c.BD3(&h.PUCSXDIR)
	c.BD3(&h.PUCSYDIR)
	c.H(&h.PUCSNAME)
	c.H(&h.PUCSORTHOREF)
	c.BS(&h.PUCSORTHOVIEW)
	c.H(&h.PUCSBASE)
	c.BD3(&h.PUCSORGTOP)
	c.BD3(&h.PUCSORGBOTTOM)
	c.BD3(&h.PUCSORGLEFT)
	c.BD3(&h.PUCSORGRIGHT)
	c.BD3(&h.PUCSORGFRONT)
	c.BD3(&h.PUCSORGBACK)
	c.BD3(&h.INSBASE)
	c.BD3(&h.EXTMIN)
	c.BD3(&h.EXTMAX)
	c.RD2(&h.LIMMIN)
	c.RD2(&h.LIMMAX)
	c.BD(&h.ELEVATION)
	c.BD3(&h.UCSORG)
	c.BD3(&h.UCSXDIR)
	c.BD3(&h.UCSYDIR)
	c.H(&h.UCSNAME)
	c.H(&h.UCSORTHOREF)
	c.BS(&h.UCSORTHOVIEW)
	c.H(&h.UCSBASE)
	c.BD3(&h.UCSORGTOP)
	c.BD3(&h.UCSORGBOTTOM)
	c.BD3(&h.UCSORGLEFT)
	c.BD3(&h.UCSORGRIGHT)
	c.BD3(&h.UCSORGFRONT)
	c.BD3(&h.UCSORGBACK)
	c.TV(&h.DIMPOST)
	c.TV(&h.DIMAPOST)
	c.BD(&h.DIMSCALE)
	c.BD(&h.DIMASZ)
	c.BD(&h.DIMEXO)
	c.BD(&h.DIMDLI)
	c.BD(&h.DIMEXE)
	c.BD(&h.DIMRND)
	c.BD(&h.DIMDLE)
	c.BD(&h.DIMTP)
	c.BD(&h.DIMTM)
	if c.R2007 {
		c.BD(&h.DIMFXL)
	}
	if c.R2007 {
		c.BD(&h.DIMJOGANG)
	}
	if c.R2007 {
		c.BS(&h.DIMTFILL)
	}
	if c.R2007 {
		c.CMC(&h.DIMTFILLCLR)
	}
	c.B(&h.DIMTOL)
	c.B(&h.DIMLIM)
	c.B(&h.DIMTIH)
	c.B(&h.DIMTOH)
	c.B(&h.DIMSE1)
	c.B(&h.DIMSE2)
	c.BS(&h.DIMTAD)
	c.BS(&h.DIMZIN)
	c.BS(&h.DIMAZIN)
	if c.R2007 {
		c.BS(&h.DIMARCSYM)
	}
	c.BD(&h.DIMTXT)
	c.BD(&h.DIMCEN)
	c.BD(&h.DIMTSZ)
	c.BD(&h.DIMALTF)
	c.BD(&h.DIMLFAC)
	c.BD(&h.DIMTVP)
	c.BD(&h.DIMTFAC)
	c.BD(&h.DIMGAP)
	c.BD(&h.DIMALTRND)
	c.B(&h.DIMALT)
	c.BS(&h.DIMALTD)
	c.B(&h.DIMTOFL)
	c.B(&h.DIMSAH)
	c.B(&h.DIMTIX)
	c.B(&h.DIMSOXD)
	c.CMC(&h.DIMCLRD)
	c.CMC(&h.DIMCLRE)
	c.CMC(&h.DIMCLRT)
	c.BS(&h.DIMADEC)
	c.BS(&h.DIMDEC)
	c.BS(&h.DIMTDEC)
	c.BS(&h.DIMALTU)
	c.BS(&h.DIMALTTD)
	c.BS(&h.DIMAUNIT)
	c.BS(&h.DIMFRAC)
	c.BS(&h.DIMLUNIT)
	c.BS(&h.DIMDSEP)
	c.BS(&h.DIMTMOVE)
	c.BS(&h.DIMJUST)
	c.B(&h.DIMSD1)
	c.B(&h.DIMSD2)
	c.BS(&h.DIMTOLJ)
	c.BS(&h.DIMTZIN)
	c.BS(&h.DIMALTZ)
	c.BS(&h.DIMALTTZ)
	c.B(&h.DIMUPT)
	c.BS(&h.DIMATFIT)
	if c.R2007 {
		c.B(&h.DIMFXLON)
	}
	if c.R2010 {
		c.B(&h.DIMTXTDIRECTION)
	}
	if c.R2010 {
		c.BD(&h.DIMALTMZF)
	}
	if c.R2010 {
		c.TV(&h.DIMALTMZS)
	}
	if c.R2010 {
		c.BD(&h.DIMMZF)
	}
	if c.R2010 {
		c.TV(&h.DIMMZS)
	}
	c.H(&h.DIMTXSTY)
	c.H(&h.DIMLDRBLK)
	c.H(&h.DIMBLK)
	c.H(&h.DIMBLK1)
	c.H(&h.DIMBLK2)
	if c.R2007 {
		c.H(&h.DIMLTYPE)
	}
	if c.R2007 {
		c.H(&h.DIMLTEX1)
	}
	if c.R2007 {
		c.H(&h.DIMLTEX2)
	}
	c.BS(&h.DIMLWD)
	c.BS(&h.DIMLWE)
	c.H(&h.BlockControl)
	c.H(&h.LayerControl)
	c.H(&h.StyleControl)
	c.H(&h.LtypeControl)
	c.H(&h.ViewControl)
	c.H(&h.UCSControl)
	c.H(&h.VportControl)
	c.H(&h.AppIDControl)
	c.H(&h.DimStyleControl)
	c.H(&h.DictGroup)
	c.H(&h.DictMLineStyle)
	c.H(&h.DictNamedObjects)
	c.BS(&h.TSTACKALIGN)
	c.BS(&h.TSTACKSIZE)
	c.TV(&h.HYPERLINKBASE)
	c.TV(&h.STYLESHEET)
	c.H(&h.DictLayouts)
	c.H(&h.DictPlotSettings)
	c.H(&h.DictPlotStyles)
	c.H(&h.DictMaterials)
	c.H(&h.DictColors)
	if c.R2007 {
		c.H(&h.DictVisualStyle)
	}
	if c.R2013 {
		c.H(&h.UnknownHandle1)
	}
	c.BL(&h.Flags)
	c.BS(&h.INSUNITS)
	c.BS(&h.CEPSNTYPE)
	if h.CEPSNTYPE == 3 {
		c.H(&h.CPSNID)
	}
	c.TV(&h.FINGERPRINTGUID)
	c.TV(&h.VERSIONGUID)
	c.RC(&h.SORTENTS)
	c.RC(&h.INDEXCTL)
	c.RC(&h.HIDETEXT)
	c.RC(&h.XCLIPFRAME)
	c.RC(&h.DIMASSOC)
	c.RC(&h.HALOGAP)
	c.BS(&h.OBSCOLOR)
	c.BS(&h.INTERSECTIONCOLOR)
	c.RC(&h.OBSLTYPE)
	c.RC(&h.INTERSECTIONDISPLAY)
	c.TV(&h.PROJECTNAME)
	c.H(&h.BlockRecordPaperSpace)
	c.H(&h.BlockRecordModelSpace)
	c.H(&h.LtypeByLayer)
	c.H(&h.LtypeByBlock)
	c.H(&h.LtypeContinuous)
	if c.R2007 {
		c.B(&h.CAMERADISPLAY)
	}
	if c.R2007 {
		c.BL(&h.UnknownLong9)
	}
	if c.R2007 {
		c.BL(&h.UnknownLong10)
	}
	if c.R2007 {
		c.BD(&h.UnknownDouble1)
	}
	if c.R2007 {
		c.BD(&h.STEPSPERSEC)
	}
	if c.R2007 {
		c.BD(&h.STEPSIZE)
	}
	if c.R2007 {
		c.BD(&h.DWFPREC3D)
	}
	if c.R2007 {
		c.BD(&h.LENSLENGTH)
	}
	if c.R2007 {
		c.BD(&h.CAMERAHEIGHT)
	}
	if c.R2007 {
		c.RC(&h.SOLIDHIST)
	}
	if c.R2007 {
		c.RC(&h.SHOWHIST)
	}
	if c.R2007 {
		c.BD(&h.PSOLWIDTH)
	}
	if c.R2007 {
		c.BD(&h.PSOLHEIGHT)
	}
	if c.R2007 {
		c.BD(&h.LOFTANG1)
	}
	if c.R2007 {
		c.BD(&h.LOFTANG2)
	}
	if c.R2007 {
		c.BD(&h.LOFTMAG1)
	}
	if c.R2007 {
		c.BD(&h.LOFTMAG2)
	}
	if c.R2007 {
		c.BS(&h.LOFTPARAM)
	}
	if c.R2007 {
		c.RC(&h.LOFTNORMALS)
	}
	if c.R2007 {
		c.BD(&h.LATITUDE)
	}
	if c.R2007 {
		c.BD(&h.LONGITUDE)
	}
	if c.R2007 {
		c.BD(&h.NORTHDIRECTION)
	}
	if c.R2007 {
		c.BL(&h.TIMEZONE)
	}
	if c.R2007 {
		c.RC(&h.LIGHTGLYPHDISPLAY)
	}
	if c.R2007 {
		c.RC(&h.TILEMODELIGHTSYNCH)
	}
	if c.R2007 {
		c.RC(&h.DWFFRAME)
	}
	if c.R2007 {
		c.RC(&h.DGNFRAME)
	}
	if c.R2007 {
		c.B(&h.REALWORLDSCALE)
	}
	if c.R2007 {
		c.CMC(&h.INTERFERECOLOR)
	}
	if c.R2007 {
		c.H(&h.INTERFEREOBJVS)
	}
	if c.R2007 {
		c.H(&h.INTERFEREVPVS)
	}
	if c.R2007 {
		c.H(&h.DRAGVS)
	}
	if c.R2007 {
		c.RC(&h.CSHADOW)
	}
	if c.R2007 {
		c.BD(&h.SHADOWPLANELOCATION)
	}
}
