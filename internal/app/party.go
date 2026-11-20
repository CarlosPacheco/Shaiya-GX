package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type partyView struct {
	inviteSerial, replySerial uint64
	raidErrorSerial           uint64
	raidHidden                bool // the configuration window is closed
	creating                  bool // Create Union: the window shows the party before converting
	moving                    bool // Move Group picked a slot, waiting for the target
	moveSlot                  int
	expires                   float64
	inviteName                string // last name typed in an invite command, for refusals of players on other maps
}

// Native SysMsg ids for party and raid refusals, indexed by the server's
// result code. They go to the chat log like every other notification.
var partyReplyMessages = [...]int{206, 304, 345, 346, 347, 348}
var raidErrorMessages = map[uint32]int{0: 206, 1: 304, 2: 6540, 3: 346, 4: 347, 5: 348, 10: 6522, 11: 6523, 12: 6535, 13: 6536, 14: 6538}

// partyNotice writes a native SysMsg line to the chat (never a screen overlay).
func (a *App) partyNotice(id int, name string) {
	a.systemMessage(3, id, map[string]string{"p": name})
}

// invitedNotice tells the inviter whom the request went to (native SysMsg 203).
func (a *App) invitedNotice(name string) {
	if name != "" {
		a.systemMessage(7, 203, map[string]string{"p": name}) // 0x48e050: FUN_0041fce0(7, 0xcb)
	}
}

// partyTargetName is the refused player's name: nearby players are known by
// id, global invitations by the name just typed.
func (a *App) partyTargetName(id uint32) string {
	if e := a.state.Entities[id]; e != nil && e.Shape != nil && e.Shape.Name != "" {
		return e.Shape.Name
	}
	return a.party.inviteName
}

func (a *App) updatePartyInvite(in *platform.Input, w, h int) bool {
	if a.partyMessage == nil || a.state == nil {
		return false
	}
	p := &a.state.Party
	if p.InviteSerial != a.party.inviteSerial {
		a.party.inviteSerial = p.InviteSerial
		if p.InviteSerial == 0 {
			a.partyMessage.Visible = false
			return false
		}
		name := p.InviteName
		if name == "" {
			name = fmt.Sprintf("Player %d", p.InviteID)
		}
		kind := "party"
		if p.RaidInvite {
			kind = "raid"
		}
		a.partyMessage.Open(fmt.Sprintf("%s invites you to a %s. Join?", name, kind), true)
		a.party.expires = a.clock + 30 // Rapture's invite window; expires only the prompt.
	}
	if p.RaidErrorSerial != a.party.raidErrorSerial {
		a.party.raidErrorSerial = p.RaidErrorSerial
		id, ok := raidErrorMessages[p.RaidError]
		if !ok {
			id = 304
		}
		a.partyNotice(id, a.party.inviteName)
	}
	if p.ReplySerial != a.party.replySerial {
		a.party.replySerial = p.ReplySerial
		id := 304
		if int(p.Reply.Result) < len(partyReplyMessages) {
			id = partyReplyMessages[p.Reply.Result]
		}
		a.partyNotice(id, a.partyTargetName(p.Reply.ID))
	}
	if !a.partyMessage.Visible {
		return false
	}
	if len(p.Members) > 0 || a.clock >= a.party.expires {
		a.partyMessage.Visible = false
		return false
	}
	answer := a.partyMessage.Update(in, w, h)
	if answer != ui.MessageNone {
		packet := proto.PartyAnswerPacket(answer == ui.MessageOK, a.state.SelfID)
		if p.RaidInvite {
			packet = proto.RaidAnswerPacket(answer == ui.MessageOK, a.state.SelfID)
		}
		if err := a.world.Send(packet); err != nil {
			a.backToLogin("Unable to answer party invitation.")
		}
	}
	return true
}
func (a *App) updatePartyPointer(in *platform.Input, w, h int) bool {
	if a.partyUI == nil {
		return false
	}
	p := &a.state.Party
	if p.Raid || a.party.creating && len(p.Members) > 0 {
		return a.updateUnion(in, w, h)
	}
	if len(p.Members) == 0 {
		return false
	}
	id, captured := a.partyUI.Update(in, w, h, p.Members)
	if id != 0 {
		a.state.Target = id
	}
	return captured
}

