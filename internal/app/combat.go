package app

import (
	"fmt"
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Rapture's 0x0205 failure payload contains an explicit reason. Successful
// item and money packets retain their normal inventory handling.
func (a *App) confirmedPickup(p []byte) {
	if proto.Op(p) != proto.AddItem || len(p) != 4 || p[2] != 0 {
		return
	}
	message := "Unable to pick up that item."
	switch p[3] {
	case 0:
		message = "That item is reserved for another player."
	case 1:
		message = "Your inventory is full."
	case 2:
		message = "You do not need that quest item."
	case 3:
		message = "That item is no longer available."
	case 4:
		message = "That item is too far away."
	}
	ids := map[byte]int{0: 202, 1: 238, 3: 239, 4: 233}
	if id := ids[p[3]]; id != 0 && a.msg.Get(id) != "" {
		a.systemMessage(1, id, nil)
	} else {
		a.questMessage(message)
	}
}

// Delivery controls: click a player or mob to select;
// Double click or F requests auto-attack. The server owns range, cadence and damage.
func (a *App) updateCombat(in *platform.Input, w, h int) {
	if a.wv == nil {
		return
	}
	if a.state.Self.Dead {
		a.wv.mobClick = mobClick{}
		for _, r := range in.Chars {
			if (r == 'r' || r == 'R') && a.clock >= a.wv.reviveAt {
				if err := a.world.Send(proto.RevivePacket()); err != nil {
					a.backToLogin("Unable to request resurrection.")
					return
				}
				a.wv.reviveAt = a.clock + 3
			}
		}
		return
	}
	if in.HasKey(platform.KeySelfTarget) {
		a.selectSelf()
		return
	}
	id := a.pickGroundModel(in, w, h)
	if id != 0 {
		a.wv.mobClick = mobClick{}
		a.approachLoot(id)
		in.Clicked = false // This click issued the approach; do not cancel it in movement.
		return
	}
	if id := a.pickNPC(in, w, h); id != 0 {
		// A click targets the NPC; a double click walks to it and talks.
		if a.wv.mobClick.double(id, in.MouseX, in.MouseY, a.clock) {
			a.openNPC(id)
		} else {
			a.selectNPC(id)
		}
		in.Clicked = false
		return
	}
	if in.Clicked || in.HasKey(platform.KeyTarget) || in.HasKey(platform.KeyPreviousTarget) {
		target := a.pickMob(in, w, h)
		if in.HasKey(platform.KeyPreviousTarget) {
			target = a.state.PreviousTarget
			if e := a.state.Entities[target]; target != a.state.SelfID && (e == nil || e.Dead) {
				return
			}
		}
		if e := a.state.Entities[target]; e != nil && e.Kind == game.KindUser && target != a.state.Target {
			if old := a.state.Entities[a.state.Target]; old != nil && old.Kind == game.KindMob {
				if err := a.world.Send(proto.StopAttackPacket()); err != nil {
					a.backToLogin("Unable to stop attack.")
					return
				}
			}
		}
		if e := a.state.Entities[target]; target != a.state.Target && e != nil && e.Kind == game.KindMob {
			if err := a.world.Send(proto.TargetMobPacket(target)); err != nil {
				a.backToLogin("Unable to select target.")
				return
			}
		}
		if target != a.state.Target {
			a.state.PreviousTarget = a.state.Target
		}
		a.state.Target = target
		if in.Clicked {
			a.wv.travel = false
			if target == 0 {
				a.moveToGround(in, w, h)
			}
		}
		if in.Clicked {
			if a.wv.mobClick.double(target, in.MouseX, in.MouseY, a.clock) && !a.attackTarget() {
				return
			}
		} else {
			a.wv.mobClick = mobClick{}
		}
	}
	for _, r := range in.Chars {

		if r == 'h' || r == 'H' {
			if !a.pickupNearest() {
				return
			}
			continue
		}
		if r != 'f' && r != 'F' {
			continue
		}
		a.wv.mobClick = mobClick{}
		if !a.attackTarget() {
			return
		}
	}
}

func (a *App) pickupNearest() bool {
	id := uint32(0)
	// Native 0x43bfa0 nearest-item action searches within 30 units.
	best := float32(30 * 30)
	for k, it := range a.state.GroundItems {
		dx, dz := it.X-a.wv.pos[0], it.Z-a.wv.pos[2]
		d := dx*dx + dz*dz
		if d < best || d == best && (id == 0 || k < id) {
			id, best = k, d
		}
	}
	if id != 0 {
		return a.approachLoot(id)
	}
	return true
}

func (a *App) attackTarget() bool {
	if !a.attackMovementAllowed() || a.clock < a.wv.attackMoveUntil {
		return true
	}
	if a.state.Self.Dead {
		a.wv.sitAttack = 0
		return true
	}
	e := a.state.Entities[a.state.Target]
	if e == nil || e.Dead || e.Kind != game.KindMob && (e.Kind != game.KindUser || e.ID == a.state.SelfID) {
		return true
	}
	if a.friendlyMob(e) {
		a.wv.approachTarget = 0
		a.wv.approachSkill = nil
		a.systemMessage(17, 317, nil)
		return true
	}
	if a.state.Self.Sitting || a.wv.sitPending || a.wv.self != nil && a.wv.self.PlayingAction(11) {
		a.wv.sitAttack = e.ID
		if a.state.Self.Sitting && !a.wv.sitPending {
			return a.requestPosture(proto.MotionStandUp)
		}
		return true
	}
	a.wv.approachTarget, a.wv.approachSkill = 0, nil
	a.wv.approachNPC = false
	a.wv.travel, a.wv.autoRun = false, false
	if !a.faceRequestedTarget(e.ID) {
		return false
	}
	delta := render.V3{e.X - a.wv.pos[0], 0, e.Z - a.wv.pos[2]}
	if delta.Len() > a.attackReach() {
		a.wv.autoRun = false
		a.wv.approachTarget, a.wv.approachLoot = e.ID, false
		a.wv.approachSkill = nil
		return a.updateActionApproach(&platform.Input{})
	}
	if !a.stopForAttack() {
		return false
	}
	// Rapture decides whether a player is hostile (faction, PvP map, duel,
	// guild duel). A client-side duel-only check would block normal PvP.
	if e != nil && e.Kind == game.KindUser && e.ID != a.state.SelfID && !e.Dead {
		if err := a.world.Send(proto.UserAttackPacket(e.ID)); err != nil {
			a.backToLogin("Unable to attack duel opponent.")
			return false
		}
		return true
	}
	if e != nil && e.Kind == game.KindMob && !e.Dead {
		if err := a.world.Send(proto.MobAttackPacket(e.ID)); err != nil {
			a.backToLogin("Unable to send attack request.")
			return false
		}
	}
	return true
}

func (a *App) drawCombat(w, h int) {
	a.drawCombatNumbers(w, h)
	s := a.state
	if a.clock < a.combatNoticeUntil {
		a.font.Draw(a.sprites, a.combatNotice, w/2-45, h/2-80, white, 0)
	}
	view := ui.HUDState{ViewportW: w, ViewportH: h,
		Portrait: a.hudPortrait(w, h),
		HP:       ui.Resource{Current: s.Pools.HP, Max: s.Details.MaxHP},
		MP:       ui.Resource{Current: s.Pools.MP, Max: s.Details.MaxMP},
		SP:       ui.Resource{Current: s.Pools.SP, Max: s.Details.MaxSP},
		Exp:      s.Details.Exp, PrevExp: s.Details.PrevExp, NextExp: s.Details.NextExp,
	}
	if s.Blessing.Known && s.Blessing.Country < 2 && s.Blessing.Maximum > 0 {
		view.BlessCount, view.BlessMax, view.BlessKnown = s.Blessing.Count, s.Blessing.Maximum, true
		view.BlessRemainingMS, view.BlessCountry, view.Clock = s.Blessing.RemainingMS, s.Blessing.Country, a.clock
	}
	for _, slot := range s.Slots {
		if slot.ID == s.SelfID {
			view.Name, view.Level = slot.Name, slot.Level
			break
		}
	}
	if s.Target == s.SelfID && s.SelfID != 0 {
		view.TargetName, view.TargetHP, view.TargetKnown = view.Name, view.HP, true
	} else if e := s.Entities[s.Target]; e != nil && e.Kind == game.KindMob {
		view.TargetName = fmt.Sprintf("Target %d", e.TypeID)
		if def, err := a.assets.MonsterDefinition(e.TypeID); err == nil {
			view.TargetName, view.TargetHP.Max = def.Name, def.HP
			view.TargetColor = ui.RGBColor(ui.MobNameRGB(def.Level, view.Level))
			view.TargetElement = def.Element
		} else if !a.assets.Missing[err.Error()] {
			log.Printf("target definition: %v", err)
			a.assets.Missing[err.Error()] = true
		}
		view.TargetHP.Current = uint32(max(e.HP, 0))
		view.TargetKnown = e.HealthKnown
	} else if e != nil && e.Kind == game.KindNpc {
		a.loadNPCs()
		view.TargetName = a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}].Name
		view.TargetColor = [4]float32{1, .96, .3, 1}
		view.TargetHP, view.TargetKnown, view.TargetNoValue = ui.Resource{Current: 1, Max: 1}, true, true
	} else if e != nil && e.Kind == game.KindUser && e.Shape != nil {
		view.TargetName = e.Shape.Name
		if member := s.Party.Member(e.ID); member != nil {
			view.TargetHP = ui.Resource{Current: member.HP, Max: member.MaxHP}
			view.TargetKnown = true
		}
		if h, ok := s.UserHealth[e.ID]; ok && !view.TargetKnown {
			view.TargetHP = ui.Resource{Current: h.HP, Max: h.Max}
			view.TargetKnown = true
		}
	}
	a.hud.Draw(a.sprites, a.font, view)
	a.drawTargetEffects(w, h)
	if a.wv != nil && a.wv.self != nil {
		left, total, active := a.wv.self.CastTiming()
		if active {
			// Native 0x004c02d0 projects player position plus half a world
			// unit along camera-right; this is not a fixed HUD element.
			if x, y, ok := castBarOrigin(a.wv.pos, a.scene3d.View, a.scene3d.Proj, w, h); ok {
				a.hud.DrawCast(a.sprites, a.white, x, y, left, total)
			}
		}
	}
	a.bottom.QuestOpen, a.bottom.GuildOpen = a.questLog.open, a.guild.open
	a.bottom.Draw(a.sprites, a.font, a.white, w, h, view, a.inventory.open, a.skills.open, a.status.open, a.subActionsOpen)
	a.drawBlessingTooltip(w, h)
	a.drawBlessingBanner(w, h)
}

