package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/forgecost"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"strconv"
)

func (a *App) forgeGoldFor(it proto.InventoryItem) uint32 {
	if a.items == nil && a.arc == nil {
		return 0
	}
	if def := a.itemsItem(it.Type, it.TypeID); def != nil {
		return forgecost.Gold(def.ReqIg)
	}
	return 0
}

func (a *App) insufficientForgeGold(balance, cost uint32, id int) bool {
	if balance >= cost {
		return false
	}
	a.localizedNotice(3, id, map[string]string{"b": strconv.FormatUint(uint64(cost), 10)})
	return true
}
