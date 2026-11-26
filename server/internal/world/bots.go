package world

import (
	"fmt"
	"math"
	"math/rand/v2"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const botHealth = 12000
const botTick = 100 * time.Millisecond
const botHomeRange = 5
const botMeleeSpacing = 2.15

type worldBot struct {
	castAt             time.Time
	castTarget         int
	castSkill          *db.SkillDef
	wire               *botWire
	char               db.Character
	home               mapgeometry.Position
	faction            uint8
	items              []db.Item
	skills             []*db.SkillDef
	target             int
	nextAction, revive time.Time
}

type worldBotArena struct {
	failed   bool
	owner    uint32
	geometry *mapgeometry.Geometry
	bots     []worldBot
	visible  map[*ConnectedPlayer]*botObserver
}

var worldBots struct {
	sync.Mutex
	nextID uint32
	arena  *worldBotArena
}

func (s *Session) gmBots(args []string) error {
	host, _, err := net.SplitHostPort(config.WorldListen())
	if err != nil || !net.ParseIP(host).IsLoopback() && host != "localhost" {
		return newGMError(12442)
	}
	if len(args) == 1 && strings.EqualFold(args[0], "off") {
		worldBots.Lock()
		clearWorldBots()
		worldBots.Unlock()
		s.gmMessage(12441)
		return nil
	}
	n, err := gmNumbers(args, 1000)
	if err != nil {
		return newGMError(12443)
	}
	g, err := s.playerGeometry(s.char.MapID)
	if err != nil {
		return err
	}
	a := &worldBotArena{owner: s.char.ID, geometry: g, visible: make(map[*ConnectedPlayer]*botObserver)}
	defs := db.MustDefs()
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		return err
	}
	defer listener.Close()
	listener.SetDeadline(time.Now().Add(30 * time.Second))
	gear := botEquipment(defs)
	worldBots.Lock()
	defer worldBots.Unlock()
	homes := make([]mapgeometry.Position, 0, int(n[0]))
	for i := range int(n[0]) {
		angle := float64(i/2) * 2.399963229728653
		radius := float32(2.5 + math.Sqrt(float64(i/2))*2.15)
		sin, cos := math.Sincos(angle)
		side := float32(-2)
		if i%2 != 0 {
			side = -side
		}
		x := s.char.PosX + radius*float32(cos) - side*float32(sin)
		z := s.char.PosZ + radius*float32(sin) + side*float32(cos)
		home, ok := separatedBotHome(g, x, s.char.PosY, z, homes)
		if !ok {
			return newGMError(12444, n[0])
		}
		homes = append(homes, home)
	}
	for i, home := range homes {
		if worldBots.nextID < 0x60000000 {
			worldBots.nextID = 0x60000000
		}
		for {
			worldBots.nextID++
			if FindPlayerByID(worldBots.nextID) == nil {
				break
			}
		}
		faction := uint8(i % 2)
		job := uint8((i / 2) % 6)
		if i < 2 {
			job = 5
		} // A healer in each faction, including small scenarios.
		b := worldBot{home: home, faction: faction, target: i ^ 1, char: db.Character{
			ID: worldBots.nextID, Name: fmt.Sprintf("GX Bot %04d", i+1), MapID: s.char.MapID,
			PosX: home[0], PosY: home[1], PosZ: home[2], Class: job, Race: [2][6]uint8{{0, 0, 1, 1, 1, 0}, {3, 3, 2, 2, 2, 2}}[faction][job], Height: 2,
			Gender: uint8(rand.IntN(2)), Face: uint8(rand.IntN(3)), Hair: uint8(rand.IntN(3)), Level: defs.LevelCap,
			MaxHP: botHealth, CurrentHP: botHealth, MaxMP: 3000, CurrentMP: 3000, MaxSP: 3000, CurrentSP: 3000,
		}}
		b.wire, err = openBotWire(listener)
		if err != nil {
			for j := range a.bots {
				a.bots[j].wire.close()
			}
			return err
		}
		b.equip(defs, gear)
		a.bots = append(a.bots, b)
	}
	clearWorldBots()
	worldBots.arena = a
	s.gmMessage(12440, len(a.bots))
	return nil
}

// When terrain blocks a formation point, look for a nearby spot before
// opening any connections. Sending blocked bots back to the GM stacks them.
func separatedBotHome(g *mapgeometry.Geometry, x, y, z float32, homes []mapgeometry.Position) (mapgeometry.Position, bool) {
	pick := func(x, z float32) (mapgeometry.Position, bool) {
		for _, home := range homes {
			dx, dz := x-home[0], z-home[2]
			if dx*dx+dz*dz < 1.8*1.8 {
				return mapgeometry.Position{}, false
			}
		}
		if p, ok := exactMapDestination(g, x, y, z); ok {
			return p, true
		}
		if g.HasTerrain() && abs32(g.Height(x, z)-y) <= 12 {
			return exactMapDestination(g, x, g.Height(x, z), z)
		}
		return mapgeometry.Position{}, false
	}
	if p, ok := pick(x, z); ok {
		return p, true
	}
	for radius := float64(2); radius <= 18; radius += 2 {
		points := int(math.Ceil(math.Pi * radius))
		for j := range points {
			sin, cos := math.Sincos(float64(j) * 2 * math.Pi / float64(points))
			if p, ok := pick(x+float32(radius*cos), z+float32(radius*sin)); ok {
				return p, true
			}
		}
	}
	return mapgeometry.Position{}, false
}

