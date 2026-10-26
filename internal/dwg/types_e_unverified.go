package dwg

// Types implemented from the spec only: no instances in the sample drawings (unverified).

func init() {
	bodyFactory[0x48] = func() Body { return &Group{} }
	classBodyFactory["GROUP"] = func() Body { return &Group{} }
	classBodyFactory["IMAGEDEFREACTOR"] = func() Body { return &ImageDefReactor{} }
	classBodyFactory["IMAGEDEF_REACTOR"] = func() Body { return &ImageDefReactor{} } // class DXF name
	classBodyFactory["RASTERVARIABLES"] = func() Body { return &RasterVariables{} }
	classBodyFactory["SPATIAL_FILTER"] = func() Body { return &SpatialFilter{} }
	classBodyFactory["FIELD"] = func() Body { return &Field{} }
	classBodyFactory["GEODATA"] = func() Body { return &GeoData{} }
	bodyFactory[0x1F3] = func() Body { return &ProxyObject{} }
}

// Group §20.4.72.
type Group struct {
	Name       string
	Unnamed    int16
	Selectable int16
	Entities   []Ref
}

func (b *Group) data(c *codec, o *Object) {
	c.TV(&b.Name)
	c.BS(&b.Unnamed)
	c.BS(&b.Selectable)
	n := c.BLCount(len(b.Entities))
	b.Entities = Slice(c, b.Entities, n)
}

func (b *Group) handles(c *codec, o *Object) {
	for i := range b.Entities {
		c.H(&b.Entities[i])
	}
}

// ImageDefReactor §20.4.82.
type ImageDefReactor struct{ ClassVersion int32 }

func (b *ImageDefReactor) data(c *codec, o *Object)    { c.BL(&b.ClassVersion) }
func (b *ImageDefReactor) handles(c *codec, o *Object) {}

// RasterVariables §20.4.91.
type RasterVariables struct {
	ClassVersion int32
	DisplayFrame int16
	DisplayQual  int16
	Units        int16
}

func (b *RasterVariables) data(c *codec, o *Object) {
	c.BL(&b.ClassVersion)
	c.BS(&b.DisplayFrame)
	c.BS(&b.DisplayQual)
	c.BS(&b.Units)
}
func (b *RasterVariables) handles(c *codec, o *Object) {}

// SpatialFilter §20.4.94.
type SpatialFilter struct {
	Points       [][2]float64
	Extrusion    [3]float64
	ClipOrigin   [3]float64
	DisplayBound int16
	FrontClipOn  int16
	FrontDist    float64
	BackClipOn   int16
	BackDist     float64
	InvBlockTr   [12]float64
	ClipBoundTr  [12]float64
}

func (b *SpatialFilter) data(c *codec, o *Object) {
	n := c.BSCount(len(b.Points))
	b.Points = Slice(c, b.Points, n)
	for i := range b.Points {
		c.RD2(&b.Points[i])
	}
	c.BD3(&b.Extrusion)
	c.BD3(&b.ClipOrigin)
	c.BS(&b.DisplayBound)
	c.BS(&b.FrontClipOn)
	if b.FrontClipOn == 1 {
		c.BD(&b.FrontDist)
	}
	c.BS(&b.BackClipOn)
	if b.BackClipOn == 1 {
		c.BD(&b.BackDist)
	}
	for i := range b.InvBlockTr {
		c.BD(&b.InvBlockTr[i])
	}
	for i := range b.ClipBoundTr {
		c.BD(&b.ClipBoundTr[i])
	}
}
func (b *SpatialFilter) handles(c *codec, o *Object) {}

// Value §20.4.99 (variant used by FIELD, tables).
type Value struct {
	Flags     int32 // R2007+
	DataType  int32
	Long      int32
	Double    float64
	String    string
	Bytes     []byte // date, general
	Point     []float64
	ObjectID  Ref
	UnitType  int32 // R2007+
	FormatStr string
	ValueStr  string
}

