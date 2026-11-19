package scene

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const MinCrowdActors = 24
const maxCrowdPaletteMatrices = 1 << 20 // 64 MiB, below the OpenGL 4.5 SSBO limit.

type rigidMaterial struct {
	mesh    *render.Mesh
	texture *render.Texture
	ref     float32
}

type rigidGroup struct {
	material rigidMaterial
	worlds   []render.M4
}

type CrowdDrawStats struct {
	Skin         int `json:"skin"`
	Rigid        int `json:"weapons_shields"`
	Cloth        int `json:"cloth"`
	Fallback     int `json:"serial"`
	ClothBytes   int `json:"cloth_position_bytes"`
	ClothUploads int `json:"cloth_position_uploads"`
}

// CharacterBatch gathers already visible crowd actors for one camera pass.
type CharacterBatch struct {
	keys        map[render.SkinGroup]int
	groups      []render.SkinGroup
	buckets     [][]render.SkinInstance
	instances   []render.SkinInstance
	palettes    []render.M4
	offsets     [][4]int32
	rigidKeys   map[rigidMaterial]int
	rigidGroups []rigidGroup
	cloaks      []*Character
	cloth       render.ClothCrowd
	Stats       CrowdDrawStats
}

func (b *CharacterBatch) Draw(k *render.Skinner, s *render.Scene3D, actors []*Character) {
	b.Stats = CrowdDrawStats{}
	if len(actors) < MinCrowdActors || !s.HideWeaponEffects || s.Shadow {
		before := render.DrawCalls()
		for _, c := range actors {
			c.Draw(k, s)
		}
		b.Stats.Fallback = render.DrawCalls() - before
		return
	}
	if b.keys == nil {
		b.keys = make(map[render.SkinGroup]int)
		b.rigidKeys = make(map[rigidMaterial]int)
	}
	b.cloth.Begin(len(actors))
	for _, c := range actors {
		if c.Concealed || c.hasBlendedAttachment() {
			b.flush(k, s)
			before := render.DrawCalls()
			c.Draw(k, s)
			b.Stats.Fallback += render.DrawCalls() - before
			continue
		}
		finals, visible := c.prepareDraw(s)
		if visible {
			if len(b.palettes)+len(c.parts)*render.MaxBones > maxCrowdPaletteMatrices {
				b.flush(k, s)
			}
			b.add(s, c, finals)
		}
	}
	b.flush(k, s)
	b.Stats.ClothBytes, b.Stats.ClothUploads = b.cloth.Uploaded, b.cloth.UploadOps
}

func (b *CharacterBatch) Delete() { b.cloth.Delete() }

func (c *Character) hasBlendedAttachment() bool {
	if !c.Mounted {
		for _, a := range c.attachments {
			if a.part.Alpha {
				return true
			}
		}
	}
	return false
}

func (b *CharacterBatch) add(s *render.Scene3D, c *Character, finals []render.M4) {
	instance := s.CrowdSkinInstance(c.World)
	b.offsets = b.offsets[:0]
	for i, part := range c.parts {
		source := c.poseSources[i]
		if source == i {
			count := min(len(part.pose), render.MaxBones)
			b.offsets = append(b.offsets, [4]int32{int32(len(b.palettes)), int32(count)})
			b.palettes = append(b.palettes, part.pose[:count]...)
		} else {
			b.offsets = append(b.offsets, b.offsets[source])
		}
		if part.mesh.Count == 0 {
			continue
		}
		ref := float32(-1)
		if part.cut {
			ref = .5
		}
		group := render.SkinGroup{Mesh: part.mesh, Texture: part.tex, AlphaRef: ref, Unlit: part.unlit, TwoSided: part.twoSided}
		index, ok := b.keys[group]
		if !ok {
			index = len(b.groups)
			b.keys[group] = index
			b.groups = append(b.groups, group)
			if index == len(b.buckets) {
				b.buckets = append(b.buckets, nil)
			}
		}
		instance.Palette = b.offsets[i]
		instance.Palette[2] = int32(math.Float32bits(c.CorpseFade))
		b.buckets[index] = append(b.buckets[index], instance)
	}
	if c.cloak != nil {
		b.cloaks = append(b.cloaks, c)
	}
	if c.Mounted {
		return
	}
	for i, a := range c.attachments {
		if a.bone < 0 || a.bone >= len(finals) || i == 0 && c.hiddenWeaponAction != 0 && c.hiddenWeaponAction == c.ActionID() {
			continue
		}
		world := a.extra.Mul(finals[a.bone]).Mul(c.World)
		if !s.RigidInstanceVisible(a.part.Mesh, world) {
			continue
		}
		material := rigidMaterial{mesh: a.part.Mesh, texture: a.part.Tex, ref: -1}
		if a.cut {
			material.ref = .5
		}
		index, ok := b.rigidKeys[material]
		if !ok {
			index = len(b.rigidKeys)
			b.rigidKeys[material] = index
			if index == len(b.rigidGroups) {
				b.rigidGroups = append(b.rigidGroups, rigidGroup{})
			}
			b.rigidGroups[index].material = material
		}
		group := &b.rigidGroups[index]
		group.worlds = append(group.worlds, world)
	}
}

func (b *CharacterBatch) flush(k *render.Skinner, s *render.Scene3D) {
	for i := range b.groups {
		instances := b.buckets[i]
		b.groups[i].First, b.groups[i].Count = len(b.instances), len(instances)
		b.instances = append(b.instances, instances...)
		b.buckets[i] = instances[:0]
	}
	ambient := s.CharacterAmbient
	if ambient <= 0 {
		ambient = .8
	}
	before := render.DrawCalls()
	k.DrawCrowd(s, b.groups, b.instances, b.palettes, ambient)
	b.Stats.Skin += render.DrawCalls() - before
	before = render.DrawCalls()
	if len(b.cloaks) != 0 || len(b.rigidKeys) != 0 {
		s.Begin()
	}
	for _, c := range b.cloaks {
		cloak := c.cloak
		if !s.ModelBoundsOccluded(cloak.min, cloak.max, c.World) {
			if !b.cloth.Queue(s, cloak.geometry.mesh, cloak.geometry.texture, cloak.id, cloak.version, cloak.vertices, c.World) {
				cloak.drawInPass(s, c.World, false)
			}
		}
	}
	b.cloth.Draw(s)
	b.Stats.Cloth += render.DrawCalls() - before
	before = render.DrawCalls()
	if len(b.rigidKeys) != 0 {
		for i := range len(b.rigidKeys) {
			group := &b.rigidGroups[i]
			m := group.material
			s.SetInstances(group.worlds)
			s.DrawInstanced(m.mesh, m.texture, m.ref)
			group.material = rigidMaterial{}
			group.worlds = group.worlds[:0]
		}
	}
	b.Stats.Rigid += render.DrawCalls() - before
	clear(b.keys)
	clear(b.rigidKeys)
	clear(b.groups)
	clear(b.cloaks)
	b.groups, b.instances, b.palettes, b.cloaks = b.groups[:0], b.instances[:0], b.palettes[:0], b.cloaks[:0]
}
