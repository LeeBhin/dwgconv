package container

import (
	"bytes"
	"math/rand"
	"os"
	"testing"
)

func TestCompress21RoundTrip(t *testing.T) {
	raw, _ := os.ReadFile("../../.work/orig/s10.dwg")
	f, err := Read(raw)
	if err != nil {
		t.Skip(err)
	}
	r := rand.New(rand.NewSource(1))
	inputs := [][]byte{[]byte("short"), []byte("exactly8"), make([]byte, 1000)}
	rnd := make([]byte, 5000)
	r.Read(rnd)
	inputs = append(inputs, rnd)
	for _, s := range f.Sections {
		if len(s.Data) > 0 {
			inputs = append(inputs, s.Data[:min(len(s.Data), 0xF800)])
		}
	}
	for i, in := range inputs {
		c := Compress21(in)
		out, err := Decompress21(c, len(in))
		if err != nil || !bytes.Equal(out, in) {
			t.Fatalf("input %d (%d bytes): round trip failed: %v", i, len(in), err)
		}
		t.Logf("input %d: %d -> %d", i, len(in), len(c))
	}
}
