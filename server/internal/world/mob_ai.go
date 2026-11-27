package world

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"github.com/Spelunkern/Shaiya-GX/server/ai"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// All AI decisions enter Lua. This adapter supplies authoritative world
// primitives under the shard lock; cross-controller operations are deferred.
var mobAIProgram atomic.Pointer[ai.Program]
var mobAIClock = time.Now()

func loadMobAI() error {
	dir := os.Getenv("RAPTURE_AI_DIR")
	if dir == "" {
		candidate := filepath.Join(filepath.Dir(config.DataDir()), "ai")
		if st, e := os.Stat(candidate); e == nil && st.IsDir() {
			dir = candidate
		}
	}
	var p *ai.Program
	var err error
	if dir != "" {
		p, err = ai.Load(os.DirFS(dir))
	} else {
		p, err = ai.Default()
	}
	if err != nil {
		return fmt.Errorf("mob Lua catalog: %w", err)
	}
	mobAIProgram.Store(p)
	return nil
}
func aiProgram() (*ai.Program, error) {
	if p := mobAIProgram.Load(); p != nil {
		return p, nil
	}
	return ai.Default()
}

type mobAIState struct {
	actor                                        *ai.Actor
	rng                                          *rand.Rand
	initialized                                  bool
	fault                                        error
	lastTarget                                   uint32
	suppressHelp                                 bool
	needReset                                    bool
	damage, speed, defense, recovery             float32
	mask                                         uint8
	invulnerable, hold, disabled, recoverEnabled bool
	recoverAt, nextFSMUpdate                     time.Time
	recoverPeriod                                time.Duration
	attacked, recentHit, counting                bool
	hits, attackedCount                          int
	battleType                                   int
	element                                      uint8
	elementSet                                   bool
	selected                                     mapnav.Point
	selectedValid, fixed, arrived                bool
	fixedRun                                     bool
	escapeX, escapeY, escapeZ                    float32
}

// The native zone scheduler runs UpdateIdle every 1000 ms (0x407646,
// 0x40766d). FSM cases advance there, even while the mob is in combat.
func (m *MapMob) mobAIUpdateDue(now time.Time) bool {
	if m.AI < 10 || m.AI > 36 {
		return true
	}
	if now.Before(m.lua.nextFSMUpdate) {
		return false
	}
	m.lua.nextFSMUpdate = now.Add(time.Second)
	return true
}

func (m *MapMob) resetAI() {
	m.lua = mobAIState{initialized: true, damage: 1, speed: 1, defense: 1, recovery: 1, mask: 15, recoverPeriod: 5 * time.Second}
	if m.aiSummoned && m.aiAttackType != 4 {
		m.lua.mask = 0
		if m.aiAttackType < 4 {
			m.lua.mask = 1 << m.aiAttackType
		}
	}
}
func (m *MapMob) ensureAI(h *mobAIHost) bool {
	if m.lua.fault != nil {
		return false
	}
	if m.lua.actor != nil {
		return true
	}
	suppress := m.lua.suppressHelp
	m.aiMapID = h.mapID
	m.resetAI()
	m.lua.suppressHelp = suppress
	if h.shard.lua == nil {
		p, e := aiProgram()
		if e != nil {
			m.failAI(e)
			return false
		}
		h.shard.lua, e = p.NewVM()
		if e != nil {
			m.failAI(e)
			return false
		}
	}
	commandStart := len(h.shard.aiCommands)
	actor, e := h.shard.lua.NewActor(m.AI, h)
	if e != nil {
		m.failAI(e)
		return false
	}
	m.lua.actor = actor
	for i := commandStart; i < len(h.shard.aiCommands); i++ {
		h.shard.aiCommands[i].actor = actor
	}
	return true
}
func (m *MapMob) failAI(err error) {
	m.lua.fault = err
	m.AggroTarget = 0
	log.Printf("GX Server World AI error mob_global_id=%d mob_id=%d ai=%d: %v", m.ID, m.MobID, m.AI, err)
}
func (h *mobAIHost) event(name string, args ...any) bool {
	m := h.mob
	if !m.ensureAI(h) {
		return false
	}
	if e := m.lua.actor.Event(h, name, args...); e != nil {
		m.failAI(e)
		return false
	}
	return true
}
func (h *mobAIHost) notifyTarget() {
	m := h.mob
	if m.AggroTarget != 0 && m.lua.lastTarget != m.AggroTarget {
		help := m.lua.lastTarget == 0 && !m.lua.suppressHelp
		m.lua.lastTarget = m.AggroTarget
		m.lua.suppressHelp = false
		h.event("target", m.AggroTarget, help)
	}
}

