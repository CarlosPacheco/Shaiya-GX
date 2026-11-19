// Package bots runs a client-only PvP simulation with its own actors, combat
// and scene resources. It has no dependency on sessions or game state.
package bots

import (
	"math"
	"math/rand/v2"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

const (
	MaxCount          = 1000
	thinkStep         = .08
	maxHP             = 1000
	outfitsPerFaction = 24
)

var up = render.V3{0, 1, 0}

type Arena struct {
	assets                 *scene.Assets
	world                  *scene.World
	catalog                catalog
	rng                    *rand.Rand
	origin, forward, right render.V3
	actors                 []fighter
	visible                []*scene.Character
	visibleCloth           []*scene.Character
	characterBatch         scene.CharacterBatch
	count                  int
	outfits                [2][]loadout
	effects                []visualEffect
	effectUpdates          []scene.EffectUpdate
	visibleEffects         []*scene.Effect
	failedEffects          map[string]bool
	time                   float32
}

type fighter struct {
	name                                 string
	avatar                               *scene.Character
	loadout                              loadout
	spawn, from, to, facing              render.V3
	started                              float32
	hp                                   int
	nextThink, nextAttack, died, respawn float32
	side                                 float32
	attack                               attack
	number                               damageNumber
}

type attack struct {
	target     int
	skill      *ability
	stage      uint8
	at         float32
	variant    uint8
	actionID   uint64
	contact    float32
	flight     *scene.Effect
	start, end render.V3
	launched   float32
}

const (
	attackCast = iota + 1
	attackContact
	attackFlight
)

func New(arc *archive.Archive, world *scene.World, origin, forward render.V3, count int, seed uint64) *Arena {
	forward[1] = 0
	if forward.Len() == 0 {
		forward = render.V3{0, 0, 1}
	}
	forward = forward.Norm()
	assets := scene.NewAssets(arc)
	return &Arena{
		assets: assets, world: world, catalog: loadCatalog(assets),
		rng:    rand.New(rand.NewPCG(seed, 0)),
		origin: origin, forward: forward, right: up.Cross(forward),
		count: count, actors: make([]fighter, 0, count), failedEffects: make(map[string]bool),
	}
}

func (a *Arena) Close() {
	a.actors, a.effects, a.visible, a.visibleCloth = nil, nil, nil, nil
	a.effectUpdates, a.visibleEffects = nil, nil
	a.characterBatch.Delete()
	a.assets.CloseCharacterResources()
}

func (a *Arena) spawn() {
	index := len(a.actors)
	faction := index % 2
	pool := &a.outfits[faction]
	indexInPool := len(*pool)
	if len(*pool) < outfitsPerFaction {
		*pool = append(*pool, a.catalog.outfit(a.assets, a.rng, faction))
	} else {
		indexInPool = a.rng.IntN(len(*pool))
	}
	kit := (*pool)[indexInPool]
	avatar, _ := scene.BuildCharacter(a.assets, kit.look, "000_normal")
	pos := a.spawnPosition(index)
	facing := a.right.Scale(float32(1 - 2*faction))
	side := float32(a.rng.IntN(2)*2 - 1)
	f := fighter{
		name: "Bot " + strconv.Itoa(index+1), avatar: avatar, loadout: kit,
		spawn: pos, from: pos, to: pos, facing: facing, hp: maxHP,
		nextThink:  a.time + a.random(0, thinkStep),
		nextAttack: a.time + a.random(.25, 1.8), side: side,
	}
	if avatar != nil {
		f.transform(pos)
	}
	a.actors = append(a.actors, f)
}

func (a *Arena) spawnPosition(index int) render.V3 {
	// Grow from the player outward in rows of duels. Opponents start by
	// facing each other, with enough room for approaches and ranged skills.
	columns := int(math.Ceil(math.Sqrt(float64((a.count + 1) / 2))))
	pair := index / 2
	x := (float32(pair%columns) - float32(columns-1)/2) * 4.6
	z := float32(pair/columns)*4.4 + 6
	x += float32(index%2*2-1) * 1.45
	desired := a.origin.Add(a.right.Scale(x)).Add(a.forward.Scale(z))
	for _, scale := range [...]float32{1, .75, .5, .33, .18} {
		p := a.origin.Add(desired.Sub(a.origin).Scale(scale))
		if a.world.HasTerrain() {
			if a.world.WLD == nil || p[0] < 0 || p[2] < 0 || p[0] >= float32(a.world.WLD.MapSize) || p[2] >= float32(a.world.WLD.MapSize) {
				continue
			}
			p[1] = a.world.Height(p[0], p[2])
			if a.world.SwimmingAt(p[0], p[1], p[2]) {
				continue
			}
		}
		if floor, ok := a.world.RayFloor(p.Add(up.Scale(2)), up.Scale(-1), 6); ok {
			p = floor
		}
		p = a.world.StandingPosition(p)
		if a.world.PlayerSupported(p) {
			return p
		}
	}
	return a.origin
}

func (a *Arena) random(lo, hi float32) float32 {
	return lo + (hi-lo)*a.rng.Float32()
}

func (f *fighter) position(now float32) render.V3 {
	return f.from.Add(f.to.Sub(f.from).Scale(min(1, max(0, (now-f.started)/thinkStep))))
}

func (f *fighter) transform(pos render.V3) {
	scale := f.loadout.look.HeightScale()
	f.avatar.World = render.Basis(pos, f.facing.Scale(-1), up)
	for i := 4; i < 7; i++ {
		f.avatar.World[i] *= scale
	}
}

func (a *Arena) Count() int { return len(a.actors) }

// Update advances gameplay and returns true on the frame spawning finishes.
func (a *Arena) Update(dt float32, effects bool) bool {
	dt = min(max(dt, 0), .15)
	a.time += dt
	// Stage avatar construction on the GL owner thread. A large spawn can
	// start fighting while the rest of the requested actors are loading.
	before := len(a.actors)
	budget := 12
	if len(a.actors) < 2*outfitsPerFaction {
		budget = 3
	}
	for range min(budget, a.count-len(a.actors)) {
		a.spawn()
	}
	for i := range a.actors {
		f := &a.actors[i]
		if f.avatar == nil {
			continue
		}
		pos := f.position(a.time)
		f.transform(pos)
		f.avatar.Update(dt)
		if f.hp == 0 {
			f.avatar.CorpseFade = min(1, max(0, (a.time-f.died-1.3)/1.2))
			if a.time >= f.respawn {
				a.revive(f)
			}
			continue
		}
		if f.attack.stage != 0 {
			a.advanceAttack(f, pos, dt, effects)
			continue
		}
		if a.time >= f.nextThink {
			f.nextThink = a.time + thinkStep
			a.think(i, f, pos, effects)
		}
	}
	a.updateEffects(dt, effects)
	return before < a.count && len(a.actors) == a.count
}

func (a *Arena) opponent(i int) int {
	target := i ^ 1
	if target >= a.count && i > 0 {
		target = i - 1
	}
	if target >= len(a.actors) {
		return -1
	}
	return target
}

func (a *Arena) think(i int, f *fighter, pos render.V3, effects bool) {
	target := a.opponent(i)
	if target < 0 && a.count != 1 {
		return
	}
	// A single actor shadow-spars. Larger groups fight the opposing faction.
	dest := f.spawn.Add(f.facing.Scale(2))
	if target >= 0 {
		other := &a.actors[target]
		if other.hp == 0 || other.avatar == nil {
			f.avatar.SetClip("000_normal")
			f.from, f.to = pos, pos
			return
		}
		dest = other.position(a.time)
	}
	delta := dest.Sub(pos)
	delta[1] = 0
	distance := delta.Len()
	dir := f.facing
	if distance > .01 {
		dir = delta.Scale(1 / distance)
	}
	f.facing = dir
	if f.avatar.ActionActive() {
		return
	}
	reach := float32(2.2)
	if f.loadout.ranged {
		reach = 11
	}
	inReach := distance <= reach && a.world.SpellLineOfSight(pos, dest)
	if inReach && a.time >= f.nextAttack && (!f.loadout.ranged || distance > 2.5 || a.rng.IntN(3) == 0) {
		f.from, f.to = pos, pos
		a.startAttack(f, target, pos, effects)
		return
	}

	if distance > reach*.85 {
		a.move(f, pos, dir, "002_run")
	} else if f.loadout.ranged && distance < 4 {
		a.move(f, pos, dir.Scale(-1), "003_bstep")
	} else if target >= 0 && (a.rng.IntN(4) != 0 || !inReach) {
		side := up.Cross(dir).Scale(f.side)
		clip := "004_lstep"
		if f.side < 0 {
			clip = "005_rstep"
		}
		if a.rng.IntN(18) == 0 {
			f.side = -f.side
		}
		a.move(f, pos, side, clip)
	} else {
		f.avatar.SetClip("000_normal")
	}
}

func (a *Arena) move(f *fighter, pos, dir render.V3, clip string) {
	speed := float32(4.4)
	if clip != "002_run" {
		speed = 2.5
	}
	next := a.world.GroundPlayer(pos, dir.Scale(speed*thinkStep), false)
	if f.spawn.Sub(next).Len() > 15 || next.Sub(pos).Len() < .015 || a.world.SwimmingAt(next[0], next[1], next[2]) {
		next, clip = pos, "000_normal"
		f.side = -f.side
	}
	f.from, f.to, f.started = pos, next, a.time
	f.avatar.SetClip(clip)
}

func (a *Arena) startAttack(f *fighter, target int, pos render.V3, effects bool) {
	hit := attack{target: target, variant: uint8(a.rng.IntN(4))}
	if use := f.loadout.skills; len(use) > 0 && (f.loadout.magic || a.rng.IntN(3) == 0) {
		hit.skill = &use[a.rng.IntN(len(use))]
	}
	f.attack = hit
	if hit.skill != nil && hit.skill.def.Attack == 3 {
		cast := a.random(.55, 1)
		f.avatar.PlayCast(scene.SkillCastAction(hit.skill.def.Animation, hit.skill.def.Effect), cast)
		f.attack.stage, f.attack.at = attackCast, a.time+cast
		if effects {
			a.impact(hit.skill.cast, pos, 0, cast)
		}
		return
	}
	a.releaseAttack(f)
}

func (a *Arena) releaseAttack(f *fighter) {
	hit := &f.attack
	weapon := f.loadout.look.WeaponType
	var played bool
	if hit.skill == nil {
		played = f.avatar.PlayBasicAttack(weapon, hit.variant)
	} else {
		s := hit.skill
		slot := scene.SkillAction(s.def.Animation, s.def.Attack, s.def.Effect, weapon, hit.variant)
		played = f.avatar.PlaySkillAction(s.id, slot) || f.avatar.PlayBasicAttack(weapon, hit.variant)
	}
	hit.stage, hit.at = attackContact, a.time+.25
	if contact := f.avatar.ContactTime(); played && contact > 0 {
		hit.actionID, hit.contact = f.avatar.ActionID(), contact
		hit.at += contact
	}
	f.nextAttack = a.time + a.random(.75, 1.7)
}

func (a *Arena) advanceAttack(f *fighter, pos render.V3, dt float32, effects bool) {
	hit := &f.attack
	if hit.stage == attackFlight && hit.flight != nil && effects {
		progress := min(1, (a.time-hit.launched)/(hit.at-hit.launched))
		to := hit.end
		if hit.target >= 0 {
			to = a.actors[hit.target].position(a.time).Add(up.Scale(1))
		}
		dir := to.Sub(hit.start)
		point := hit.start.Add(dir.Scale(progress))
		hit.flight.SetTransform(render.Basis(point, dir.Norm(), up))
		a.effectUpdates = append(a.effectUpdates, scene.EffectUpdate{Instance: hit.flight, Delta: dt})
	}
	atContact := hit.stage == attackContact && hit.contact > 0 &&
		f.avatar.ActionID() == hit.actionID && f.avatar.Time >= hit.contact
	if a.time < hit.at && !atContact {
		return
	}
	switch hit.stage {
	case attackCast:
		a.releaseAttack(f)
		return
	case attackContact:
		if hit.skill != nil && hit.skill.flight != "" && hit.target >= 0 {
			hit.start = pos.Add(up.Scale(1.3))
			hit.end = a.actors[hit.target].position(a.time).Add(up.Scale(1))
			if effects {
				hit.flight = a.effect(hit.skill.flight, 0)
				if hit.flight != nil {
					hit.flight.SetTransform(render.Basis(hit.start, f.facing, up))
				}
			}
			hit.stage, hit.launched, hit.at = attackFlight, a.time, a.time+.24
			return
		}
	}
	if hit.target >= 0 {
		other := &a.actors[hit.target]
		dest := other.position(a.time)
		reach := float32(3)
		if f.loadout.ranged {
			reach = 14
		}
		if other.hp > 0 && dest.Sub(pos).Len() <= reach && a.world.SpellLineOfSight(pos, dest) {
			a.damage(other, dest, hit.skill, effects)
		}
	} else if effects {
		a.impact("Effect/DA_target_Adamage.EFT", pos.Add(f.facing.Scale(1.5)), 0, 1.1)
	}
	f.attack = attack{}
}

func (a *Arena) damage(other *fighter, pos render.V3, skill *ability, effects bool) {
	other.number = damageNumber{point: pos.Add(up.Scale(1.65)), at: a.time}
	if a.rng.IntN(10) == 0 {
		other.number.missed = true
		return
	}
	damage := 55 + a.rng.IntN(100)
	file := "Effect/DA_target_Adamage.EFT"
	if skill != nil {
		damage += 55
		file = skill.impact
	}
	if a.rng.IntN(8) == 0 {
		damage *= 2
		other.number.critical = true
	}
	other.number.amount = uint32(damage)
	other.hp = max(0, other.hp-damage)
	if effects {
		a.impact(file, pos, 0, 2)
	}
	if other.hp == 0 {
		other.died, other.respawn = a.time, a.time+a.random(3.5, 5.5)
		other.from, other.to = pos, pos
		other.attack = attack{}
		other.avatar.PlayAction(9, true)
	}
}

func (a *Arena) revive(f *fighter) {
	f.hp = maxHP
	f.from, f.to = f.spawn, f.spawn
	f.avatar.CorpseFade = 0
	f.avatar.ReleaseAction()
	f.avatar.SetClip("000_normal")
	f.transform(f.spawn)
	f.nextAttack = a.time + a.random(.4, 1.5)
}
