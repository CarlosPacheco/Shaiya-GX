package world

import (
	"fmt"
	"math"
	"strings"
)

// GXSK follows the native DUN footer. Its three names use the same fixed
// 256-byte encoding and order as FLD, leaving the native DUN bytes intact.
const dungeonSkyTag = "GXSK"
const dungeonSkyNameSize = 256

func (r *reader) readDungeonSky(w *WLD) error {
	rest := r.b[r.off:]
	if len(rest) < len(dungeonSkyTag) || string(rest[:len(dungeonSkyTag)]) != dungeonSkyTag {
		return nil
	}
	if len(rest) != len(dungeonSkyTag)+3*dungeonSkyNameSize {
		return fmt.Errorf("world: invalid GXSK footer length %d", len(rest))
	}
	r.take(len(dungeonSkyTag))
	w.Sky, w.CloudA, w.CloudB = r.fixed(dungeonSkyNameSize), r.fixed(dungeonSkyNameSize), r.fixed(dungeonSkyNameSize)
	return validateDungeonSkyNames(w.Sky, w.CloudA, w.CloudB)
}

func validateDungeonSkyNames(names ...string) error {
	for _, name := range names {
		if len(name) >= dungeonSkyNameSize || strings.ContainsAny(name, "/\\:\x00") || name == "." || name == ".." {
			return fmt.Errorf("world: invalid dungeon sky filename %q", name)
		}
	}
	return nil
}

// PatchDungeonSky writes or replaces GXSK without changing the base DUN,
// including its native fog, geometry and placement records.
func PatchDungeonSky(raw []byte, sky, cloudA, cloudB string) ([]byte, error) {
	r := &reader{b: raw}
	w, err := parseWLD(r)
	if err != nil {
		return nil, err
	}
	if !w.Dungeon {
		return nil, fmt.Errorf("world: GXSK requires a DUN file")
	}
	baseEnd := r.off
	if err := r.readDungeonSky(w); err != nil {
		return nil, err
	}
	if r.off != len(raw) {
		return nil, fmt.Errorf("world: unrecognized DUN footer data")
	}
	if err := validateDungeonSkyNames(sky, cloudA, cloudB); err != nil {
		return nil, err
	}
	out := append([]byte(nil), raw[:baseEnd]...)
	if sky == "" && cloudA == "" && cloudB == "" {
		return out, nil
	}
	out = append(out, dungeonSkyTag...)
	for _, name := range [...]string{sky, cloudA, cloudB} {
		start := len(out)
		out = append(out, make([]byte, dungeonSkyNameSize)...)
		copy(out[start:], name)
	}
	return out, nil
}

// HasDungeonFog distinguishes authored fog from the stock DUN placeholder.
func (w *WLD) HasDungeonFog() bool {
	if !w.Dungeon || !(w.FogStart >= 0 && w.FogEnd > w.FogStart) || math.IsInf(float64(w.FogEnd), 0) {
		return false
	}
	for _, c := range w.FogColor {
		if !(c >= 0 && c <= 255) {
			return false
		}
	}
	return w.FogColor != [3]float32{128, 128, 128} || w.FogStart != 150 || w.FogEnd != 200
}
