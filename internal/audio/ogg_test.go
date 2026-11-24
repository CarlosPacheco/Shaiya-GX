package audio

import (
	"bytes"
	_ "embed"
	"math/rand/v2"
	"testing"
)

// Generated 440 Hz sine at 22050 Hz, mono, 80 ms; no game recording.
//
//go:embed testdata/tone.ogg
var oggTone []byte

func TestOGGResamplesAndDuplicatesMono(t *testing.T) {
	pcm, err := DecodeOGG(oggTone)
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) != SampleRate*4*80/1000 {
		t.Fatal("duration", len(pcm))
	}
	nonzero := false
	for i := 0; i < len(pcm); i += 4 {
		if !bytes.Equal(pcm[i:i+2], pcm[i+2:i+4]) {
			t.Fatal("mono channels differ")
		}
		nonzero = nonzero || pcm[i] != 0 || pcm[i+1] != 0
	}
	if !nonzero {
		t.Fatal("silent output")
	}
}
func TestOGGRejectsMalformedAndTruncated(t *testing.T) {
	for _, raw := range [][]byte{nil, []byte("RIFFnot an ogg"), oggTone[:15], oggTone[:len(oggTone)/2]} {
		if _, err := DecodeOGG(raw); err == nil {
			t.Fatal("accepted invalid OGG")
		}
	}
}

// A damaged file must come back as an error: the Vorbis library panics on some
// corrupt setup headers, and the client decodes sounds on its main loop.
func TestOGGCorruptionNeverPanics(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 13))
	for round := 0; round < 6000; round++ {
		raw := append([]byte(nil), oggTone...)
		for k := 0; k < 1+rng.IntN(6); k++ {
			raw[rng.IntN(min(len(raw), 1200))] = byte(rng.IntN(256))
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("round %d: DecodeOGG panicked: %v", round, r)
				}
			}()
			DecodeOGG(raw)
		}()
	}
}

// Twenty-one corrupted bytes of the tone make the Vorbis library index an empty
// table ("index out of range [255] with length 0"): it must surface as an error.
func TestOGGLibraryPanicBecomesError(t *testing.T) {
	raw := append([]byte(nil), oggTone...)
	for _, c := range [][2]int{{1000, 75}, {3824, 227}, {2817, 11}, {990, 121}, {1975, 189}, {1641, 139}, {3378, 200}, {26, 0}, {3789, 28}, {3561, 226}, {253, 40}, {1976, 226}, {1874, 166}, {1906, 160}, {3975, 43}, {685, 242}, {1269, 9}, {3230, 203}, {3215, 146}, {1380, 153}, {2067, 68}} {
		raw[c[0]] = byte(c[1])
	}
	if _, err := DecodeOGG(raw); err == nil {
		t.Fatal("corrupt stream was accepted")
	}
}
