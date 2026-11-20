package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Union (raid) flows of the original UI, from the popup builder 0x47ebe0 and
// the dispatcher 0x48e050:
//
//	own portrait  (mode 1): Create Union 6500 | Leave 6501, Dismantle 6502 (leader),
//	                         Config 6503
//	union slot    (mode 2): Move Group 6506, Delegate leader 6507, sub-leader 6508,
//	                         Kick 6509, Whisper 6510, by role
//
// The configuration window (union_party.png) opens from Create or Config; the
// compact union window is shown while DAT_021fd8c8 ("Open Union") is set, which
// a conversion turns on.
const (
	contextClassic = iota
	contextPortrait
	contextUnion
)

// unionSlots is the slot -> member map the windows show; while a raid is
// being created the party members fill the first slots in roster order.
func (a *App) unionSlots() map[uint16]uint32 {
	p := &a.state.Party
	if p.Raid {
		return p.RaidSlots
	}
	slots := make(map[uint16]uint32, len(p.Members))
	for i, m := range p.Members {
		slots[uint16(i)] = m.ID
	}
	return slots
}

// raidGridShown: the party2 raid window; it stays under the dialog so the
// transparency slider can be judged live.
func (a *App) raidGridShown() bool {
	return a.state.Party.Raid && a.partyUI != nil && a.partyUI.RaidGridVisible()
}

func (a *App) unionConfigOpen() bool {
	p := &a.state.Party
	return p.Raid && !a.party.raidHidden || !p.Raid && a.party.creating && len(p.Members) > 0
}

// openUnionCreate is menu entry 0x1964. The loot choice is a placeholder for
// now: Group Loot is preselected.
func (a *App) openUnionCreate() {
	a.party.creating, a.party.raidHidden = true, true
	a.partyUI.ResetRaidSettings()
}

// openUnionConfig is 0x1967: the window with the confirmed settings.
func (a *App) openUnionConfig() {
	a.party.raidHidden = false
	a.partyUI.ResetRaidSettings()
}

func (a *App) closeUnionConfig() {
	a.party.creating, a.party.raidHidden = false, true
	a.partyUI.ResetRaidSettings()
}

// unionRole: 0 leader, 1 assistant, 2 member (DAT_021fd7f0).
func (a *App) unionRole() int {
	p := &a.state.Party
	switch a.state.SelfID {
	case p.Leader:
		return 0
	case p.RaidSubLeader:
		return 1
	}
	return 2
}

// portraitActions is popup mode 1, the own-portrait menu.
func (a *App) portraitActions() []contextAction {
	p := &a.state.Party
	if !p.Raid {
		if p.Leader != a.state.SelfID {
			return nil // only the party leader can create a raid
		}
		return []contextAction{{label: "Create union"}}
	}
	out := []contextAction{{label: "Leave union", packet: proto.PartyLeavePacket()}}
	if a.unionRole() == 0 {
		out = append(out, contextAction{label: "Dismantle union", packet: proto.RaidDisbandPacket()})
	}
	return append(out, contextAction{label: "Config union"})
}

// unionActions is popup mode 2, a slot of either union window.
func (a *App) unionActions(slot int) []contextAction {
	p := &a.state.Party
	if !p.Raid || slot < 0 || slot >= 30 {
		return nil
	}
	id := p.RaidSlots[uint16(slot)]
	move := contextAction{label: "Move group", slot: slot}
	member := p.Member(id)
	if id == 0 || member == nil {
		return []contextAction{move}
	}
	whisper := contextAction{label: "Union whisper", name: member.Name}
	switch a.unionRole() {
	case 0:
		out := []contextAction{move}
		if id == a.state.SelfID {
			return out
		}
		out = append(out, contextAction{label: "Delegate leader", packet: proto.RaidLeaderPacket(id)})
		if id != p.RaidSubLeader {
			out = append(out, contextAction{label: "Make assistant", packet: proto.RaidSubLeaderPacket(id)})
		}
		return append(out, contextAction{label: "Remove from raid", packet: proto.RaidExpelPacket(id)}, whisper)
	case 1:
		if id == a.state.SelfID {
			return []contextAction{move}
		}
		return []contextAction{move, whisper}
	}
	return []contextAction{whisper}
}

func (a *App) menuActions(c *playerContext) []contextAction {
	switch c.mode {
	case contextPortrait:
		return a.portraitActions()
	case contextUnion:
		return a.unionActions(c.slot)
	}
	return a.contextActions(c.id)
}

// unionMenuAt is the slot menu trigger: a right click over a union window.
func (a *App) unionSlotAt(in *platform.Input, w, h int) (int, bool) {
	p := &a.state.Party
	if a.partyUI == nil || !p.Raid {
		return -1, false
	}
	if !a.party.raidHidden && a.partyUI.RaidInside(w, h, in.MouseX, in.MouseY) {
		return a.partyUI.RaidSlotAt(w, h, in.MouseX, in.MouseY), true
	}
	if a.raidGridShown() {
		if slot := a.partyUI.RaidGridSlotAt(w, h, in.MouseX, in.MouseY); slot >= 0 {
			return slot, true
		}
	}
	if slot := a.partyUI.RaidCompactSlotAt(w, h, in.MouseX, in.MouseY, a.state.SelfID, p.RaidSlots); slot >= 0 {
		return slot, true
	}
	return -1, false
}

// finishMove is the left click of "Move Group" (0x5986c0): swap two slots.
func (a *App) finishMove(slot int) {
	from := a.party.moveSlot
	a.party.moving = false
	if slot < 0 || slot == from || a.world == nil {
		return
	}
	if err := a.world.Send(proto.RaidSwapPacket(uint32(from), uint32(slot))); err != nil {
		a.backToLogin("Unable to move the group.")
	}
}

// drainPartyNotices prints the SysMsg lines the party/raid packets queued.
func (a *App) drainPartyNotices() {
	p := &a.state.Party
	for _, n := range p.Notices {
		a.systemMessage(7, n.ID, map[string]string{"p": n.Name})
	}
	p.Notices = p.Notices[:0]
}

// raidFormed / raidEnded keep the window flags in step with the roster:
// a conversion opens the union window (DAT_021fd8c8 = 1) and no config window.
func (a *App) syncUnionState(wasRaid bool) {
	if a.partyUI == nil {
		return
	}
	raid := a.state.Party.Raid
	switch {
	case raid && !wasRaid:
		// the 30-cell raid window appears; the configuration dialog stays closed
		a.party.creating, a.party.raidHidden, a.party.moving = false, true, false
		a.partyUI.ResetRaidSettings()
	case !raid && wasRaid:
		a.party.raidHidden, a.party.moving = true, false
		a.partyUI.ResetRaidSettings()
	}
	if len(a.state.Party.Members) == 0 {
		a.party.creating = false
	}
}