type mobAIHost struct {
	mob     *MapMob
	shard   *mapShard
	mapID   uint16
	now     time.Time
	players []PlayerCombatSnapshot
	tick    *MobCombatTick
}

func (h *mobAIHost) queue(kind string, id uint16, count int, x, z float32, mode int) error {
	cost := 1
	if kind == "spawn" {
		cost += count
		live := 0
		for _, mob := range h.shard.entities.Mobs {
			if mob.aiSummoned && !mob.corpseGone {
				live++
			}
		}
		if live+h.shard.aiPendingSummons+count > 4096 {
			return fmt.Errorf("summoned mob budget exceeded")
		}
	}
	if len(h.shard.aiCommands) >= 1024 || h.shard.aiCommandCost+cost > 2048 {
		return fmt.Errorf("AI command budget exceeded")
	}
	h.shard.aiCommandCost += cost
	if kind == "spawn" {
		h.shard.aiPendingSummons += count
	}
	m := h.mob
	h.shard.aiCommands = append(h.shard.aiCommands, mobAICommand{kind: kind, mapID: h.mapID, owner: m.ID, actor: m.lua.actor, id: id, count: count, x: x, z: z, y: m.PosY, mode: mode})
	return nil
}
func aiNumber(args []any, i int) float64 {
	if i >= len(args) {
		return 0
	}
	switch x := args[i].(type) {
	case int64:
		return float64(x)
	case float64:
		return x
	case float32:
		return float64(x)
	case uint32:
		return float64(x)
	}
	return 0
}
func aiBool(args []any, i int) bool {
	if i >= len(args) {
		return false
	}
	if x, ok := args[i].(bool); ok {
		return x
	}
	return aiNumber(args, i) != 0
}
func (h *mobAIHost) Call(name string, args []any) (any, error) {
	m := h.mob
	o := &m.lua
	n := func(i int) float64 { return aiNumber(args, i) }
	for _, a := range args {
		if x, ok := a.(float64); ok && (math.IsNaN(x) || math.IsInf(x, 0)) {
			return nil, fmt.Errorf("non-finite argument")
		}
	}
	switch name {
	case "time_ms":
		return uint32(h.now.Sub(mobAIClock) / time.Millisecond), nil
	case "hp_fraction":
		return float32(m.CurrentHP) / float32(max(m.MaxHP, 1)), nil
	case "status":
		if m.CurrentHP == 0 {
			return "dead", nil
		}
		if o.fixed {
			return "moving", nil
		}
		if m.returningHome {
			return "return_home", nil
		}
		if m.AggroTarget != 0 {
			return "combat", nil
		}
		return "idle", nil
	case "can_act":
		return m.mods().canAct(), nil
	case "lua_callbacks_enabled":
		return !o.disabled, nil
	case "native_attack_ready":
		return !h.now.Before(m.nextAttackAt), nil
	case "acquisition_ready":
		return !h.now.Before(m.noTargetUntil) && !o.hold, nil
	case "find_target":
		return m.findTarget(h.now, h.players, aiBool(args, 0)) != nil, nil
	case "was_attacked":
		return o.attacked, nil
	case "consume_recent_hit":
		v := o.recentHit
		o.recentHit = false
		return v, nil
	case "is_in_combat":
		return m.AggroTarget != 0, nil
	case "legacy_end_battle_stub":
		return nil, nil
	case "target_user_id":
		return m.AggroTarget, nil
	case "random_0_99":
		return rand.IntN(100), nil
	case "lua_random":
		limit := int(n(0))
		if limit < 1 || limit > 1000000 {
			return nil, fmt.Errorf("invalid RNG bound")
		}
		if o.rng != nil {
			return o.rng.IntN(limit) + 1, nil
		}
		return rand.IntN(limit) + 1, nil
	case "seed_lua_random_from_wall_clock":
		seed := uint64(h.now.Unix())
		o.rng = rand.New(rand.NewPCG(seed, seed))
		return nil, nil
	case "highest_current_hp_attacker_id":
		var id uint32
		best := int64(-1)
		for _, p := range h.players {
			if !p.Alive || p.Invisible || mobFriendly(m.def, p.Faction) {
				continue
			}
			_, ok := m.aggro[p.CharacterID]
			if !ok {
				continue
			}
			score := int64(p.HP)
			if score > best {
				best = score
				id = p.CharacterID
			}
		}
		return id, nil
	case "hit_count":
		return max(o.hits, 0), nil
	case "begin_hit_count":
		o.counting = true
	case "end_hit_count":
		o.counting = false
	case "reset_hit_count":
		o.hits = 0
	case "set_damage_multiplier", "set_attack_speed_multiplier", "set_physical_defense_multiplier", "set_recovery_multiplier":
		v := float32(n(0))
		if v <= 0 || v > 1000 {
			return nil, fmt.Errorf("invalid multiplier")
		}
		switch name {
		case "set_damage_multiplier":
			o.damage = v
		case "set_attack_speed_multiplier":
			o.speed = v
		case "set_physical_defense_multiplier":
			o.defense = v
		case "set_recovery_multiplier":
			o.recovery = v
		}
	case "set_defensive_element":
		v := int(n(0))
		if v < 0 || v > 8 {
			return nil, fmt.Errorf("invalid element")
		}
		o.element = uint8(v)
		o.elementSet = true
	case "set_invulnerable":
		o.invulnerable = aiBool(args, 0)
	case "hold_position":
		o.hold = aiBool(args, 0)
	case "disable_lua_callbacks":
		o.disabled = true
		log.Printf("GX Server World AI missing Lua source mob_id=%d ai=%d; native stack retained", m.MobID, m.AI)
	case "select_attack_type":
		v := int(n(0))
		switch {
		case v == 4:
			o.mask = 15
		case v == 5:
			o.mask = 0
		case v >= 0 && v < 4:
			o.mask = 1 << v
		default:
			return nil, fmt.Errorf("invalid attack category")
		}
	case "enable_all_attack_types":
		o.mask = 15
	case "set_battle_type":
		o.battleType = int(n(0))
	case "enable_recovery":
		o.recoverEnabled = aiBool(args, 0)
	case "recovery_enabled":
		return o.recoverEnabled, nil
	case "set_recovery_interval_ms":
		v := int64(n(0))
		if v < 1 || v > 86400000 {
			return nil, fmt.Errorf("invalid recovery interval")
		}
		o.recoverPeriod = time.Duration(v) * time.Millisecond
	case "recover_hp":
		if m.def != nil && m.def.RecoverValue > 0 && m.CurrentHP < m.MaxHP && !h.now.Before(o.recoverAt) {
			amount := m.def.RecoverValue
			period := 5 * time.Second
			if o.recoverEnabled {
				amount = int32(float32(amount) * o.recovery)
				period = o.recoverPeriod
			}
			h.heal(uint32(max(amount, 0)))
			o.recoverAt = h.now.Add(period)
		}
	case "heal":
		h.heal(uint32(max(min(n(0), math.MaxUint32), 0)))
	case "refresh_target":
		h.refreshTarget()
	case "idle_wander":
		if !aiBool(args, 0) && !o.hold && (m.def == nil || !m.def.Obelisk) && m.Area != nil && m.movable() {
			m.advanceRoam(h.now, h.mapID, h.tick)
		}
	case "return_home":
		mods := m.mods()
		if o.hold {
			m.returningHome = false // a held actor never walks home
		} else if m.stepHome(h.now, h.mapID, &mods, h.tick) {
			return nil, nil
		}
		if !m.returningHome {
			o.needReset = true
		}
	case "attack_ai":
		h.attack(aiBool(args, 0))
	case "battle":
		h.attack(false)
	case "reset_generic":
		m.AggroTarget = 0
		o.lastTarget = 0
		o.attacked = false
		o.recentHit = false
	case "clear_target_flags":
		if m.mods().VisionFix == 0 {
			m.AggroTarget = 0
			o.attacked = false
			o.recentHit = false
		}
	case "recruit_idle":
		if len(args) < 2 {
			return nil, fmt.Errorf("recruit_idle requires user and policy")
		}
		kind, ok := args[1].(string)
		if !ok || (kind != "same_template" && kind != "aggressive" && kind != "all_idle") {
			return nil, fmt.Errorf("invalid recruitment policy")
		}
		h.recruit(uint32(n(0)), m.PosX, m.PosZ, m.ChaseRange, kind)
	case "recruit_around_user":
		user := uint32(n(0))
		for _, p := range h.players {
			if p.CharacterID == user {
				h.recruit(user, p.PosX, p.PosZ, float32(n(1)), "all_idle")
				break
			}
		}
	case "select_target_user":
		id := uint32(n(0))
		for _, p := range h.players {
			if p.CharacterID == id && p.Alive && !p.Invisible && !mobFriendly(m.def, p.Faction) {
				m.AggroTarget = id
				break
			}
		}
	case "use_mob_skill":
		h.skill(uint16(n(0)))
	case "announce", "announce_legacy", "say_legacy":
		index := int(n(0))
		radius := float32(n(1))
		if name == "say_legacy" {
			radius = BroadcastRadius
		}
		if radius <= 0 {
			radius = BroadcastRadius
		}
		kind := "say_text"
		if name == "announce" {
			kind = "say_index"
		}
		if err := h.queue(kind, uint16(index), 1, m.PosX, m.PosZ, int(radius)); err != nil {
			return nil, err
		}
	case "create_mobs", "remove_mobs", "summon_legacy":
		id := uint16(n(0))
		count := int(n(1))
		x, z := float32(n(2)), float32(n(3))
		mode := 4
		spread := float32(10)
		kind := "spawn"
		if name == "remove_mobs" {
			kind = "remove"
		}
		if name == "summon_legacy" {
			count = int(n(2))
			mode = int(n(3))
			spread = float32(n(1))
			if mode < 0 || mode > 5 || spread < 1 || spread > 1000 {
				return nil, fmt.Errorf("invalid summon arguments")
			}
			x, z = m.PosX, m.PosZ
		}
		if count < 1 || count > 128 {
			return nil, fmt.Errorf("invalid mob count")
		}
		if x == 0 && z == 0 {
			x, z = m.PosX, m.PosZ
		}
		if db.MustDefs().Mobs[id] == nil {
			return nil, fmt.Errorf("unknown mob template %d", id)
		}
		if err := h.queue(kind, id, count, x, z, mode); err != nil {
			return nil, err
		}
		h.shard.aiCommands[len(h.shard.aiCommands)-1].spread = spread
		h.shard.aiCommands[len(h.shard.aiCommands)-1].legacy = name == "summon_legacy"
	case "update_instance_portal":
		if err := h.queue("portal", uint16(n(0)), 1, 0, 0, int(n(1))); err != nil {
			return nil, err
		}
	case "record_escape_origin":
		o.escapeX, o.escapeY, o.escapeZ = m.PosX, m.PosY, m.PosZ
	case "distance_from_escape_origin":
		return float32(math.Sqrt(float64((m.PosX-o.escapeX)*(m.PosX-o.escapeX) + (m.PosY-o.escapeY)*(m.PosY-o.escapeY) + (m.PosZ-o.escapeZ)*(m.PosZ-o.escapeZ)))), nil
	case "select_escape_direction":
		h.selectDirection(float32(n(0)))
	case "select_idle_ally_position":
		var best *MapMob
		distance := float32(n(0))
		for i := range h.shard.entities.Mobs {
			candidate := &h.shard.entities.Mobs[i]
			if candidate.ID == m.ID || candidate.CurrentHP == 0 || candidate.AggroTarget != 0 {
				continue
			}
			d := DistanceXZ(m.PosX, m.PosZ, candidate.PosX, candidate.PosZ)
			if d < distance {
				best = candidate
				distance = d
			}
		}
		if best != nil {
			o.selected = mapnav.Point{best.PosX, best.PosY, best.PosZ}
			o.selectedValid = true
			o.arrived = false
		}
	case "fixed_move":
		if o.hold || m.returningHome || m.CurrentHP == 0 || !m.nav.allowsXZ(h.mapID, float32(n(1)), float32(n(2))) {
			return nil, nil
		}
		if n(0) != 0 && n(0) != 1 {
			return nil, fmt.Errorf("invalid movement mode")
		}
		o.fixedRun = n(0) == 1
		o.selected = mapnav.Point{float32(n(1)), m.PosY, float32(n(2))}
		o.selectedValid = true
		o.fixed = true
		o.arrived = false
	case "advance_selected_move":
		if o.fixed {
			h.moveSelected(o.fixedRun)
			return true, nil
		}
		return false, nil
	case "arrived_at_selected_position":
		return o.arrived, nil
	case "move_to_selected_position":
		h.moveSelected(true)
	case "random_walk", "random_run":
		if !o.selectedValid || o.arrived {
			h.selectDirection(0)
		}
		h.moveSelected(name == "random_run")
	default:
		return nil, fmt.Errorf("unimplemented AI operation %q", name)
	}
	return nil, nil
}
func (h *mobAIHost) heal(amount uint32) {
	m := h.mob
	if m.CurrentHP == 0 {
		return
	}
	before := m.CurrentHP
	m.CurrentHP = min(satAdd(before, amount), m.MaxHP)
	if m.CurrentHP != before {
		h.tick.Restores = append(h.tick.Restores, mapPacket{h.mapID, m.PosX, m.PosZ, buildMobCurrentHP(m.ID, m.CurrentHP)})
	}
}
func (h *mobAIHost) refreshTarget() {
	m := h.mob
	mods := m.mods()
	if mods.VisionFix != 0 {
		for _, p := range h.players {
			if p.CharacterID == mods.VisionFix && p.Alive && !p.Invisible {
				m.AggroTarget = p.CharacterID
				m.returningHome = false
			}
		}
	}
	target := m.liveTarget(h.players)
	if m.AggroTarget != 0 && target == nil {
		m.retargetByAggro(h.players)
		target = m.liveTarget(h.players)
		if target == nil {
			m.dropTarget(true)
		}
	}
	if target != nil && mods.VisionFix == 0 {
		pos := m.positionAt(h.now)
		if DistanceXZ(target.PosX, target.PosZ, m.homeX, m.homeZ) > mobLeash || DistanceXZ(target.PosX, target.PosZ, pos[0], pos[2]) > mobLeash {
			m.dropTarget(true)
		}
	}
}
func (h *mobAIHost) recruit(user uint32, x, z, radius float32, kind string) {
	caller := h.mob
	for i := range h.shard.entities.Mobs {
		m := &h.shard.entities.Mobs[i]
		if m.ID == caller.ID || m.CurrentHP == 0 || m.AggroTarget != 0 || m.returningHome || !isNeighborCell(m.cell, caller.cell) {
			continue
		}
		pos := m.positionAt(h.now)
		if DistanceXZ(x, z, pos[0], pos[2]) >= radius {
			continue
		}
		if kind == "same_template" && m.MobID != caller.MobID {
			continue
		}
		if kind == "aggressive" { // Policy query is evaluated by the helper's Lua definition.
			p, e := aiProgram()
			if e != nil || !p.AcquiresProximity(m.AI) {
				continue
			}
		}
		m.AggroTarget = user
		m.lua.suppressHelp = true
	}
}
func (h *mobAIHost) selectDirection(extra float32) {
	m := h.mob
	angle := rand.Float64() * math.Pi * 2
	length := float32(1)
	if m.def != nil {
		length = float32(m.def.NormalStep) + extra
	}
	m.lua.selected = mapnav.Point{m.PosX + float32(math.Sin(angle))*length, m.PosY, m.PosZ + float32(math.Cos(angle))*length}
	m.lua.selectedValid = true
	m.lua.arrived = false
}
func (h *mobAIHost) moveSelected(run bool) {
	m := h.mob
	o := &m.lua
	if !o.selectedValid || o.hold || m.movingAt(h.now) || !m.movable() || h.now.Before(m.nextChaseAt) {
		return
	}
	goal := o.selected
	d := DistanceXZ(m.PosX, m.PosZ, goal[0], goal[2])
	if d < .1 {
		o.arrived = true
		o.selectedValid = false
		return
	}
	mods := m.mods()
	step := min(m.chaseStep(&mods), d)
	if !run && m.def != nil {
		step = min(max(float32(m.def.NormalStep), 1), d)
	}
	next := mapnav.Point{m.PosX + (goal[0]-m.PosX)/d*step, m.PosY, m.PosZ + (goal[2]-m.PosZ)/d*step}
	if m.nav != nil {
		var ok bool
		next, ok = m.navigate(h.now, h.mapID, goal, step)
		if !ok {
			return
		}
	} else if !IsMapWalkable(h.mapID, next[0], next[2]) {
		return
	}
	motion := uint8(mobMotionWalk)
	if run {
		motion = mobMotionRun
	}
	h.tick.Moves = append(h.tick.Moves, m.moveTo(h.mapID, h.now, next, motion))
}