// code runs the data pass (hp=false) or the handle pass (hp=true).
func (v *Value) code(c *codec, hp bool) {
	if hp {
		if v.Flags&1 == 0 && v.DataType == 64 {
			c.H(&v.ObjectID)
		}
		return
	}
	if c.R2007 {
		c.BL(&v.Flags)
	}
	c.BL(&v.DataType)
	if v.Flags&1 == 0 {
		switch v.DataType {
		case 0, 1:
			c.BL(&v.Long)
		case 2:
			c.BD(&v.Double)
		case 4:
			c.TV(&v.String)
		case 8, 512:
			n := c.BLCount(len(v.Bytes))
			c.Raw(&v.Bytes, n)
		case 16, 32:
			n := c.BLCount(len(v.Point) * 8)
			v.Point = Slice(c, v.Point, n/8)
			for i := range v.Point {
				c.RD(&v.Point[i])
			}
		}
	}
	if c.R2007 {
		c.BL(&v.UnitType)
		c.TV(&v.FormatStr)
		c.TV(&v.ValueStr)
	}
}

// Field §20.4.76.
type Field struct {
	EvaluatorID   string
	FieldCode     string
	Children      []Ref
	FieldObjects  []Ref
	Format        string // R2004 and earlier
	EvalOption    int32
	FilingOption  int32
	State         int32
	EvalStatus    int32
	EvalErrorCode int32
	EvalErrorMsg  string
	Value         Value
	ValueString   string
	ValueStrLen   int32
	ChildKeys     []string
	ChildValues   []Value
}

func (b *Field) data(c *codec, o *Object) {
	c.TV(&b.EvaluatorID)
	c.TV(&b.FieldCode)
	n := c.BLCount(len(b.Children))
	b.Children = Slice(c, b.Children, n)
	n = c.BLCount(len(b.FieldObjects))
	b.FieldObjects = Slice(c, b.FieldObjects, n)
	if !c.R2007 {
		c.TV(&b.Format)
	}
	c.BL(&b.EvalOption)
	c.BL(&b.FilingOption)
	c.BL(&b.State)
	c.BL(&b.EvalStatus)
	c.BL(&b.EvalErrorCode)
	c.TV(&b.EvalErrorMsg)
	b.Value.code(c, false)
	c.TV(&b.ValueString)
	c.BL(&b.ValueStrLen)
	n = c.BLCount(len(b.ChildKeys))
	b.ChildKeys = Slice(c, b.ChildKeys, n)
	b.ChildValues = Slice(c, b.ChildValues, n)
	for i := range b.ChildKeys {
		c.TV(&b.ChildKeys[i])
		b.ChildValues[i].code(c, false)
	}
}

func (b *Field) handles(c *codec, o *Object) {
	for i := range b.Children {
		c.H(&b.Children[i])
	}
	for i := range b.FieldObjects {
		c.H(&b.FieldObjects[i])
	}
	b.Value.code(c, true)
	for i := range b.ChildValues {
		b.ChildValues[i].code(c, true)
	}
}

// GeoMeshPoint is a GEODATA mesh point pair.
type GeoMeshPoint struct{ Source, Dest [2]float64 }

// GeoData §20.4.78.
type GeoData struct {
	ObjectVersion   int32
	HostBlock       Ref
	CoordType       int16
	DesignPoint     [3]float64
	ReferencePoint  [3]float64
	HorzUnitScale   float64
	HorzUnits       int32
	VertUnitScale   float64
	VertUnits       int32
	UpDirection     [3]float64
	NorthDirection  [3]float64 // 3RD (version >= 2010)
	ScaleEstimation int32
	UserScale       float64
	SeaLevelCorr    bool
	SeaLevelElev    float64
	ProjRadius      float64
	CoordSystemDef  string
	GeoRSSTag       string
	// version 1 (AutoCAD 2009) layout
	Obsolete1   [3]float64
	NorthAngle  float64
	Obsolete2   [3]float64
	DatumName   string
	ObsoleteWKT string
	ObsFromTag  string
	ObsToTag    string
	ObsCoverTag string
	MeshPoints  []GeoMeshPoint
	MeshFaces   [][3]int32
	// civil data (R21 and earlier)
	HasCivil      bool
	CivilFalse    bool
	CivilRef      [4]float64
	CivilZero1    int32
	CivilZero2    int32
	CivilPt1      [2]float64
	CivilPt2      [2]float64
	CivilFalse2   bool
	NorthDeg      float64
	NorthRad      float64
	CivilScaleEst int32
	CivilUserSc   float64
	CivilSeaCorr  bool
	CivilSeaElev  float64
	CivilProjRad  float64
}