// updateUnion drives the raid window (the 30-member union_party.png, on top)
// and the side list; the list never hides. While a raid is only being
// created the party list stays as it is and the window shows the party.
func (a *App) updateUnion(in *platform.Input, w, h int) bool {
	p := &a.state.Party
	if a.party.moving && (in.HasKey(platform.KeyEscape) || !p.Raid) {
		a.party.moving = false
	}
	slots := a.unionSlots()
	var captured bool
	if a.unionConfigOpen() {
		a.syncUnionDraft()
		controls := a.party.creating || p.Leader == a.state.SelfID
		id, action, c := a.partyUI.UpdateRaid(in, w, h, slots, controls)
		captured = c
		if id != 0 {
			a.state.Target = id
		}
		if in.Clicked && a.party.moving && c && p.Raid {
			a.finishMove(a.partyUI.RaidSlotAt(w, h, in.MouseX, in.MouseY))
		}
		switch action {
		case ui.RaidClose:
			a.closeUnionConfig()
		case ui.RaidApply:
			if !a.applyUnionConfig() {
				return true
			}
		}
		if captured {
			return true
		}
	}
	if p.Raid {
		if a.raidGridShown() {
			id, c := a.partyUI.UpdateRaidGrid(in, w, h, slots)
			if id != 0 {
				a.state.Target = id
			}
			if in.Clicked && a.party.moving && c {
				a.finishMove(a.partyUI.RaidGridSlotAt(w, h, in.MouseX, in.MouseY))
			}
			if c {
				return true
			}
		}
		id, c := a.partyUI.UpdateRaidCompact(in, w, h, a.state.SelfID, slots)
		if id != 0 {
			a.state.Target = id
		}
		if in.Clicked && a.party.moving && c {
			a.finishMove(a.partyUI.RaidCompactSlotAt(w, h, in.MouseX, in.MouseY, a.state.SelfID, slots))
		}
		return captured || c
	}
	// Creating: the ordinary party list keeps working underneath.
	id, c := a.partyUI.Update(in, w, h, p.Members)
	if id != 0 {
		a.state.Target = id
	}
	return captured || c
}

// syncUnionDraft keeps the window's draft in step with the confirmed state.
func (a *App) syncUnionDraft() {
	p := &a.state.Party
	if a.party.creating {
		a.partyUI.SetRaidSettings(1, false)
		return
	}
	a.partyUI.SetRaidSettings(p.RaidDivision, p.RaidAutoJoin)
}

// applyUnionConfig is the OK button (0x48e050 case 4): a conversion sends the
// chosen settings, a raid sends what changed.
func (a *App) applyUnionConfig() bool {
	defer a.saveUILayout() // transparency and list options persist
	p := &a.state.Party
	division, auto := a.partyUI.RaidSettings()
	if a.party.creating {
		a.partyUI.ApplyRaidViewSettings()
		if err := a.world.Send(proto.RaidConvertPacket(1, auto, division)); err != nil {
			a.backToLogin("Unable to create the union.")
			return true
		}
		a.closeUnionConfig()
		return true
	}
	a.partyUI.ApplyRaidViewSettings()
	if p.Leader == a.state.SelfID {
		if division != p.RaidDivision {
			if err := a.world.Send(proto.RaidDivisionPacket(division)); err != nil {
				a.backToLogin("Unable to update raid settings.")
				return true
			}
		}
		if auto != p.RaidAutoJoin {
			if err := a.world.Send(proto.RaidAutoJoinPacket(auto)); err != nil {
				a.backToLogin("Unable to update raid settings.")
				return true
			}
		}
	}
	a.closeUnionConfig()
	return true
}
func (a *App) drawParty(w, h int) {
	if a.partyUI == nil {
		return
	}
	// Self resources are delivered separately from remote party pool updates.
	members := append([]proto.PartyMember(nil), a.state.Party.Members...)
	for i := range members {
		if members[i].ID == a.state.SelfID {
			members[i] = a.state.SelfPartyMember()
		}
	}
	a.prepareMemberBuffs()
	defer a.drawMemberBuffTooltip()
	if a.state.Party.Raid {
		a.partyUI.DrawRaidCompact(a.sprites, a.font, w, h, a.state.SelfID, members, a.state.Party.RaidSlots, a.state.Party.Leader, a.state.Target)
		a.partyUI.DrawRaidInfo(a.sprites, a.font, w, h, a.raidInfoLines(), a.lastMouseX, a.lastMouseY, float32(a.frameDT))
		if a.raidGridShown() {
			moveSlot := -1
			if a.party.moving {
				moveSlot = a.party.moveSlot
			}
			a.partyUI.DrawRaidGrid(a.sprites, a.font, w, h, members, a.state.Party.RaidSlots, a.state.Party.Leader, a.state.Party.RaidSubLeader, a.state.Target, moveSlot)
		}
		if a.unionConfigOpen() {
			a.syncUnionDraft()
			a.partyUI.DrawRaid(a.sprites, a.font, w, h, members, a.state.Party.RaidSlots, a.state.Party.Leader, a.state.Target)
		}
	} else {
		a.partyUI.Draw(a.sprites, a.font, w, h, members, a.state.Party.Leader, a.state.Target)
		if a.unionConfigOpen() {
			a.syncUnionDraft()
			a.partyUI.DrawRaid(a.sprites, a.font, w, h, members, a.unionSlots(), a.state.Party.Leader, a.state.Target)
		}
	}
}

