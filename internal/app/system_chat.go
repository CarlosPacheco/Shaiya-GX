package app

import (
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// systemMessage is the only formatter for native gameplay notices. The data
// archive owns their language and wording; missing IDs do not invent messages.
func (a *App) systemMessage(style uint8, id int, values map[string]string) {
	if a.msg.Lookup(id) == "" {
		return
	}
	a.addSystemText(style, id, a.msg.Format(id, values))
}

func (a *App) addSystemText(style uint8, id int, text string) {
	a.state.AddChat(game.ChatLine{Text: text, Style: style, MessageID: id})
	if nativeNoticeStyle(style) {
		a.addScreenNotice(style, text)
	}
	if style == 8 {
		a.playNewsSound()
	}
	// Native: every refusal in the error channel (17) comes with ui_error001,
	// 53 call sites in Game.exe from 0x4381ab to 0x5765a0.
	if style == 17 {
		a.playUISound("ui_error001.ogg")
	}
}

// ReportScreenshot displays the native world notice after the PNG is written.
func (a *App) ReportScreenshot(filename string) {
	if a.state == nil || a.state.Phase != game.PhaseInWorld {
		return
	}
	if filename == "" {
		a.systemMessage(17, 360, nil)
		return
	}
	a.systemMessage(7, 359, map[string]string{"b": filename})
}

// Keep diagnostic fallbacks usable without an archive in offline tools/tests.
func (a *App) localizedNotice(style uint8, id int, values map[string]string) {
	a.systemMessage(style, id, values)
}

func (a *App) chatEntityName(id uint32) string {
	if id == a.state.SelfID {
		if own, ok := a.selfSlot(); ok {
			return own.Name
		}
	}
	if e := a.state.Entities[id]; e != nil {
		if e.Shape != nil {
			return e.Shape.Name
		}
		if e.Kind == game.KindMob && a.assets != nil {
			if d, err := a.assets.MonsterDefinition(e.TypeID); err == nil {
				return d.Name
			}
		}
	}
	if m := a.state.Party.Member(id); m != nil {
		return m.Name
	}
	return ""
}

// Called only after State.Apply accepts the packet. This reports acknowledgments,
// never predicted attacks, inventory changes, or rewards.
func (a *App) confirmedSystemChat(p []byte, previousMoney uint32) {
	s := a.state
	number := func(n uint32) string { return strconv.FormatUint(uint64(n), 10) }
	itemName := func(typ, id uint8) string {
		if a.arc != nil {
			if d := a.itemsItem(typ, id); d != nil {
				return d.Name
			}
		}
		return ""
	}
	skillName := func(id uint16, level uint8) string { d, _ := a.skillDefinition(id, level); return d.Name }
	switch proto.Op(p) {
	case proto.MobSpeechText:
		if s.LastMobSpeech.Text != "" {
			a.addSystemText(30, 0, s.LastMobSpeech.Text)
		}
	case proto.MobSpeechIndex:
		a.systemMessage(30, int(s.LastMobSpeech.MessageID), nil)
	case proto.ChangeSexDenied:
		// 0x56b820: SysMsg 862 in style 17; the request is over.
		a.inventory.pending = false
		a.systemMessage(17, 862, nil)
	case proto.GXSkillFailure:
		style := uint8(1)
		switch s.LastSkillFailure.Message {
		case proto.MsgCannotUseSkill, proto.MsgCannotAttack, 317:
			style = 17
		}
		a.systemMessage(style, int(s.LastSkillFailure.Message), nil)
	case proto.GXLootNotice:
		n := s.LastLootNotice
		if name := itemName(n.Type, n.TypeID); name != "" && a.msg.Get(61) != "" {
			// SysMsg 61 "<p> attained <i>" for a group mate's item, with its link.
			a.state.AddChat(game.ChatLine{Text: a.msg.Format(61, map[string]string{"p": a.chatEntityName(n.Receiver), "i": "\uFFFC " + name}), Style: 3, MessageID: 61, ItemType: n.Type, ItemID: n.TypeID})
		}
	case proto.MobDeath:
		a.mobDeathMessage(p)
	case proto.GXKillNotice:
		a.systemMessage(1, 8, map[string]string{"t": s.LastKillNotice}) // red: "Killed <t>"
	case proto.GXDrowningDamage:
		r := s.LastDrowningDamage
		if r.Target == s.SelfID {
			a.combatMessage(1, 12372, map[string]string{"v": number(r.Amount)})
		}
	case proto.GXDrowningDeath:
		if s.LastDrowningDeath == s.SelfID {
			a.systemMessage(1, 12373, nil)
		}
	case proto.GXFallDamage:
		r := s.LastFallDamage
		if r.Target == s.SelfID {
			a.combatMessage(1, 5, map[string]string{"v": number(r.Amount)})
		}
	case proto.PortalNotice:
		v := s.LastPortalNotice
		if v.PortalID == 1060 || v.PortalID == 1061 {
			id := 0
			if v.NewOwner == 101 {
				id = 5514
			} else if v.NewOwner == 102 {
				id = 5515
			}
			if id != 0 {
				a.systemMessage(8, id, nil)
			}
		}
	case proto.GXServerNotice:
		if a.msg.Get(356) != "" {
			a.systemMessage(8, 356, map[string]string{"b": s.LastServerNotice}) // "[Notice] <b>"
		} else {
			a.addSystemText(8, 0, s.LastServerNotice)
		}
	case proto.RelicNotice:
		v := s.LastRelicNotice
		if id, ok := relicNoticeMessage(v.OldOwner, v.NewOwner, s.Faction.Faction); ok && a.assets != nil {
			if d, err := a.assets.MonsterDefinition(v.MobID); err == nil {
				style := relicNoticeStyle(s.MapID)
				a.systemMessage(style, id, map[string]string{"p": d.Name})
			}
		}
	case proto.ExpGain:
		gain, err := proto.DecodeExperienceGain(p)
		if err != nil {
			return
		}
		if gain.Base != 0 {
			a.systemMessage(3, 50, map[string]string{"b": number(gain.Base)})
		}
		if gain.Perfect != 0 {
			a.systemMessage(3, 66, map[string]string{"b": number(gain.Perfect)})
		}
	case proto.SetMoney:
		if s.Details.Money > previousMoney {
			a.systemMessage(3, 53, map[string]string{"b": number(s.Details.Money - previousMoney)})
		}
	case proto.LevelUpSelf, proto.LevelUpOther:
		r := proto.NewReader(p)
		id, level := r.U32(), r.U16()
		message := 218
		if id == s.SelfID {
			message = 217
		}
		a.systemMessage(3, message, map[string]string{"p": a.chatEntityName(id), "v": number(uint32(level))})
	case proto.AddItem:
		if len(p) == 4 {
			return
		}
		it, err := proto.DecodeItemAdd(p)
		if err == nil && it.Type != 26 {
			if name := itemName(it.Type, it.TypeID); name != "" {
				if a.msg.Get(52) != "" {
					a.state.AddChat(game.ChatLine{Text: a.msg.Format(52, map[string]string{"i": "\uFFFC " + name}), Style: 3, MessageID: 52, ItemType: it.Type, ItemID: it.TypeID})
				}
			}
		}
	case proto.UseItem:
		it := s.LastItemUse
		if it.ID == s.SelfID {
			if a.arc != nil {
				if d := a.itemsItem(it.Type, it.TypeID); d != nil && (d.RecoveryHP > 0 || d.RecoveryMP > 0 || d.RecoverySP > 0) {
					return
				}
			}
			if name := itemName(it.Type, it.TypeID); name != "" {
				a.systemMessage(3, 111, map[string]string{"i": name})
			}
		}
	case proto.LearnNewSkill:
		if r := s.LastLearn; r.Result == 0 {
			if name := skillName(r.ID, r.Level); name != "" {
				a.systemMessage(3, 59, map[string]string{"s": name})
			}
		}
	case proto.CharacterMobAutoAttack, proto.CharacterUserAutoAttack, proto.MobAttackCharacter:
		r := s.LastAttack
		target := a.chatEntityName(r.Target)
		message, style := 1000, uint8(0)
		if r.Source == s.SelfID {
			if r.Result == 1 {
				message = 1002
			}
			if r.Result == 3 {
				message = 1004
			}
		} else if r.Target == s.SelfID {
			message, style, target = 1001, 1, a.chatEntityName(r.Source)
			if r.Result == 1 {
				message = 1003
			}
			if r.Result == 3 {
				message = 1005
			}
		} else {
			return
		}
		a.combatMessage(style, message, map[string]string{"t": target, "p": a.chatEntityName(r.Source), "v": number(uint32(r.HP))})
	case proto.SkillUse, proto.SkillUseNoTarget, proto.SkillUserDrain, proto.SkillMobDrain, proto.SkillPartyMember, proto.SkillMobOther, proto.MobSkillUse, proto.MobSkillDrain, proto.MobSkillOther:
		r := s.LastSkill
		if r.Source != s.SelfID && r.Target != s.SelfID {
			return
		}
		if r.Result == 4 {
			if r.Source == s.SelfID {
				a.systemMessage(0, 1100, map[string]string{"s": skillName(r.ID, r.Level)})
			}
			return
		}
		if r.Result > 3 {
			switch r.Result {
			case 5:
				a.systemMessage(1, 233, nil)
			case 11:
				a.systemMessage(1, 317, nil)
			}
			return
		}
		level := r.Level
		if proto.Op(p) == proto.MobSkillUse || proto.Op(p) == proto.MobSkillDrain || proto.Op(p) == proto.MobSkillOther {
			level = 100
		}
		def, _ := a.skillDefinition(r.ID, level)
		name := def.Name
		target := a.chatEntityName(r.Target)
		if proto.Op(p) == proto.SkillUse || proto.Op(p) == proto.SkillUseNoTarget || proto.Op(p) == proto.SkillUserDrain || proto.Op(p) == proto.SkillMobDrain || proto.Op(p) == proto.MobSkillUse || proto.Op(p) == proto.MobSkillDrain {
			id, style := 1100, uint8(0)
			if r.Source != s.SelfID {
				id, style = 1300, 1
			}
			a.systemMessage(style, id, map[string]string{"s": name, "p": a.chatEntityName(r.Source)})
		}
		if r.Result == 3 {
			if r.Source == s.SelfID {
				a.systemMessage(0, 1108, map[string]string{"s": name, "t": target})
			} else {
				a.systemMessage(1, 1306, map[string]string{"s": name, "p": a.chatEntityName(r.Source)})
			}
			return
		}
		id := 1101
		if r.Result == 1 {
			id = 1102
		}
		if def.Effect == 2 {
			id = 1104
			if r.Target == s.SelfID {
				id = 1103
			}
		}
		style := uint8(0)
		if r.Source != s.SelfID {
			style = 1
			id = 1301
			if r.Result == 1 {
				id = 1302
			}
			if def.Effect == 2 {
				id = 1303
				style = 4
			}
		}
		if r.HP == 0 && r.MP == 0 && r.SP == 0 {
			notice := 1106
			if r.Source != s.SelfID {
				notice = 1305
			}
			a.systemMessage(style, notice, map[string]string{"s": name, "t": target, "p": a.chatEntityName(r.Source)})
			return
		}
		for poolIndex, amount := range []uint16{r.HP, r.MP, r.SP} {
			if amount > 0 {
				a.combatMessage(style, id, map[string]string{"s": name, "t": target, "p": a.chatEntityName(r.Source), "v": number(uint32(amount)), "hp": a.combatPoolLabel(poolIndex, def.Effect != 2)})
			}
		}
	case proto.SkillKeep:
		a.periodicSystemChat()
	case proto.SkillUserExtraDamage, proto.SkillExtraDamage:
		a.extraDamageSystemChat()
	case proto.PartyInvite, proto.PartyInviteName:
		name := s.Party.InviteName
		if name == "" {
			name = a.chatEntityName(s.Party.InviteID)
		}
		a.systemMessage(17, 204, map[string]string{"p": name})
	case proto.PartyJoined:
		a.playUISound("ui_party002.ogg") // native 0x574544: you or a member joined the party
		if m, err := proto.DecodePartyJoined(p); err == nil {
			id := 205
			if m.ID == s.SelfID {
				id = 207
			}
			a.systemMessage(3, id, map[string]string{"p": m.Name})
		}
	case proto.PartyLeft, proto.PartyExpelled:
		if id, err := proto.DecodeU32(p); err == nil && id == s.SelfID {
			a.systemMessage(3, 358, nil)
		}
	case proto.CharacterDeath:
		r := proto.NewReader(p)
		id := r.U32()
		if id == s.SelfID {
			a.systemMessage(2, 1, nil)
		}
	}
}

// mobDeathMessage is Game.exe 0x431190, run by the 0x0606 handler 0x564e50
// for a kill (death type 1): "<t> died" (SysMsg 3) when this character
// killed the monster, "<p> killed <t>" (SysMsg 4) when a party member did.
// Both use style 2; other killers print nothing.
func (a *App) mobDeathMessage(p []byte) {
	r := proto.NewReader(p)
	id, deathType, killer := r.U32(), r.U8(), r.U32()
	if r.Err() != nil || deathType != 1 || a.assets == nil {
		return
	}
	e := a.state.Entities[id]
	if e == nil || e.Kind != game.KindMob {
		return
	}
	def, err := a.assets.MonsterDefinition(e.TypeID)
	if err != nil || def.Name == "" {
		return
	}
	if killer == a.state.SelfID {
		a.systemMessage(2, 3, map[string]string{"t": def.Name})
		return
	}
	for _, m := range a.state.Party.Members {
		if m.ID == killer && killer != 0 {
			a.systemMessage(2, 4, map[string]string{"p": m.Name, "t": def.Name})
			return
		}
	}
}