func (b *GeoData) data(c *codec, o *Object) {
	c.BL(&b.ObjectVersion)
	c.BS(&b.CoordType)
	if b.ObjectVersion >= 2 {
		c.BD3(&b.DesignPoint)
		c.BD3(&b.ReferencePoint)
		c.BD(&b.HorzUnitScale)
		c.BL(&b.HorzUnits)
		c.BD(&b.VertUnitScale)
		c.BL(&b.VertUnits)
		c.BD3(&b.UpDirection)
		c.RD3(&b.NorthDirection)
		c.BL(&b.ScaleEstimation)
		c.BD(&b.UserScale)
		c.B(&b.SeaLevelCorr)
		c.BD(&b.SeaLevelElev)
		c.BD(&b.ProjRadius)
		c.TV(&b.CoordSystemDef)
		c.TV(&b.GeoRSSTag)
	} else {
		c.BD3(&b.ReferencePoint)
		c.BL(&b.HorzUnits)
		c.BD3(&b.DesignPoint)
		c.BD3(&b.Obsolete1)
		c.BD3(&b.UpDirection)
		c.BD(&b.NorthAngle)
		c.BD3(&b.Obsolete2)
		c.TV(&b.CoordSystemDef)
		c.TV(&b.GeoRSSTag)
		c.BD(&b.HorzUnitScale)
		c.TV(&b.DatumName)
		c.TV(&b.ObsoleteWKT)
	}
	c.TV(&b.ObsFromTag)
	c.TV(&b.ObsToTag)
	c.TV(&b.ObsCoverTag)
	n := c.BLCount(len(b.MeshPoints))
	b.MeshPoints = Slice(c, b.MeshPoints, n)
	for i := range b.MeshPoints {
		c.RD2(&b.MeshPoints[i].Source)
		c.RD2(&b.MeshPoints[i].Dest)
	}
	n = c.BLCount(len(b.MeshFaces))
	b.MeshFaces = Slice(c, b.MeshFaces, n)
	for i := range b.MeshFaces {
		for k := range b.MeshFaces[i] {
			c.BL(&b.MeshFaces[i][k])
		}
	}
	if !c.R2010 { // "DWG version R21 or lower"
		c.B(&b.HasCivil)
		c.B(&b.CivilFalse)
		for i := range b.CivilRef {
			c.RD(&b.CivilRef[i])
		}
		c.BL(&b.CivilZero1)
		c.BL(&b.CivilZero2)
		c.RD2(&b.CivilPt1)
		c.RD2(&b.CivilPt2)
		c.B(&b.CivilFalse2)
		c.BD(&b.NorthDeg)
		c.BD(&b.NorthRad)
		c.BL(&b.CivilScaleEst)
		c.BD(&b.CivilUserSc)
		c.B(&b.CivilSeaCorr)
		c.BD(&b.CivilSeaElev)
		c.BD(&b.CivilProjRad)
	}
}

func (b *GeoData) handles(c *codec, o *Object) { c.H(&b.HostBlock) }

// ProxyObject §20.4.90. The wrapped object's data is kept verbatim.
type ProxyObject struct {
	ClassID        int32
	DrawingFormat  int32 // before R2018
	Version        int32 // R2018+
	MaintVersion   int32 // R2018+
	OrigDataFormat bool
	Data           []byte
	DataBits       int
	Strings        []byte // R2007+: raw string stream
	StringBits     int
	ObjIDs         []Ref
}

func (b *ProxyObject) data(c *codec, o *Object) {
	c.BL(&b.ClassID)
	if c.R2018 {
		c.BL(&b.Version)
		c.BL(&b.MaintVersion)
	} else {
		c.BL(&b.DrawingFormat)
	}
	c.B(&b.OrigDataFormat)
	eRestBits(c, &b.Data, &b.DataBits)
	if c.R2007 {
		eRestStrBits(c, &b.Strings, &b.StringBits)
	}
}

func (b *ProxyObject) handles(c *codec, o *Object) { eRestHandles(c, &b.ObjIDs) }

// eRestStrBits codes the remaining string-stream bits verbatim (R2007+).
func eRestStrBits(c *codec, b *[]byte, n *int) {
	if c.w {
		c.ws.AppendBits(*b, 0, *n)
		return
	}
	if c.str == c.dat {
		*n = 0
		return
	}
	*n = c.str.Remaining()
	out := make([]byte, (*n+7)/8)
	for i := 0; i < *n; i++ {
		if c.str.B() == 1 {
			out[i>>3] |= 1 << (7 - uint(i&7))
		}
	}
	*b = out
}
