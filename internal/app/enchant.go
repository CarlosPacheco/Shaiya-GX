package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type enchantView struct {
	lapCount, materialCount      uint16
	cost                         uint32
	noGold                       bool
	lap, material, item          [2]uint8
	pending, success, refreshing bool
	refresh                      gemRefresh
}

func (a *App) sendEnchant(lapKey, materialKey, itemKey [2]uint8, count uint16) bool {
	count = 1 // Each operation uses exactly one lapisia and one material.
	if a.state == nil || a.world == nil || a.state.Self.Dead || a.shop.pending || a.inventory.pending || a.state.Exchange.Active ||
		a.state.Duel.Phase != game.DuelNone || lapKey == materialKey || lapKey == itemKey || materialKey == itemKey {
		return false
	}
	lap, l := a.state.Inventory[lapKey]
	material, m := a.state.Inventory[materialKey]
	item, i := a.state.Inventory[itemKey]
	if !l || !m || !i || lap.Type != 95 || lap.Count == 0 || item.Count != 1 || count == 0 || count > 100 || count > material.Count {
		return false
	}
	cost := a.forgeGoldFor(lap)
	if a.insufficientForgeGold(a.state.Details.Money, cost, 10102) {
		return false
	}
	a.enchant = enchantView{lapCount: lap.Count, materialCount: material.Count, cost: cost, lap: lapKey, material: materialKey, item: itemKey, pending: true}
	a.shop.pending, a.shop.deadline = true, a.clock+10
	if err := a.world.Send(proto.EnchantPacket(lapKey[0], lapKey[1], materialKey[0], materialKey[1], count, itemKey[0], itemKey[1])); err != nil {
		a.backToLoginID(12057)
		return false
	}
	return true
}

func (a *App) confirmedEnchant(p []byte) {
	v := &a.enchant
	if !v.pending {
		return
	}
	switch proto.Op(p) {
	case proto.EnchantAdd:
		r, err := proto.DecodeEnchantResult(p)
		if err != nil || v.refreshing || v.lap != [2]uint8{r.LapisiaBag, r.LapisiaSlot} || v.material != [2]uint8{r.MaterialBag, r.MaterialSlot} || v.item != [2]uint8{r.ItemBag, r.ItemSlot} {
			return
		}
		v.noGold = r.Result != 1 && r.Money < v.cost && r.LapisiaCount == v.lapCount && r.MaterialCount == v.materialCount
		v.success, v.refreshing = r.Result == 1, true
		v.refresh.start([][2]uint8{v.item})
		a.shop.deadline = a.clock + 10
		if err := a.world.Send(proto.ItemMovePacket(v.item[0], v.item[1], v.item[0], v.item[1])); err != nil {
			a.backToLoginID(12058)
		}
	case proto.ItemBagToBag:
		if v.refreshing && v.refresh.acknowledge(p) && len(v.refresh.pending) == 0 {
			v.pending, a.shop.pending = false, false
			a.syncBlacksmithItems()
			a.forgeResult(v.success)
			if v.noGold {
				a.insufficientForgeGold(0, v.cost, 10102)
			} else if v.success {
				a.questMessageID(12059)
			} else {
				a.questMessageID(12060)
			}
		}
	}
}

func (a *App) updateEnchantQuantity(in *platform.Input) {
	v := &a.blacksmith
	if !a.blacksmithOpen() || !a.blacksmithEnchantReady() || a.shop.pending || a.inventory.pending {
		v.quantity = false
		return
	}
	count := a.quantityUI.Update(in)
	if count == 0 {
		return
	}
	v.quantity = false
	if count < 0 {
		return
	}
	v.materialCount, v.confirm = uint16(count), 5
	if a.gemConfirm != nil {
		a.requestForgeQuote(proto.ForgeQuoteEnchant, 5, true)
	}
}
