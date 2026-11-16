package world

import (
	"path/filepath"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// The guild bank follows CGuild::ItemBagToGuildBank 0x0042f390,
// ItemGuildBankToGuildBank 0x0042fb40, ItemGuildBankToBag 0x0042fea0 and
// SendBankItemList 0x0042f250, reached from CUser::PacketMaininterface's
// 0x0204 with bag 0xFF (CUser::IsGuildBank: slots below 0xF0). Only guilds
// ranked 1-30 may use it; the usable slots come from GuildHouse.ini [Bank]
// PermitSlot at the house bank NPC level (0 without a house).

const (
	guildBankBag     = 0xff
	guildBankSlots   = 240
	guildBankRanked  = 30 // CGuild +0x48 must be at most this
	guildBankBatch   = 50 // 0x0D29 entries per packet
	guildBankRefused = 1  // 0x0D28 result: guild not ranked
	guildBankNoRight = 2  // not an admin / slot beyond the permitted pages
	guildBankMember9 = 4  // rank 9 may not deposit
)

// guildBankPermitSlot is GuildHouse.ini [Bank] PermitSlot.
var guildBankPermitSlot = []int{40, 80, 120, 160, 200, 240}

func loadGuildHouseIni() {
	ini, err := readIni(filepath.Join(gameDataDir(), "GuildHouse.ini"))
	if err != nil {
		return
	}
	if v := ini["BANK"]["PERMITSLOT"]; v != "" {
		var slots []int
		for _, n := range numbers(v) {
			slots = append(slots, int(n))
		}
		if len(slots) > 0 {
			guildBankPermitSlot = slots
		}
	}
}

// bankSlots is GetGuildHouseBankAllowSlotCount; no guild house exists yet,
// so the bank NPC level is 0.
func (g *guild) bankSlots() int { return guildBankPermitSlot[0] }

func loadGuildBank(g *guild) error {
	items, err := db.GuildStoredItems(g.ID)
	if err != nil {
		return err
	}
	for i := range items {
		it := items[i]
		if int(it.Slot) < guildBankSlots && g.bank[it.Slot] == nil {
			it.Bag = guildBankBag
			g.bank[it.Slot] = &it
		}
	}
	return nil
}

// bankListPackets is SendBankItemList: 0x0D29, 33-byte entries, 50 each.
func (g *guild) bankListPackets() [][]byte {
	var out [][]byte
	var batch []*db.Item
	flush := func() {
		if len(batch) == 0 {
			return
		}
		w := packet.New(packet.GuildBankList)
		w.U8(uint8(len(batch)))
		for _, it := range batch {
			w.U8(it.Slot)
			w.U8(it.Type)
			w.U8(it.TypeID)
			w.U16(it.Quality)
			w.Bytes(it.Gems[:])
			w.Count(it.Count)
			w.FixedString(it.CraftName, 21)
		}
		out = append(out, w.MustFinish())
		batch = nil
	}
	for _, it := range g.bank {
		if it != nil {
			batch = append(batch, it)
			if len(batch) == guildBankBatch {
				flush()
			}
		}
	}
	flush()
	return out
}

// buildGuildBankSlot (0x0D31 placed / 0x0D32 taken): slot, item, char id.
func buildGuildBankSlot(op packet.Opcode, slot uint8, it *db.Item, charID uint32) []byte {
	w := packet.New(op)
	w.U8(slot)
	if it != nil {
		w.U8(it.Type)
		w.U8(it.TypeID)
		w.U16(it.Quality)
		w.Bytes(it.Gems[:])
		w.Count(it.Count)
		w.FixedString(it.CraftName, 21)
	} else {
		w.Bytes(make([]byte, 10))
		w.Count(0)
		w.FixedString("", 21)
	}
	w.U32(charID)
	return w.MustFinish()
}

func buildGuildBankResult(result, arg uint8) []byte {
	w := packet.New(packet.GuildBankResult)
	w.U8(result)
	w.U8(arg)
	return w.MustFinish()
}

// guildBankCheck is the shared gate: ranked guild, the rank rule and the
// permitted slot; it sends the 0x0D28 refusal itself.
func (s *Session) guildBankCheck(g *guild, rank uint8, slot uint8, adminOnly bool) bool {
	switch {
	case g.Rank > guildBankRanked:
		s.send(buildGuildBankResult(guildBankRefused, guildBankRanked), "0x0D28 guild bank refused (rank)")
	case !adminOnly && rank == guildRankMember:
		s.send(buildGuildBankResult(guildBankMember9, 8), "0x0D28 guild bank refused (rank 9)")
	case adminOnly && rank > guildRankAdmin:
		s.send(buildGuildBankResult(guildBankNoRight, guildBankNoRight), "0x0D28 guild bank refused (not admin)")
	case int(slot) >= g.bankSlots():
		s.send(buildGuildBankResult(guildBankNoRight, slot/40), "0x0D28 guild bank refused (slot)")
	default:
		return true
	}
	return false
}

// handleGuildBankMove is the 0x0204 with bag 0xFF on either side.
func (s *Session) handleGuildBankMove(srcBag, srcSlot, destBag, destSlot uint8) {
	if s.char == nil || s.player == nil {
		return
	}
	guildMu.Lock()
	defer guildMu.Unlock()
	g := s.player.guild.guild
	if g == nil || g.members[s.char.ID] == nil {
		return
	}
	planned := *g
	var before []db.Item
	for i, item := range g.bank {
		if item != nil {
			copy := *item
			planned.bank[i] = &copy
			before = append(before, copy)
		}
	}
	committed := s.inventoryOperationWithCommit(func() {
		rank := g.members[s.char.ID].Rank
		switch {
		case srcBag == guildBankBag && destBag == guildBankBag:
			if srcSlot < guildBankSlots && destSlot < guildBankSlots {
				s.guildBankRearrange(&planned, rank, srcSlot, destSlot)
			}
		case destBag == guildBankBag:
			if destSlot < guildBankSlots {
				s.guildBankDeposit(&planned, rank, srcBag, srcSlot, destSlot)
			}
		default:
			if srcSlot < guildBankSlots {
				s.guildBankWithdraw(&planned, rank, srcSlot, destBag, destSlot)
			}
		}
	}, func(change db.CharacterChange) error {
		var after []db.Item
		for _, item := range planned.bank {
			if item != nil {
				after = append(after, *item)
			}
		}
		return db.CommitGuildBank(g.ID, change, before, after)
	})
	if committed {
		g.bank = planned.bank
	}

}

// guildBankDeposit is ItemBagToGuildBank: not from the equipment, no quest
// items (real type 27) nor ReqOg items; a taken slot only for admins (swap).
func (s *Session) guildBankDeposit(g *guild, rank, bag, slot, dest uint8) {
	ch := s.char
	if bag == 0 || !isValidItemPosition(bag, slot) {
		return
	}
	i := findItem(s.items, bag, slot)
	if i < 0 {
		return
	}
	d := db.MustDefs().Item(s.items[i].Type, s.items[i].TypeID)
	if d == nil || realType(d.Type) == 27 || d.ReqOg != 0 {
		return
	}
	if !s.guildBankCheck(g, rank, dest, false) {
		return
	}
	old := g.bank[dest]
	if old != nil && rank > guildRankAdmin {
		s.send(buildGuildBankResult(guildBankNoRight, guildBankNoRight), "0x0D28 guild bank refused (slot taken)")
		return
	}
	stored := s.items[i]
	s.items = append(s.items[:i], s.items[i+1:]...)
	stored.Bag, stored.Slot = guildBankBag, dest
	var back *db.Item
	if old != nil {
		b := *old
		b.Bag, b.Slot = bag, slot
		s.items = append(s.items, b)
		back = &b
	}
	g.bank[dest] = &stored
	s.send(buildBagToBag(bag, slot, back, guildBankBag, dest, &stored), "0x0204 guild bank deposit")
	if old != nil {
		s.guildPublication(g, buildGuildBankSlot(packet.GuildBankTaken, dest, old, ch.ID), "0x0D32 guild bank taken", s.operationPlayer())
	}
	s.guildPublication(g, buildGuildBankSlot(packet.GuildBankPlaced, dest, &stored, ch.ID), "0x0D31 guild bank placed", s.operationPlayer())
}

// guildBankWithdraw is ItemGuildBankToBag: admins only, 5% of Sell x
// count (truncated; type 94 free), a taken bag slot swaps into the bank.
func (s *Session) guildBankWithdraw(g *guild, rank, src, bag, slot uint8) {
	ch := s.char
	it := g.bank[src]
	if it == nil || bag == 0 || !isValidItemPosition(bag, slot) {
		return
	}
	if !s.guildBankCheck(g, rank, src, true) {
		return
	}
	fee := uint32(0)
	if d := db.MustDefs().Item(it.Type, it.TypeID); d != nil && d.Type != 94 {
		fee = uint32(float32(uint32(it.Count)*uint32(max(d.Sell, 0))) * float32(0.05))
	}
	if ch.Money < fee {
		return
	}
	taken := *it
	g.bank[src] = nil
	var into *db.Item
	if j := findItem(s.items, bag, slot); j >= 0 {
		in := s.items[j]
		s.items = append(s.items[:j], s.items[j+1:]...)
		in.Bag, in.Slot = guildBankBag, src
		g.bank[src] = &in
		into = &in
	}
	taken.Bag, taken.Slot = bag, slot
	s.items = append(s.items, taken)
	ch.Money -= fee
	s.persist()
	s.send(withMoney(buildBagToBag(guildBankBag, src, into, bag, slot, &taken), ch.Money), "0x0204 guild bank withdraw")
	s.guildPublication(g, buildGuildBankSlot(packet.GuildBankTaken, src, &taken, ch.ID), "0x0D32 guild bank taken", s.operationPlayer())
	if into != nil {
		s.guildPublication(g, buildGuildBankSlot(packet.GuildBankPlaced, src, into, ch.ID), "0x0D31 guild bank placed", s.operationPlayer())
	}
}

// guildBankRearrange is ItemGuildBankToGuildBank: admins only; equal items
// merge up to the stack, otherwise the slots swap; every member sees the
// 0x0204.
func (s *Session) guildBankRearrange(g *guild, rank, src, dest uint8) {
	a := g.bank[src]
	if a == nil || src == dest {
		return
	}
	if !s.guildBankCheck(g, rank, dest, true) {
		return
	}
	b := g.bank[dest]

	if b != nil && itemsMatch(a, b) {
		if ms := maxStackOf(a.Type, a.TypeID); ms > 1 && b.Count < ms {
			moved := min(a.Count, ms-b.Count)
			b.Count += moved
			a.Count -= moved

			if a.Count == 0 {

				g.bank[src] = nil
			} else {

			}
			s.guildPublication(g, buildBagToBag(guildBankBag, src, g.bank[src], guildBankBag, dest, b), "0x0204 guild bank merge", nil)
			return
		}
	}
	g.bank[src], g.bank[dest] = b, a
	a.Slot = dest

	if b != nil {
		b.Slot = src

	}
	s.guildPublication(g, buildBagToBag(guildBankBag, src, b, guildBankBag, dest, a), "0x0204 guild bank move", nil)
}