func botEquipment(defs *db.Defs) (bySlot [8][]*db.ItemDef) {
	for _, item := range defs.Items {
		if item.TypeID == 0 {
			continue
		}
		rt := realType(item.Type)
		for slot := range bySlot {
			if slot == weaponSlot && isWeaponReal(rt) || slot != weaponSlot && int(rt) == equipSlotRealType[uint8(slot)] {
				bySlot[slot] = append(bySlot[slot], item)
				break
			}
		}
	}
	return
}

func (b *worldBot) equip(defs *db.Defs, gear [8][]*db.ItemDef) {
	for slot, options := range gear {
		// Choose a random starting point, then check each candidate at most once.
		if len(options) == 0 {
			continue
		}
		start := rand.IntN(len(options))
		for offset := range options {
			item := options[(start+offset)%len(options)]
			if canEquip(&b.char, item, uint8(slot), b.items) {
				b.items = append(b.items, db.Item{Type: item.Type, TypeID: item.TypeID, Slot: uint8(slot), Count: 1})
				break
			}
		}
	}
	for _, skill := range defs.Skills {
		if skill.SkillLevel == 1 && skill.ClassEligible[b.char.Class] && skill.ReqLevel <= b.char.Level &&
			skillCountryAllows(skill.Country, b.char.Race) && skill.TypeShow != 0 &&
			(skill.TypeEffect == typeEffectAttack && skill.DamageHP > 0 || skill.TypeEffect == typeEffectHeal && skill.HealHP > 0) {
			b.skills = append(b.skills, skill)
		}
	}
	slices.SortFunc(b.skills, func(a, b *db.SkillDef) int { return int(a.SkillID) - int(b.SkillID) })
	b.skills = b.skills[:min(len(b.skills), 255)]
}

func (a *worldBotArena) advance(i int, now time.Time) {
	b := &a.bots[i]
	if b.char.CurrentHP == 0 {
		if now.Before(b.revive) {
			return
		}
		b.char.PosX, b.char.PosY, b.char.PosZ = b.home[0], b.home[1], b.home[2]
		b.char.CurrentHP = botHealth
		b.castSkill = nil
		b.nextAction = now.Add(time.Second)
		a.emit(buildCharacterReviveInit(b.char.ID), i, -1)
		a.emit(buildCharacterResurrect(b.char.ID, 1, 0, b.home[0], b.home[1], b.home[2]), i, -1)
		a.emit(buildCharacterReviveHitpoints(b.char.ID, botHealth, 3000, 3000), i, -1)
		return
	}
	if b.castSkill != nil {
		if now.Before(b.castAt) {
			return
		}
		skill, target := b.castSkill, b.castTarget
		b.castSkill = nil
		a.resolveAttack(i, target, skill, now)
		return
	}
	// Prefer the opponent deployed beside this bot. While that opponent is
	// down, look near home so temporary targets cannot draw the formation in.
	pair := i ^ 1
	if pair < len(a.bots) && a.bots[pair].char.CurrentHP > 0 {
		b.target = pair
	} else if b.target == pair || b.target < 0 || b.target >= len(a.bots) || a.bots[b.target].char.CurrentHP == 0 {
		b.target = -1
		nearest := float32(8 * 8)
		for j := 1 - int(b.faction); j < len(a.bots); j += 2 {
			enemy := &a.bots[j].char
			dx, dz := enemy.PosX-b.home[0], enemy.PosZ-b.home[2]
			if distance := dx*dx + dz*dz; enemy.CurrentHP > 0 && distance < nearest {
				b.target, nearest = j, distance
			}
		}
	}
	if b.target < 0 {
		return
	}
	target := b.target
	t := &a.bots[target]
	dx, dz := t.char.PosX-b.char.PosX, t.char.PosZ-b.char.PosZ
	distance := float32(math.Hypot(float64(dx), float64(dz)))
	dir := uint16(int(math.Atan2(float64(dx), float64(dz)) * 65536 / (2 * math.Pi)))
	if !now.Before(b.nextAction) {
		for _, slot := range rand.Perm(len(b.skills)) {
			skill := b.skills[slot]
			k := target
			if skill.TypeEffect == typeEffectHeal {
				k = -1
				for j := int(b.faction); j < len(a.bots); j += 2 {
					ally := &a.bots[j]
					if i == j || ally.char.CurrentHP == 0 || ally.char.CurrentHP > botHealth-900 {
						continue
					}
					ax, az := ally.char.PosX-b.char.PosX, ally.char.PosZ-b.char.PosZ
					if ax*ax+az*az <= float32(max(skill.AttackRange, 8))*float32(max(skill.AttackRange, 8)) {
						k = j
						break
					}
				}
			} else if distance > float32(max(skill.AttackRange, 3)) {
				continue
			}
			if k < 0 {
				continue
			}
			a.applyRequest(i, proto.UseSkillPacket(uint8(slot), a.bots[k].char.ID, true), k, now)
			return
		}
		if distance < 2.6 {
			a.applyRequest(i, proto.UserAttackPacket(t.char.ID), target, now)
			return
		}
	}
	if distance > botMeleeSpacing {
		step := min(.44, distance-botMeleeSpacing)
		from := mapgeometry.Position{b.char.PosX, b.char.PosY, b.char.PosZ}
		p := a.geometry.GroundMove(from, mapgeometry.Position{dx / distance * step, 0, dz / distance * step}, false)
		hx, hz := p[0]-b.home[0], p[2]-b.home[2]
		if p != from && hx*hx+hz*hz <= botHomeRange*botHomeRange {
			a.applyRequest(i, proto.MovePacket(1, dir, p[0], p[1], p[2]), -1, now)
		}
	}
}

