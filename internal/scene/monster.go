package scene

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/attackspeed"
	"path"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// CreatureActor shares the character skinning path, with clips from MON.
type CreatureActor struct {
	boneEffects      []creatureBoneEffect
	boneEffectsReady bool
	attackEffects    []*Effect
	pendingEffects   []string
	*Character
	Name        string
	Level       uint16
	Definition  sdata.Monster
	folder      string
	visual      model.Creature
	action      int
	cueTime     float32
	visualScale float32

	plateTop                     float32 // body height for FitPlate
	plateTopReady, plateTopValid bool
}

func (s *Assets) MonsterDefinition(id uint16) (sdata.Monster, error) {
	if s.monsters == nil {
		b, err := s.Arc.ReadFile("Monster/Monster.SData")
		if err != nil {
			return sdata.Monster{}, err
		}
		s.monsters, err = sdata.ParseMonsters(b)
		if err != nil {
			return sdata.Monster{}, err
		}
	}
	if int(id) >= len(s.monsters) {
		return sdata.Monster{}, fmt.Errorf("monster %d out of range", id)
	}
	return s.monsters[id], nil
}

func (s *Assets) BuildMonster(id uint16) (*CreatureActor, error) {
	def, err := s.MonsterDefinition(id)
	if err != nil {
		return nil, err
	}
	if s.creatures == nil {
		b, err := s.Arc.ReadFile("Monster/monster.MON")
		if err != nil {
			return nil, err
		}
		s.creatures, err = model.ParseMON(b)
		if err != nil {
			return nil, err
		}
	}
	if int(def.Model) >= len(s.creatures) {
		return nil, fmt.Errorf("monster model %d out of range", def.Model)
	}
	visual := s.creatures[def.Model]
	actor, err := s.buildCreature("Monster", visual, def.Name)
	if actor != nil {
		actor.Level, actor.Definition = def.Level, def
		actor.visualScale = def.VisualScale()
		actor.SetTransform(render.Basis(render.V3{}, render.V3{0, 0, -1}, render.V3{0, 1, 0}))
	}
	return actor, err
}

func (s *Assets) BuildNPC(def sdata.NPC) (*CreatureActor, error) {
	if s.npcVisuals == nil {
		b, err := s.Arc.ReadFile("Npc/npc.MON")
		if err != nil {
			return nil, err
		}
		s.npcVisuals, err = model.ParseMON(b)
		if err != nil {
			return nil, err
		}
	}
	if def.Model < 0 || int(def.Model) >= len(s.npcVisuals) {
		return nil, fmt.Errorf("NPC model %d out of range", def.Model)
	}
	return s.buildCreature("Npc", s.npcVisuals[def.Model], def.Name)
}

func (s *Assets) buildCreature(folder string, visual model.Creature, name string) (*CreatureActor, error) {
	c := &Character{assets: s}
	for _, part := range visual.Parts {
		p, err := s.skinFile(folder + "/3DC/" + path.Base(part[0]))
		if err != nil {
			return nil, err
		}
		cp := *p
		cp.tex = s.modelTexture(folder + "/DDS/" + DDSName(path.Base(part[1])))
		cp.cut = true
		c.parts = append(c.parts, &cp)
	}
	m := &CreatureActor{Character: c, Name: name, folder: folder, visual: visual, action: -1}
	if err := m.SetAction(6); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *CreatureActor) SetAction(action int) error {
	return m.setAction(action, false)
}

func (m *CreatureActor) setAction(action int, oneShot bool) error {
	if action == 5 {
		m.cueTime = 0
	}
	if m.cueTime > 0 {
		return nil
	}
	if m.action == action && !oneShot {
		return nil
	}
	if action < 0 || action >= len(m.visual.Animations) {
		return fmt.Errorf("invalid creature action %d", action)
	}
	clip, err := m.assets.Clip(m.folder + "/ANI/" + path.Base(m.visual.Animations[action]))
	if err != nil {
		return err
	}
	m.startClip(clip, oneShot || action == 5)
	m.action = action
	if action == 5 {
		m.queueModelEffect(action)
	}
	return nil
}

// Death is a one-shot; live movement/breath clips loop in the shared sampler.
func (m *CreatureActor) Update(dt float32) {
	oneShot := m.cueTime > 0
	// The cue lasts one playback of the clip: the blend runs at real time, the
	// rest at the actor's rate.
	blend := min(m.blendLeft, dt)
	m.cueTime = max(0, m.cueTime-(blend+(dt-blend)*m.rate.factor()))
	m.Character.Update(dt)
	if (m.action == 5 || oneShot) && m.clip != nil {
		if end := m.clip.Duration(); end > 0 {
			m.Time = min(m.Time, end)
		}
	}
}

// PlayOnce presents a confirmed attack/hit; locomotion resumes afterward.
func (m *CreatureActor) PlayOnce(action int) {
	// Late attack, reaction or repeated death packets cannot restart a corpse.
	// A real respawn switches back to a live action through SetAction.
	if m.action == 5 {
		return
	}
	// A damage reaction must not restart or replace a confirmed attack.
	if action == 7 && m.Attacking() {
		return
	}
	m.cueTime = 0
	if err := m.setAction(action, true); err != nil {
		m.assets.Missing[err.Error()] = true
		return
	}
	if action != 5 {
		m.queueModelEffect(action)
	}
	if m.clip != nil {
		m.cueTime = m.clip.Duration() + m.blendLeft
	}
}

// Sound returns the original MON attack1/2/3 or death WAV reference.
func (m *CreatureActor) Sound(index int) string {
	if m == nil || index < 0 || index >= len(m.visual.Sounds) {
		return ""
	}
	return m.visual.Sounds[index]
}

func (m *CreatureActor) Attacking() bool { return m.cueTime > 0 && m.action >= 2 && m.action <= 4 }
func (m *CreatureActor) AttackAction(slot uint8) int {
	if slot > 2 {
		slot = 0
	}
	action := int(m.Definition.AttackAnimations[slot]) - 1
	if action < 0 || action > 2 {
		action = 0
	}
	return action + 2
}

// NumberHeight is the authored MON vertical anchor used by native 0x450a10.
func (m *CreatureActor) NumberHeight() float32 {
	height := m.visual.Height*m.modelScale() + .2 // 0x431770: MON height plus native headroom.
	if height <= -1 {
		return 1.2
	}
	return height
}

func (m *CreatureActor) modelScale() float32 {
	// NPCs retain their authored model size.
	if m.visualScale == 0 {
		return 1
	}
	return m.visualScale
}

// SetTransform applies the native uniform model scale before world placement.
// Skinning, bounds, picking, shadows and bone attachments share this matrix.
func (m *CreatureActor) SetTransform(world render.M4) {
	scale := m.modelScale()
	for row := range 3 {
		for col := range 3 {
			world[row*4+col] *= scale
		}
	}
	m.World = world
}

// SetSpeedIndices picks the playback rate of the current action from the
// monster's attack and move indices (0x0609): walking and chasing follow the
// move index, every other action the attack index.
func (m *CreatureActor) SetSpeedIndices(move, attack uint8) {
	if m.action == 0 || m.action == 1 {
		m.SetCreatureRate(attackspeed.MobMoveFPS(move))
	} else {
		m.SetCreatureRate(attackspeed.MobActionFPS(attack))
	}
}
