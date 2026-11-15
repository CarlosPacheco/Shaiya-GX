package app

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/pvprank"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type pvpRewardView struct {
	confirm  int
	pending  proto.Opcode
	deadline float64
}

func (a *App) openPvPRewards() {
	a.pvpReward = pvpRewardView{}
	if a.pvpRewardUI != nil {
		a.pvpRewardUI.Reset()
	}
	if a.pvpRewardMessage != nil {
		a.pvpRewardMessage.Visible = false
	}
	c, _ := a.selfSlot()
	if c.Mode == 0 || c.Level < 16 {
		id := 5346
		if c.Mode != 0 {
			id = 5347
		}
		a.questMessage(a.msg.Get(id))
		a.shop.npcID = 0
		return
	}
	a.state.PvPRewards = proto.PvPRewards{}
	a.sendPvPReward(proto.PvPRewardStatus)
}

func (a *App) sendPvPReward(op proto.Opcode) {
	if a.world == nil {
		return
	}
	if err := a.world.Send(proto.NewWriter(op).Bytes()); err != nil {
		a.backToLogin("Unable to request PvP reward.")
		return
	}
	a.pvpReward.pending, a.pvpReward.deadline = op, a.clock+10
}

func (a *App) confirmedPvPReward(op proto.Opcode) {
	if op != a.pvpReward.pending {
		return
	}
	a.pvpReward.pending = 0
	if op == proto.PvPRewardStatus {
		return
	}
	v := a.state.LastPvPReward
	if v.OK && op == proto.PvPRewardKill {
		a.questMessage(a.msg.Format(5349, map[string]string{"b": fmt.Sprint(v.Value)}))
	} else if v.OK && op == proto.PvPRewardDeath {
		a.questMessage(a.msg.Format(53, map[string]string{"b": fmt.Sprint(v.Value)}))
	} else if !v.OK {
		a.questMessage(a.msg.Get(5350))
	}
}

func (a *App) pvpRewardState() ui.PvPRewardState {
	c, _ := a.selfSlot()
	r := a.state.PvPRewards
	threshold, points, _, killOK := pvprank.Reward(c.Mode, r.KillStage)
	deathThreshold, _, gold, deathOK := pvprank.Reward(c.Mode, r.DeathStage)
	state := ui.PvPRewardState{Description: a.msg.Get(5348), Busy: a.pvpReward.pending != 0,
		CanKill:  r.Known && killOK && a.state.Details.Kills >= threshold,
		CanDeath: r.Known && deathOK && a.state.Details.Deaths >= deathThreshold}
	if e := a.state.Entities[a.shop.npcID]; e != nil {
		def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
		state.Name, state.Portrait = def.Name, a.npcPortrait(def)
	}
	if !r.Known {
		return state
	}
	killStage, deathStage := min(r.KillStage, pvprank.RewardCount), min(r.DeathStage, pvprank.RewardCount)
	threshold, points, _, _ = pvprank.DisplayReward(c.Mode, killStage)
	_, _, gold, _ = pvprank.DisplayReward(c.Mode, deathStage)
	state.Reward = a.msg.Format(5353, map[string]string{"v": fmt.Sprint(killStage), "v2": fmt.Sprint(points)})
	state.Kills = fmt.Sprintf("%d / %d", a.state.Details.Kills, threshold)
	state.Deaths = fmt.Sprint(a.state.Details.Deaths)
	state.DeathLevel = fmt.Sprintf("%d %s / 31 %s", deathStage, a.msg.Get(5354), a.msg.Get(5354))
	state.Gold = fmt.Sprintf("%d %s", gold, a.msg.Get(5355))
	if threshold > 0 {
		state.KillProgress = float32(a.state.Details.Kills) / float32(threshold)
	}
	// Native 0x0051a09a multiplies the claim level by float32(1/31).
	state.DeathProgress = float32(deathStage) * float32(1.0/31)
	return state
}

func (a *App) updatePvPRewards(in *platform.Input, w, h int) bool {
	if a.pvpReward.pending != 0 && a.clock >= a.pvpReward.deadline {
		a.pvpReward.pending = 0
	}
	if a.pvpRewardMessage != nil && a.pvpRewardMessage.Visible {
		if result := a.pvpRewardMessage.Update(in, w, h); result != ui.MessageNone {
			claim := a.pvpReward.confirm
			a.pvpReward.confirm = 0
			if result == ui.MessageOK && a.pvpReward.pending == 0 {
				op := proto.PvPRewardKill
				if claim == 2 {
					op = proto.PvPRewardDeath
				}
				a.sendPvPReward(op)
			}
		}
		return true
	}
	claim, closed := a.pvpRewardUI.Update(in, w, h, a.pvpRewardState())
	if closed {
		if a.world != nil {
			if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
				a.backToLogin("Unable to close NPC conversation.")
			}
		}
		a.shop.npcID = 0
		return true
	}
	if claim != 0 && a.pvpReward.pending == 0 && a.pvpRewardMessage != nil {
		c, _ := a.selfSlot()
		r := a.state.PvPRewards
		stage := r.KillStage
		if claim == 2 {
			stage = r.DeathStage
		}
		count, points, _, ok := pvprank.Reward(c.Mode, stage)
		if ok {
			id := 5351
			if claim == 2 {
				id = 5352
			}
			a.pvpReward.confirm = claim
			a.pvpRewardMessage.OpenFull(a.msg.Format(id, map[string]string{"p": fmt.Sprint(count), "b": fmt.Sprint(points)}), true, a.font)
		}
	}
	x, y := a.pvpRewardUI.Origin(w, h)
	return panelPointer(in, x, y, ui.PvPRewardWidth, ui.PvPRewardHeight)
}

func (a *App) drawPvPRewards(w, h int) {
	x, y := a.pvpRewardUI.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.PvPRewardWidth, ui.PvPRewardHeight)
	a.pvpRewardUI.Draw(a.sprites, a.font, w, h, a.pvpRewardState())
}