func (a *worldBotArena) applyRequest(i int, payload []byte, target int, now time.Time) {
	b := &a.bots[i]
	received, err := b.wire.request(payload)
	if err != nil {
		a.failed = true
		return
	}
	payload = received
	r := proto.NewReader(payload)
	switch proto.Op(payload) {
	case proto.CharacterMove:
		motion, dir := r.U8(), r.U16()
		x, y, z := r.F32(), r.F32(), r.F32()
		if r.Err() != nil {
			return
		}
		b.char.PosX, b.char.PosY, b.char.PosZ = x, y, z
		b.char.Dir = int16(dir)
		a.emit(buildCharacterMove(&b.char, motion, dir), i, -1)
	case proto.CharacterUserAutoAttack, proto.SkillUseNoTarget:
		if target < 0 || target >= len(a.bots) {
			return
		}
		t := &a.bots[target]
		if t.char.CurrentHP == 0 {
			return
		}
		var skill *db.SkillDef
		if proto.Op(payload) == proto.SkillUseNoTarget {
			slot := int(r.U8())
			if slot >= len(b.skills) {
				return
			}
			skill = b.skills[slot]
		}
		if r.U32() != t.char.ID || r.Err() != nil {
			return
		}
		if skill != nil && skill.ReadyTime > 0 {
			b.castAt = now.Add(time.Duration(skill.ReadyTime) * 250 * time.Millisecond)
			b.castTarget, b.castSkill = target, skill
			w := packet.New(packet.SkillCastStarted)
			w.U32(b.char.ID)
			w.U32(t.char.ID)
			w.U16(skill.SkillID)
			w.U8(skill.SkillLevel)
			a.emit(w.MustFinish(), i, target)
			return
		}
		a.resolveAttack(i, target, skill, now)
	}
}

func (a *worldBotArena) resolveAttack(i, target int, skill *db.SkillDef, now time.Time) {
	b, t := &a.bots[i], &a.bots[target]
	b.nextAction = now.Add(time.Duration(850+rand.IntN(650)) * time.Millisecond)
	if t.char.CurrentHP == 0 {
		return
	}

	heal := skill != nil && skill.TypeEffect == typeEffectHeal
	if heal != (b.faction == t.faction) {
		return
	}

	if heal {
		amount := min(uint32(900), botHealth-t.char.CurrentHP)
		t.char.CurrentHP += amount
		a.emit(buildSkillUseResult(packet.SkillUseNoTarget, skillResultNormal, b.char.ID, t.char.ID, skill.SkillID, skill.SkillLevel, uint16(amount), 0, 0), i, target)
		return
	}
	damage := min(t.char.CurrentHP, uint32(300+rand.IntN(450)))
	t.char.CurrentHP -= damage
	if skill != nil {
		a.emit(buildSkillUseResult(opUserSkillPvP, attackSuccessNormal, b.char.ID, t.char.ID, skill.SkillID, skill.SkillLevel, uint16(damage), 0, 0), i, target)
	} else {
		a.emit(buildCharacterUserAttack(b.char.ID, t.char.ID, attackSuccessNormal, uint16(damage)), i, target)
	}
	if t.char.CurrentHP == 0 {
		t.revive = now.Add(3 * time.Second)
		a.emit(buildCharacterDeath(t.char.ID, deathTypeUser, b.char.ID), target, -1)
	}
}