// partyGlobalInvite is /party <name>: the click invitation without range or
// map limits. The server refuses offline players, GMs and the other faction.
func (a *App) partyGlobalInvite(fields []string) bool {
	if len(fields) != 2 {
		return false
	}
	a.party.inviteName = fields[1]
	if a.state.Party.Raid {
		// Inside a raid it is the plain raid invitation, as /invite.
		if err := a.world.Send(proto.RaidNamePacket(proto.RaidInviteName, fields[1])); err != nil {
			a.backToLogin("Unable to send raid invitation.")
			return true
		}
		a.invitedNotice(fields[1])
		return true
	}
	if err := a.world.Send(proto.PartyGlobalInvitePacket(fields[1])); err != nil {
		a.backToLogin("Unable to send party invitation.")
		return true
	}
	a.invitedNotice(fields[1])
	return true
}

// Slash commands share the same native requests as the player-context menu.
// They never add/remove members before server confirmation.
func (a *App) partyCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}
	command := strings.ToLower(fields[0])
	if strings.HasPrefix(command, "/raid") {
		return a.raidCommand(fields)
	}
	if command == "/party" {
		return a.partyGlobalInvite(fields)
	}
	if command != "/invite" && command != "/leave" && command != "/kick" && command != "/leader" {
		return false
	}
	name := ""
	if len(fields) > 1 {
		name = strings.Join(fields[1:], " ")
	}
	var packet []byte
	if command == "/invite" && name != "" {
		a.party.inviteName = name
	}
	switch command {
	case "/invite":
		if a.state.Party.Raid && name != "" {
			packet = proto.RaidNamePacket(proto.RaidInviteName, name)
		} else if a.state.Party.Raid && a.state.Target != 0 {
			packet = proto.RaidInvitePacket(a.state.Target)
		} else if name != "" {
			packet = proto.PartyInviteNamePacket(name)
		} else if a.state.Target != 0 {
			packet = proto.PartyInvitePacket(a.state.Target)
		}
	case "/leave":
		packet = proto.PartyLeavePacket()
	case "/kick", "/leader":
		id := a.state.Target
		if name != "" {
			id = 0
			for _, m := range a.state.Party.Members {
				if strings.EqualFold(m.Name, name) {
					id = m.ID
					break
				}
			}
		}
		if a.state.Party.Leader != a.state.SelfID || id == 0 || id == a.state.SelfID || a.state.Party.Member(id) == nil {
			id := 345 // only the leader may do this
			if a.state.Party.Leader == a.state.SelfID {
				id = 347 // no such party member
			}
			a.partyNotice(id, name)
			return true
		}
		if a.state.Party.Raid && command == "/kick" {
			packet = proto.RaidExpelPacket(id)
		} else if a.state.Party.Raid {
			packet = proto.RaidLeaderPacket(id)
		} else if command == "/kick" {
			packet = proto.PartyExpelPacket(id)
		} else {
			packet = proto.PartyLeaderPacket(id)
		}
	}
	if packet == nil {
		a.partyNotice(347, name)
		return true
	}
	if err := a.world.Send(packet); err != nil {
		a.backToLogin("Unable to send party request.")
		return true
	}
	if command == "/invite" {
		if name == "" {
			name = a.chatEntityName(a.state.Target)
		}
		a.invitedNotice(name)
	}
	return true
}

