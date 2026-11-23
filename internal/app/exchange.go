package app

import (
	"fmt"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type exchangeView struct {
	inviteSerial, answerSerial, endSerial uint64
	pending, desired                      bool
	quantity                              *exchangeQuantity
	moneyOpen, cancelSent                 bool
	mutation, slot                        uint8
	deadline                              float64
}

func (a *App) exchangeCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "/trade") {
		return false
	}
	e := a.state.Entities[a.state.Target]
	if len(fields) > 1 {
		name := strings.Join(fields[1:], " ")
		e = nil
		for _, candidate := range a.state.Entities {
			if candidate.Kind == game.KindUser && candidate.Shape != nil && strings.EqualFold(candidate.Shape.Name, name) {
				e = candidate
				break
			}
		}
	}
	if e == nil || e.Kind != game.KindUser || e.ID == a.state.SelfID || a.state.Exchange.Active || a.state.Duel.Phase != game.DuelNone || a.state.GuildDuel.Phase != game.DuelNone {
		a.questMessageID(12068)
		return true
	}
	if err := a.world.Send(proto.ExchangeRequestPacket(e.ID)); err != nil {
		a.backToLoginID(12062)
	}
	return true
}
func (a *App) updateExchangeInvite(in *platform.Input, w, h int) bool {
	if a.exchangeInvite == nil || a.state == nil {
		return false
	}
	x := &a.state.Exchange
	v := &a.exchange
	if x.InviteSerial != v.inviteSerial {
		v.inviteSerial = x.InviteSerial
		if x.InviteID != 0 && !x.Active && a.clientSettings().RejectTrade {
			// Reject Trade (0x56c920): declined at once and silently.
			if err := a.world.Send(proto.ExchangeAnswerPacket(false)); err != nil {
				a.backToLoginID(12063)
			}
			return false
		}
		if x.InviteID != 0 && !x.Active {
			name := a.msg.Format(12296, map[string]string{"v": fmt.Sprint(x.InviteID)})
			if e := a.state.Entities[x.InviteID]; e != nil && e.Shape != nil && e.Shape.Name != "" {
				name = e.Shape.Name
			}
			a.playUISound("ui_party001.ogg") // native 0x572e92 and friends: an incoming request
			a.exchangeInvite.Open(name+" requests a trade. Accept?", true)
		}
	}
	if x.AnswerSerial != v.answerSerial {
		v.answerSerial = x.AnswerSerial
		if x.Answer != 0 {
			a.localizedNotice(1, 321, nil)
		}
	}
	if x.Active {
		a.exchangeInvite.Visible = false
	}
	if !a.exchangeInvite.Visible {
		return false
	}
	result := a.exchangeInvite.Update(in, w, h)
	if result != ui.MessageNone {
		accept := result == ui.MessageOK
		if accept && (a.state.Duel.Phase != game.DuelNone || a.state.GuildDuel.Phase != game.DuelNone) {
			accept = false
			a.questMessageID(12069)
		}
		if err := a.world.Send(proto.ExchangeAnswerPacket(accept)); err != nil {
			a.backToLoginID(12063)
		}
	}
	return true
}
func (a *App) updateExchange(in *platform.Input, w, h int) bool {
	if a.exchangeUI == nil || a.state == nil {
		return false
	}
	v := &a.exchange
	x := &a.state.Exchange
	if v.endSerial != x.EndSerial {
		v.endSerial = x.EndSerial
		v.pending = false
		if x.EndSerial > 0 {
			if x.EndResult == 0 {
				if a.msg.Get(211) != "" {
					a.systemMessage(3, 211, nil)
				} else {
					a.questMessageID(12070)
				}
			} else {
				if a.msg.Get(212) != "" {
					a.systemMessage(3, 212, nil)
				} else {
					a.questMessageID(12071)
				}
			}
		}
	}
	if !x.Active {
		v.pending = false
		return false
	}
	if v.pending && x.OwnReady == v.desired && x.OwnConfirmed == v.desired {
		v.pending = false
	}
	confirm, close := a.exchangeUI.Update(in, w, h)
	if close || in.HasKey(platform.KeyEscape) {
		if err := a.world.Send(proto.ExchangeReadyPacket(2)); err != nil {
			a.backToLoginID(12064)
		}
		return true
	}
	x0, y0 := a.exchangeUI.Origin(w, h)
	if in.RightPressed {
		if slot, hit := ui.ExchangeHit(in.MouseX-x0, in.MouseY-y0); hit && x.Own[slot].Item.Type != 0 {
			if !a.exchangeCanEdit() {
				return true
			}
			if x.Other[slot].Type != 0 {
				a.questMessageID(12072)
				return true
			}
			if err := a.world.Send(proto.ExchangeRemovePacket(uint8(slot))); err != nil {
				a.backToLoginID(12065)
				return true
			}
			a.state.MarkExchangeRemove(uint8(slot))
			v.mutation, v.slot, v.deadline = 2, uint8(slot), a.clock+10
			return true
		}
	}
	if in.Clicked && in.MouseX >= x0+87 && in.MouseX < x0+182 && in.MouseY >= y0+175 && in.MouseY < y0+197 && a.exchangeMoney != nil {
		if a.exchangeCanEdit() {
			v.moneyOpen = true
			a.exchangeMoney.Open(w, h, x.OwnMoney)
		}
		return true
	}
	if confirm && !v.pending && v.mutation == 0 && !v.cancelSent {
		desired := !(x.OwnReady && x.OwnConfirmed)
		ready := uint8(1)
		if desired {
			ready = 0
		}
		if err := a.world.Send(proto.ExchangeReadyPacket(ready)); err != nil {
			a.backToLoginID(12066)
			return true
		}
		if err := a.world.Send(proto.ExchangeConfirmPacket(desired)); err != nil {
			a.backToLoginID(12067)
			return true
		}
		v.pending, v.desired = true, desired
		return true
	}
	return panelPointer(in, x0, y0, ui.ExchangeWidth, ui.ExchangeHeight)
}
func (a *App) drawExchange(w, h int) {
	if a.exchangeUI == nil || !a.state.Exchange.Active {
		return
	}
	x := &a.state.Exchange
	name := a.msg.Format(12296, map[string]string{"v": fmt.Sprint(x.PartnerID)})
	if e := a.state.Entities[x.PartnerID]; e != nil && e.Shape != nil && e.Shape.Name != "" {
		name = e.Shape.Name
	}
	a.exchangeUI.Draw(a.sprites, a.font, w, h, name, x.OwnMoney, x.OtherMoney, x.OwnReady && x.OwnConfirmed, x.OtherReady && x.OtherConfirmed)
	px, py := a.exchangeUI.Origin(w, h)
	for side := 0; side < 2; side++ {
		for i := 0; i < proto.ExchangeSlots; i++ {
			it := x.Other[i]
			if side == 0 {
				it = x.Own[i].Item
			}
			if it.Type == 0 || it.Count == 0 {
				continue
			}
			cx, cy, _ := ui.ExchangeCell(side, i)
			if d := a.itemsItem(it.Type, it.TypeID); d != nil {
				a.drawItemIcon(d, px+cx, py+cy, 32, white, &proto.InventoryItem{Type: it.Type, TypeID: it.TypeID, Gems: it.Gems})
			}
			if a.itemCount(it.Type, it.TypeID, it.Count) > 0 {
				text := fmt.Sprint(it.Count)
				a.font.Draw(a.sprites, text, px+cx+32-a.font.Width(text), py+cy+18, white, 32)
			}
		}
	}
}
