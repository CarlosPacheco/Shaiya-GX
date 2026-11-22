package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"slices"
	"strings"
)

type queuedSkill struct {
	packet    []byte
	key       sdata.SkillKey
	selection uint32
	expires   float64
}

// Keep only the latest intent. Stop basic auto-repeat without clearing its
// recovery; otherwise it would consume every shared slot before the queued spell.
func (a *App) queueSkill(p []byte) {
	number := proto.NewReader(p).U8()
	k, ok := a.state.Skills[number]
	if !ok {
		return
	}
	if a.skills.queuedSkill == nil && a.world != nil {
		if err := a.world.Send(proto.StopAttackPacket()); err != nil {
			a.backToLoginID(12188)
			return
		}
	}
	expires := a.clock + 5
	if a.wv != nil && a.wv.self != nil {
		if left, _, active := a.wv.self.CastTiming(); active {
			expires = max(expires, a.clock+float64(left)+1)
		}
	}
	a.skills.queuedSkill = &queuedSkill{packet: slices.Clone(p), key: sdata.SkillKey{ID: k.ID, Level: k.Level}, selection: a.state.Target, expires: expires}
	if a.wv != nil {
		a.wv.approachTarget = 0
		a.wv.approachSkill = nil
		a.wv.travel = false
		a.wv.autoRun = false
	}
}
func (a *App) updateQueuedSkill(in *platform.Input) {
	q := a.skills.queuedSkill
	if q == nil {
		return
	}
	reader := proto.NewReader(q.packet)
	reader.U8()
	recipient := reader.U32()
	selectionChanged := a.state.Target != q.selection && recipient != 0 && recipient != a.state.SelfID
	if a.state.Self.Dead || a.state.Self.Mounted || selectionChanged || a.clock > q.expires || in.SuppressAutoRun || in.Clicked || in.Space || in.HasKey(platform.KeySpace) || in.HasKey(platform.KeyNumLock) || strings.ContainsAny(in.Down, "WSADQE^v<>") {
		a.skills.queuedSkill = nil
		return
	}
	if a.world == nil || !a.attackMovementAllowed() || !a.skillActionReady(q.key.ID, a.skillDefs[q.key]) {
		return
	}
	a.skills.queuedSkill = nil
	r := proto.NewReader(q.packet)
	number, target := r.U8(), r.U32()
	k, ok := a.state.Skills[number]
	if !ok || (sdata.SkillKey{ID: k.ID, Level: k.Level}) != q.key || a.skills.cooldowns[k.ID] > a.clock {
		return
	}
	d := a.skillDefs[q.key]
	if !a.validSkillTarget(d, target) || !a.skillResourcesAvailable(d) {
		return
	}
	if a.approachSkillPacket(q.packet) || !a.prepareSkillMovement(q.packet) {
		return
	}
	if err := a.world.Send(q.packet); err != nil {
		a.backToLoginID(12189)
		return
	}
	a.requestedSkillPresentation(q.packet)
}