func (h *mobAIHost) attack(stationary bool) {
	m := h.mob
	mods := m.mods()
	if m.def == nil || !mods.canAct() || m.lua.fixed || m.movingAt(h.now) {
		return
	}
	target := m.liveTarget(h.players)
	if target == nil {
		return
	}
	canAttack := !h.now.Before(m.nextAttackAt)
	var weakest *PlayerCombatSnapshot
	if canAttack && mods.VisionFix == 0 {
		weakest = m.targetWeakest(h.players)
		m.retargetByAggro(h.players)
		if t := m.liveTarget(h.players); t != nil {
			target = t
		} else {
			m.AggroTarget = target.CharacterID
		}
	}
	d := m.targetDistance(target)
	if canAttack {
		slot, ok, declinedSpell := m.pickAttackSlot(h.now, &mods, func(slot int, reach float32) bool {
			if weakest != nil && m.def.Attacks[slot].OK == 6 {
				return m.targetDistance(weakest) <= reach
			}
			return d <= reach
		}, func() bool { return rand.IntN(2) == 0 })
		if ok {
			if weakest != nil && m.def.Attacks[slot].OK == 6 {
				m.AggroTarget = weakest.CharacterID
			}
			m.lastAttackAt = h.now
			// It stays where it is while the attack animation plays.
			m.nextChaseAt = later(m.nextChaseAt, h.now.Add(min(time.Duration(m.def.Attacks[slot].TimeMS)*time.Millisecond, mobAttackAnimation)))
			if heal := m.useSlot(slot, h.now); heal != nil {
				h.tick.Restores = append(h.tick.Restores, mapPacket{h.mapID, m.PosX, m.PosZ, heal})
				if sk := db.MustDefs().NPCSkill(m.def.Attacks[slot].Type); sk != nil {
					h.tick.Restores = append(h.tick.Restores, mapPacket{h.mapID, m.PosX, m.PosZ, buildMobSkillPacket(0x060B, 0, m.ID, m.ID, sk, [3]uint16{}, nil, slot)})
				}
				return
			}
			h.tick.Attacks = append(h.tick.Attacks, MobAttackAction{MobGlobalID: m.ID, MobID: m.MobID, Attack: m.Attack, TargetCharacterID: m.AggroTarget, Slot: slot, MobX: m.PosX, MobZ: m.PosZ})
			return
		}
		inBasicReach := d <= m.attackReach()
		if declinedSpell || inBasicReach {
			m.nextAttackAt = h.now.Add(mobAttackIdle + mods.attackDelayOffset())
		}
		if inBasicReach {
			return
		}
	}
	if !stationary && !m.lua.hold && m.def != nil && m.def.ChaseStep != 0 && !m.def.Obelisk {
		m.stepChase(h.now, h.mapID, target, d, &mods, h.tick)
	}
}
func (h *mobAIHost) skill(id uint16) {
	m := h.mob
	sk := db.MustDefs().NPCSkill(id)
	if sk == nil || m.def == nil || !m.mods().canAct() {
		return
	}
	slot := uint8(0)
	for i, attack := range m.def.Attacks {
		if attack.Type == id {
			slot = uint8(i)
			break
		}
	}
	pos := m.positionAt(h.now)
	if sk.TypeEffect == 2 {
		h.heal(uint32(max(sk.HealHP, 0)))
		h.tick.Restores = append(h.tick.Restores, mapPacket{h.mapID, pos[0], pos[2], buildMobSkillPacket(0x060B, 0, m.ID, m.ID, sk, [3]uint16{}, nil, slot)})
		m.nextAttackAt = h.now.Add(mobAttackIdle + m.mods().attackDelayOffset())
		return
	}
	target := m.liveTarget(h.players)
	if target == nil {
		return
	}
	distance := DistanceXZ(pos[0], pos[2], target.PosX, target.PosZ)
	if distance > float32(m.def.Size)+float32(sk.AttackRange)+.3 {
		mods := m.mods()
		if !m.lua.hold {
			m.stepChase(h.now, h.mapID, target, distance, &mods, h.tick)
		}
		return
	}
	m.nextAttackAt = h.now.Add(mobAttackIdle + m.mods().attackDelayOffset())
	h.tick.Attacks = append(h.tick.Attacks, MobAttackAction{MobGlobalID: m.ID, MobID: m.MobID, Attack: m.Attack, TargetCharacterID: m.AggroTarget, Slot: slot, MobX: pos[0], MobZ: pos[2], SkillID: id})
}