// selectNPC makes an NPC the current target (the NPC target frame shows its name).
func (a *App) selectNPC(id uint32) {
	if a.state.Target != id {
		if old := a.state.Entities[a.state.Target]; old != nil && old.Kind == game.KindMob && a.world != nil {
			if err := a.world.Send(proto.StopAttackPacket()); err != nil {
				a.backToLogin("Unable to stop attack.")
				return
			}
		}
		a.state.PreviousTarget = a.state.Target
	}
	a.state.Target = id
	if a.wv != nil {
		a.wv.approachSkill = nil
	}
}

func (a *App) selectSelf() {
	if a.state.Target != a.state.SelfID {
		if old := a.state.Entities[a.state.Target]; old != nil && old.Kind == game.KindMob && a.world != nil {
			if err := a.world.Send(proto.StopAttackPacket()); err != nil {
				a.backToLogin("Unable to stop attack.")
				return
			}
		}
		a.state.PreviousTarget = a.state.Target
	}
	a.state.Target = a.state.SelfID
	if a.wv != nil {
		a.wv.approachTarget = 0
		a.wv.approachSkill = nil
		a.wv.mobClick = mobClick{}
	}
}

// castBarOrigin follows 0x004c02d0 and the native FISTP rounding of screen pixels.
func castBarOrigin(pos render.V3, view, proj render.M4, w, h int) (int, int, bool) {
	// The native actor origin is .9 above GX feet. Native camera-right
	// comes from the horizontal camera direction, independent of pitch.
	right := (render.V3{view[0], 0, view[8]}).Norm()
	p := pos.Add(render.V3{0, .9, 0}).Add(right.Scale(.5))
	vp := view.Mul(proj)
	cw := p[0]*vp[3] + p[1]*vp[7] + p[2]*vp[11] + vp[15]
	z := p[0]*vp[2] + p[1]*vp[6] + p[2]*vp[10] + vp[14]
	if cw <= 0 || z < 0 || z > cw {
		return 0, 0, false
	}
	x := (1 + (p[0]*vp[0]+p[1]*vp[4]+p[2]*vp[8]+vp[12])/cw) * float32(w) * .5
	y := (1 - (p[0]*vp[1]+p[1]*vp[5]+p[2]*vp[9]+vp[13])/cw) * float32(h) * .5
	return int(math.RoundToEven(float64(x))), int(math.RoundToEven(float64(y))), true
}
