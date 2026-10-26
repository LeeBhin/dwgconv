package dwg

func init() {
	classBodyFactory["IMAGE"] = func() Body { return &Image{} }
	classBodyFactory["WIPEOUT"] = func() Body { return &Wipeout{} }
}

// Image is IMAGE (§20.4.80); WIPEOUT uses the same layout. Unverified: no samples contain these.
type Image struct {
	ClassVersion    int32      // 90
	Pt0             [3]float64 // 10
	UVec            [3]float64 // 11
	VVec            [3]float64 // 12
	Size            [2]float64 // 13
	DisplayProps    int16      // 70
	Clipping        bool       // 280
	Brightness      byte       // 281
	Contrast        byte       // 282
	Fade            byte       // 283
	ClipMode        bool       // 290, R2010+
	ClipBndType     int16      // 71: 1 rect, 2 polygon
	ClipVerts       [][2]float64
	ImageDef        Ref // 340
	ImageDefReactor Ref // 360
}

func (b *Image) data(c *codec, o *Object) {
	c.BL(&b.ClassVersion)
	c.BD3(&b.Pt0)
	c.BD3(&b.UVec)
	c.BD3(&b.VVec)
	c.RD2(&b.Size)
	c.BS(&b.DisplayProps)
	c.B(&b.Clipping)
	c.RC(&b.Brightness)
	c.RC(&b.Contrast)
	c.RC(&b.Fade)
	if c.R2010 {
		c.B(&b.ClipMode)
	}
	c.BS(&b.ClipBndType)
	if b.ClipBndType == 1 {
		if !c.w {
			b.ClipVerts = make([][2]float64, 2)
		}
		c.RD2(&b.ClipVerts[0])
		c.RD2(&b.ClipVerts[1])
		return
	}
	n := c.BLCount(len(b.ClipVerts))
	b.ClipVerts = Slice(c, b.ClipVerts, n)
	for i := range b.ClipVerts {
		c.RD2(&b.ClipVerts[i])
	}
}

func (b *Image) handles(c *codec, o *Object) {
	c.H(&b.ImageDef)
	c.H(&b.ImageDefReactor)
}

// Wipeout is WIPEOUT: the IMAGE layout under its own type so it maps to its own DXF record.
type Wipeout struct{ Image }