type mobAICommand struct {
	kind    string
	mapID   uint16
	owner   uint32
	actor   *ai.Actor
	id      uint16
	count   int
	x, y, z float32
	mode    int
	spread  float32
	legacy  bool
}

func runMobAICommands(tick *MobCombatTick) {
	for _, mapID := range shardIDs() {
		s := shard(mapID)
		s.mu.Lock()
		commands := s.aiCommands
		s.aiCommands = nil
		s.aiCommandCost = 0
		s.aiPendingSummons = 0
		s.mu.Unlock()
		for _, cmd := range commands {
			s.mu.Lock()
			var owner *MapMob
			for i := range s.entities.Mobs {
				m := &s.entities.Mobs[i]
				if m.ID == cmd.owner && m.lua.actor == cmd.actor && m.lua.fault == nil {
					owner = m
					break
				}
			}
			if owner == nil {
				s.mu.Unlock()
				continue
			}
			switch cmd.kind {
			case "spawn":
				live := 0
				for _, mob := range s.entities.Mobs {
					if mob.aiSummoned && !mob.corpseGone {
						live++
					}
				}
				def := db.MustDefs().Mobs[cmd.id]
				for range min(cmd.count, max(4096-live, 0)) {
					x, z, found := cmd.x, cmd.z, false
					for range 10 {
						span := max(int(cmd.spread*2), 1)
						x, z = cmd.x+float32(rand.IntN(span))-cmd.spread, cmd.z+float32(rand.IntN(span))-cmd.spread
						if IsMapWalkable(mapID, x, z) {
							found = true
							break
						}
						if !cmd.legacy && IsMapWalkable(mapID, cmd.x, cmd.z) {
							x, z, found = cmd.x, cmd.z, true
							break
						}
					}
					if !found {
						continue
					}
					m := NewStaticMob(def, cmd.id, x, z, nil, 0)
					m.aiSummoned = true
					m.aiAttackType = uint8(cmd.mode)
					m.resetAI()
					m.PosY = cmd.y
					m.homeY = cmd.y
					if nav, e := loadMobNavigation(mapID); e == nil && nav != nil {
						if !m.placeOnNavigation(nav, mapID) {
							continue
						}
					}
					reused := false
					for i := range s.entities.Mobs {
						old := &s.entities.Mobs[i]
						if old.aiSummoned && old.CurrentHP == 0 && old.corpseGone {
							s.entities.Mobs[i] = m
							reused = true
							break
						}
					}
					if !reused {
						s.entities.Mobs = append(s.entities.Mobs, m)
					}
					tick.Moves = append(tick.Moves, entityMove{MapID: mapID, Kind: viewMob, ID: m.ID, Fresh: true, Changed: true, To: cellFor(m.PosX, m.PosZ), Enter: buildMobEnter(&m, true)})
				}
			case "remove":
				// MobRemove2 scans five cells around the requested position and
				// removes every matching live mob within 100 units. Its count
				// argument is unused in the original executable.
				for i := range s.entities.Mobs {
					m := &s.entities.Mobs[i]
					center := cellFor(cmd.x, cmd.z)
					if m.cell.x < center.x-5 || m.cell.x > center.x+5 || m.cell.z < center.z-5 || m.cell.z > center.z+5 {
						continue
					}
					if m.MobID != cmd.id || m.CurrentHP == 0 || DistanceXZ(m.PosX, m.PosZ, cmd.x, cmd.z) >= 100 {
						continue
					}
					m.CurrentHP = 0
					m.died(time.Now())
					m.respawnAt = time.Time{}
					if m.group != nil {
						m.group.alive--
						m.group = nil
					}
					tick.Restores = append(tick.Restores, mapPacket{mapID, m.PosX, m.PosZ, buildMobCurrentHP(m.ID, 0)})
				}
			case "portal":
				sacraGate := cmd.id == 1001 && isSacraGuard(owner)
				if !isInstanceZone(mapID) && !(sacraGate && (mapID == 65 || mapID == 66)) {
					break
				}
				if sacraGate && hasLivingSacraGuard(s.entities.Mobs) {
					break
				}
				for i := range s.entities.Portals {
					p := &s.entities.Portals[i]
					if p.PortalID == int32(cmd.id) {
						p.FactionOrPortalID = int32(cmd.mode)
					}
				}
			case "say_text", "say_index": // Packet publication is deferred below, without a shard lock.
			}
			s.mu.Unlock()
			if cmd.kind == "say_text" || cmd.kind == "say_index" {
				publishMobAISpeech(cmd)
			}
		}
	}
}