// Raid commands only send requests. Membership and settings change on ACK.
func (a *App) raidCommand(fields []string) bool {
	p := &a.state.Party
	command := strings.ToLower(fields[0])
	arg := strings.Join(fields[1:], " ")
	var packet []byte
	switch command {
	case "/raidwindow":
		if p.Raid {
			if a.party.raidHidden {
				a.openUnionConfig()
			} else {
				a.closeUnionConfig()
			}
			return true
		}
	case "/raid":
		if len(p.Members) > 1 && !p.Raid && p.Leader == a.state.SelfID {
			packet = proto.RaidConvertPacket(1, false, 1)
		}
	case "/raidinvite":
		if arg != "" {
			packet = proto.RaidNamePacket(proto.RaidInviteName, arg)
		} else if a.state.Target != 0 {
			packet = proto.RaidInvitePacket(a.state.Target)
		}
	case "/raidjoin":
		if arg != "" {
			packet = proto.RaidNamePacket(proto.RaidAutoJoinName, arg)
		}
	case "/raidauto":
		if p.Raid && p.Leader == a.state.SelfID && (arg == "on" || arg == "off") {
			packet = proto.RaidAutoJoinPacket(arg == "on")
		}
	case "/raiddivision":
		if n, err := strconv.ParseUint(arg, 10, 32); err == nil && p.Raid && p.Leader == a.state.SelfID {
			packet = proto.RaidDivisionPacket(uint32(n))
		}
	case "/raidsub":
		id := a.state.Target
		if arg != "" {
			id = 0
			for _, m := range p.Members {
				if strings.EqualFold(m.Name, arg) {
					id = m.ID
					break
				}
			}
		}
		if p.Raid && p.Leader == a.state.SelfID && id != a.state.SelfID && p.Member(id) != nil {
			packet = proto.RaidSubLeaderPacket(id)
		}
	case "/raidswap":
		if len(fields) == 3 && p.Raid && (p.Leader == a.state.SelfID || p.RaidSubLeader == a.state.SelfID) {
			first, e1 := strconv.ParseUint(fields[1], 10, 8)
			second, e2 := strconv.ParseUint(fields[2], 10, 8)
			if e1 == nil && e2 == nil && first < 30 && second < 30 {
				packet = proto.RaidSwapPacket(uint32(first), uint32(second))
			}
		}
	case "/raiddisband":
		if p.Raid && p.Leader == a.state.SelfID {
			packet = proto.RaidDisbandPacket()
		}
	default:
		return false
	}
	if packet == nil {
		a.partyNotice(6540, "")
		return true
	}
	if err := a.world.Send(packet); err != nil {
		a.backToLogin("Unable to send raid request.")
	}
	return true
}

// prepareMemberBuffs feeds the member rows' effect strip (0x50abd0). Your own
// row lists your active effects; the others list what the party packets sent.
func (a *App) prepareMemberBuffs() {
	a.loadSkills()
	icon := func(skill uint16, level uint8) (ui.PartyBuff, bool) {
		def, ok := a.skillDefinition(skill, level)
		e := ui.PartyBuff{Skill: skill, Level: level, Debuff: level == 100 || def.Effect == 3}
		if !ok && level != 100 {
			return e, false
		}
		if file, uv, found := ui.SkillIcon(def.Icon); found {
			e.Tex, e.UV = a.assets.Texture("interface/"+file), uv
		}
		return e, true
	}
	b := &a.partyUI.Buffs
	b.MouseX, b.MouseY, b.Flash, b.Hovered = a.lastMouseX, a.lastMouseY, debuffFlash(a.clock), false
	a.partyUI.White = a.white
	b.Of = func(m proto.PartyMember) (out []ui.PartyBuff) {
		if m.ID == a.state.SelfID {
			for _, row := range a.buffs.rows {
				if e, ok := icon(row.buff.Skill, row.buff.Level); ok {
					out = append(out, e)
				}
			}
			return out
		}
		for _, f := range m.Effects {
			e, ok := icon(f.Skill, f.Level)
			e.Skip = !ok // the slot stays empty, as in the original
			out = append(out, e)
		}
		return out
	}
}

// drawMemberBuffTooltip names the hovered member effect with its description.
func (a *App) drawMemberBuffTooltip() {
	b := &a.partyUI.Buffs
	if !b.Hovered {
		return
	}
	def, _ := a.skillDefinition(b.Hover.Skill, b.Hover.Level)
	name := def.Name
	if name == "" {
		name = fmt.Sprintf("Skill %d, level %d", b.Hover.Skill, b.Hover.Level)
	}
	a.tooltip = []itemDetailLine{{name, [4]float32{1, 1, 0, 1}}}
	a.tooltipItem, a.tooltipSockets = false, nil
	a.tooltipDescription = strings.ReplaceAll(def.Description, `\n`, "\n")
}

// raidInfoLines are the raid properties panel's rows (0x513bc0): the join
// mode with the original texts (6528, 6529/6530) and the member count. The
// loot line of the original is left out for now.
func (a *App) raidInfoLines() [2]string {
	mode := a.msg.Get(6530)
	if a.state.Party.RaidAutoJoin {
		mode = a.msg.Get(6529)
	}
	return [2]string{
		fmt.Sprintf("%s %s", a.msg.Get(6528), mode),
		fmt.Sprintf("Members : %d", len(a.state.Party.Members)),
	}
}
