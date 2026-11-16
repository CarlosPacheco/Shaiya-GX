package app

// itemCount is the icon label quantity, not the owned or transferable amount.
// Stackability comes from Item.SData, never from the current stack size.
func (a *App) itemCount(kind, id uint8, count uint16) uint16 {
	if d := a.itemsItem(kind, id); d != nil && d.Count > 1 {
		return count
	}
	return 0
}
