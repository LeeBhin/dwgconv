package container

// Reed-Solomon (255,k) encoding over GF(256) (§5.13). Data pages use k=251 with primitive
// polynomial 1+x^2+x^3+x^4+x^8, system pages k=239 with 1+x^3+x^5+x^6+x^8.

type rsCode struct {
	k        int
	exp, log [512]int
	gen      []int // generator polynomial coefficients, gen[0] constant term
	first    int   // exponent of the first generator root
}

func newRS(k int, poly int, first int) *rsCode {
	r := &rsCode{k: k, first: first}
	x := 1
	for i := 0; i < 255; i++ {
		r.exp[i] = x
		r.log[x] = i
		x <<= 1
		if x&0x100 != 0 {
			x ^= poly
		}
	}
	for i := 255; i < 512; i++ {
		r.exp[i] = r.exp[i-255]
	}
	// gen(x) = prod (x - alpha^(first+i)), i = 0..n-k-1
	r.gen = []int{1}
	for i := 0; i < 255-k; i++ {
		root := r.exp[(first+i)%255]
		next := make([]int, len(r.gen)+1)
		for j, g := range r.gen {
			next[j+1] ^= g
			next[j] ^= r.mul(g, root)
		}
		r.gen = next
	}
	return r
}

func (r *rsCode) mul(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return r.exp[r.log[a]+r.log[b]]
}

// parity returns the n-k parity bytes for k data bytes. data[0] is the highest-degree
// coefficient; parity[0] the highest-degree remainder coefficient.
func (r *rsCode) parity(data []byte) []byte {
	np := 255 - r.k
	rem := make([]int, np) // rem[0] is the highest degree
	for _, d := range data {
		fb := int(d) ^ rem[0]
		copy(rem, rem[1:])
		rem[np-1] = 0
		if fb != 0 {
			for j := 0; j < np; j++ {
				// gen has degree np; coefficient for x^(np-1-j) is gen[np-1-j]
				rem[j] ^= r.mul(fb, r.gen[np-1-j])
			}
		}
	}
	out := make([]byte, np)
	for i, v := range rem {
		out[i] = byte(v)
	}
	return out
}

// encode returns the parity bytes as stored: the file keeps data and parity with the lowest
// degree coefficient first, and the generator roots start at alpha^1 (matched against real files).
func (r *rsCode) encode(data []byte) []byte {
	rd := make([]byte, len(data))
	for i := range data {
		rd[i] = data[len(data)-1-i]
	}
	p := r.parity(rd)
	for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
		p[i], p[j] = p[j], p[i]
	}
	return p
}

var (
	rsData   = newRS(251, 0x11D, 1)
	rsSystem = newRS(239, 0x169, 1)
)