// Sacra's first-floor bosses each send the portal update on death. The
// summoned encounter pair must both be defeated in this instance.
func isSacraGuard(m *MapMob) bool {
	return m.aiSummoned && m.MobID >= 2464 && m.MobID <= 2467
}

func hasLivingSacraGuard(mobs []MapMob) bool {
	for i := range mobs {
		if mobs[i].CurrentHP > 0 && isSacraGuard(&mobs[i]) {
			return true
		}
	}
	return false
}

func recordMobAIMiss(mapID uint16, id, user uint32) {
	s := shard(mapID)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entities.Mobs {
		m := &s.entities.Mobs[i]
		if m.ID != id || m.CurrentHP == 0 {
			continue
		}
		h := &mobAIHost{mob: m, shard: s, mapID: mapID, now: time.Now(), tick: &MobCombatTick{}}
		if !m.ensureAI(h) {
			return
		}
		// A resolved miss is still an attack attempt. Acquire the attacker
		// even when this entry point was not preceded by SetMobAggro.
		if user != 0 && m.AggroTarget == 0 {
			m.AggroTarget, m.returningHome = user, false
			if m.aggro == nil {
				m.aggro = map[uint32]int64{}
			}
			if _, ok := m.aggro[user]; !ok {
				m.aggro[user] = 0
			}
		}
		if m.lua.battleType == 0 {
			m.lua.attacked = true
			m.lua.recentHit = true
			h.event("attacked", user)
		}
		return
	}
}
