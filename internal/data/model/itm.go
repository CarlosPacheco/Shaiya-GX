package model

import "fmt"

type ItemBinding struct {
	Bone     int32
	Position [3]float32
	Rotation [4]float32
}
type ItemModel struct {
	// Glow selects item_L/F/E/W.EFT (0..3); -1 disables it. Native
	// 0x556fd8 loads this into +0x20, consumed by 0x4282fb.
	Mesh, Texture, Glow, Unk1, Format, Unk2 int32
	Color                                   uint32
	Rotation, Scale                         float32
	Unk3                                    int32
	Bindings                                [16][2]ItemBinding
}
type ITM struct {
	Signature        string
	Meshes, Textures []string
	Records          []ItemModel
}

// AlphaTest is the material flag read into native record +0x24, not the
// glow selector at +0x20 (0x42909b/0x4298a7). Extended rows store it last.
func (r ItemModel) AlphaTest() bool {
	if r.Format == 1 {
		return r.Unk3 == 0
	}
	return r.Unk2 == 0
}

// ParseITM follows Parsec Itm/ItmRecord; IT2 adds two bindings per archetype.
func ParseITM(b []byte) (*ITM, error) {
	c := &cursor{b: b}
	out := &ITM{Signature: string(c.take(3))}
	if out.Signature != "ITM" && out.Signature != "IT2" {
		return nil, fmt.Errorf("model: invalid ITM signature")
	}
	for i, n := 0, c.count(4); i < n; i++ {
		out.Meshes = append(out.Meshes, c.str())
	}
	for i, n := 0, c.count(4); i < n; i++ {
		out.Textures = append(out.Textures, c.str())
	}
	n := c.count(24)
	out.Records = make([]ItemModel, n)
	for i := range out.Records {
		r := &out.Records[i]
		r.Mesh, r.Texture, r.Glow, r.Unk1, r.Format, r.Unk2 = c.i32(), c.i32(), c.i32(), c.i32(), c.i32(), c.i32()
		if r.Format == 1 {
			r.Color, r.Rotation, r.Scale, r.Unk3 = c.u32(), c.f32(), c.f32(), c.i32()
		} else if r.Format != 0 {
			return nil, fmt.Errorf("ITM: unknown record format %d", r.Format)
		}
		if out.Signature == "IT2" {
			for j := range r.Bindings {
				for k := range r.Bindings[j] {
					bind := &r.Bindings[j][k]
					bind.Bone = c.i32()
					for v := range bind.Position {
						bind.Position[v] = c.f32()
					}
					for v := range bind.Rotation {
						bind.Rotation[v] = c.f32()
					}
				}
			}
		}
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(b) {
		return nil, fmt.Errorf("ITM: %d trailing bytes", len(b)-c.off)
	}
	return out, nil
}
