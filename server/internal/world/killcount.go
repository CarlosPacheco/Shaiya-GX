package world

import (
	"context"
	"errors"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/blessingrules"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// The country kill count ("God's blessing") follows ps_game.exe:
//
//   - CGameData::LoadKCStatus 0x004bf4a0 reads data/KillStatus.SData:
//     per record a country, a kill count and six (type, value) pairs over
//     types 1-27; each tier then keeps at least every earlier tier's value
//     per type.
//   - CWorld::AddKillCount 0x00413600: each user kill (KillPCTo) adds 1 to
//     the killer's country, unless either country waits at the top tier.
//     Crossing a tier's count moves to the highest tier reached; the top
//     tier holds 10 minutes and resets the other country to 0 at once
//     (ResetKillCount); CWorld::UpdateKillCount then resets this one too.
//     Counts are announced to the country with 0x020F (country, count up
//     to the top tier's) on tier transitions and every 10 minutes, saved with
//     usp_Save_GodBless_E, loaded from usp_Read_GodBless_R, and sent at
//     world entry with 0x0211 (country, count, ms until the reset).
//   - CUser::UpdateKCStatus 0x0048b780 / InitKCStatus: the tier's types on
//     every user of the country. 1-3 add that percent of the maximum
//     HP/MP/SP; 4-9 STR, REC, INT, WIS, DEX, LUC (REC/WIS/DEX with their
//     5 pool points); 10-12 add that percent of the maximum HP/SP/MP to the
//     sitting and standing recovery, 13-15 to the in-combat recovery; 16
//     critical (ability 18); 17 melee accuracy (20); 18/19 that percent of
//     the melee/ranged defense; 20 and 23 magic evasion (31); 21/22 melee
//     and ranged evasion (29/30). Country-wide, 24 adds to the gem chance
//     (ItemGemAdd / ItemGemRemoveAll) and 25 cuts the death exp loss by
//     that percent (ExpDrop). Types 26 and 27 have no reader in EP4.5.
//
// Below the first tier no type applies (the original reads the slot
// before the table there). Relic captures credit/debit these same country counters.
// The reference CMap::LoadObelisk writes the country amounts at 0x005471c0
// and 0x005471c4 (0x00428557/0x00428575); CMap::ObeliskDestroy consumes them.
// See server/docs/original-parity.md for the versioned obelisk source audit.

const (
	kcTypes          = 28
	kcHoldTime       = 10 * time.Minute
	kcBroadcastEvery = 10 * time.Minute
)

type kcTier struct {
	Kill   uint32
	Status [kcTypes]int32
}

type kcCountry struct {
	count       uint32
	index       int // -1 below the first tier
	next        uint32
	resetAt     time.Time // the top tier's hold
	broadcastAt time.Time
}

var (
	kcMu     sync.Mutex
	kcSendMu sync.Mutex
	kcTiers  []kcTier
	kcWorld  = [2]kcCountry{{index: -1}, {index: -1}}
)

// loadKillStatus is CGameData::LoadKCStatus.
func loadKillStatus(log *logx.Logger) error {
	pool, err := db.Pool(db.GameDefs)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()
	catalog, err := blessingrules.Read(ctx, pool)
	if err != nil {
		return err
	}
	tiers := make([]kcTier, len(catalog))
	for i, row := range catalog {
		tiers[i].Kill = row.RequiredKills
		for bonus, value := range catalog.Effective(i) {
			tiers[i].Status[bonus+1] = int32(value)
		}
	}
	kcMu.Lock()
	kcTiers = tiers
	kcMu.Unlock()
	log.Printf("GX Server World loaded shared SQLite blessing tiers=%d", len(tiers))
	return nil
}

// kcTierFor is CGameData::GetKCStatusByCount (-1 below the first tier).
func kcTierFor(count uint32) int {
	t := kcTiers
	for i := len(t) - 1; i >= 0; i-- {
		if t[i].Kill <= count {
			return i
		}
	}
	return -1
}

func kcMaxKill() uint32 {
	if t := kcTiers; len(t) > 0 {
		return t[len(t)-1].Kill
	}
	return 0
}

// setKillCountLocked is CWorld::SetKillCount / InitKillCount for c.
func setKillCountLocked(c int, count uint32, now time.Time) {
	w := &kcWorld[c]
	w.count = count
	w.index = kcTierFor(count)
	w.next, w.resetAt = 0, time.Time{}
	switch {
	case w.index+1 < len(kcTiers):
		w.next = kcTiers[w.index+1].Kill
	case len(kcTiers) > 0:
		w.resetAt = now.Add(kcHoldTime)
	}
	w.broadcastAt = now.Add(kcBroadcastEvery)
}

// InitKillCount loads the saved counts at world start.
func InitKillCount(log *logx.Logger) {
	light, dark, err := db.ReadGodBless()
	if err != nil {
		log.Printf("GX Server World kill count load failed: %v", err)
	}
	kcMu.Lock()
	now := time.Now()
	setKillCountLocked(0, light, now)
	setKillCountLocked(1, dark, now)
	kcMu.Unlock()
	log.Printf("GX Server World country kill counts light=%d dark=%d", light, dark)
}

// AddKillCount is CWorld::AddKillCount for country c.
func AddKillCount(c uint8, n uint32, log logPrinter) {
	if c > 1 {
		return
	}
	kcMu.Lock()
	if !kcWorld[0].resetAt.IsZero() || !kcWorld[1].resetAt.IsZero() {
		kcMu.Unlock()
		return
	}
	now := time.Now()
	w := &kcWorld[c]
	w.count = satAdd(w.count, n)
	if w.next == 0 || w.count < w.next {
		kcMu.Unlock()
		return
	}
	w.index = kcTierFor(w.count)
	maxed := w.index >= len(kcTiers)-1
	if maxed {
		w.resetAt = now.Add(kcHoldTime)
	} else {
		w.next = kcTiers[w.index+1].Kill
	}
	w.broadcastAt = now.Add(kcBroadcastEvery)
	other := 1 - int(c)
	if maxed {
		setKillCountLocked(other, 0, now)
		kcWorld[other].resetAt = time.Time{}
	}
	light, dark := kcWorld[0].count, kcWorld[1].count
	count, index := w.count, w.index
	kcMu.Unlock()
	kcApplyCountry(int(c))
	kcBroadcast(int(c), true)
	if maxed {
		kcApplyCountry(other)
		kcBroadcast(other, true)
		kcSave(light, dark, log)
		ResetRelics(now, log)
	}
	log.Printf("GX Server World country %d kill count %d reached tier %d (maxed=%t)", c, count, index, maxed)
}

// TickKillCount is CWorld::UpdateKillCount.
func TickKillCount(now time.Time, log logPrinter) {
	kcMu.Lock()
	var announce, apply []int
	dirty := false
	for c := range kcWorld {
		w := &kcWorld[c]
		switch {
		case w.resetAt.IsZero():
			if !now.Before(w.broadcastAt) {
				announce = append(announce, c)
				w.broadcastAt = now.Add(kcBroadcastEvery)
				dirty = true
			}
		case !now.Before(w.resetAt):
			setKillCountLocked(c, 0, now)
			w.resetAt = time.Time{}
			announce, apply = append(announce, c), append(apply, c)
			dirty = true
		}
	}
	light, dark := kcWorld[0].count, kcWorld[1].count
	kcMu.Unlock()
	for _, c := range apply {
		kcApplyCountry(c)
	}
	for _, c := range announce {
		kcBroadcast(c, slices.Contains(apply, c))
	}
	if dirty {
		kcSave(light, dark, log)
	}
}

// kcSave stores both counts; without a WorldInfo row (the world calendar
// creates it at start) it creates the row first.
func kcSave(light, dark uint32, log logPrinter) {
	if err := kcStore(light, dark); err != nil {
		log.Printf("GX Server World kill count save failed: %v", err)
	}
}

// kcStore is the storage step (tests replace it).
var kcStore = func(light, dark uint32) error {
	err := db.SaveGodBless(light, dark)
	if errors.Is(err, db.ErrNoWorldInfo) {
		if err = db.SaveWorldInfo(worldBasePacked()); err == nil {
			err = db.SaveGodBless(light, dark)
		}
	}
	return err
}

// kcBroadcast sends the native country count and, on state changes, the GX snapshot.
func kcBroadcast(c int, snapshot bool) {
	kcSendMu.Lock()
	defer kcSendMu.Unlock()
	kcMu.Lock()
	count := min(kcWorld[c].count, kcMaxKill())
	var details []byte
	if snapshot {
		details = blessingSnapshotLocked(uint8(c), false)
	}
	kcMu.Unlock()
	w := packet.New(0x020F)
	w.U8(uint8(c))
	w.U32(count)
	frame := w.MustFinish()
	for _, p := range AllPlayers() {
		if int(p.Faction()) == c {
			p.Send(frame, "0x020F country kill count")
			if details != nil {
				p.Send(details, "GX blessing snapshot")
			}
		}
	}
}

// kcApplyCountry is ApplyKillCount: every user of c refreshes its stats.
func kcApplyCountry(c int) {
	for _, p := range AllPlayers() {
		if int(p.Faction()) == c {
			p.kcRefresh.Store(true)
		}
	}
}

// buildKCStatus is CUser::SendKCStatus (0x0211).
func buildKCStatus(c uint8) []byte {
	kcMu.Lock()
	var count uint32
	var left int64
	if c < 2 {
		count = kcWorld[c].count
		if r := kcWorld[c].resetAt; !r.IsZero() {
			left = max(time.Until(r).Milliseconds(), 0)
		}
	}
	kcMu.Unlock()
	w := packet.New(0x0211)
	w.U8(c)
	w.U32(count)
	w.U32(uint32(left))
	return w.MustFinish()
}

func buildBlessingSnapshot(c uint8) []byte {
	kcMu.Lock()
	defer kcMu.Unlock()
	return blessingSnapshotLocked(c, true)
}

func blessingSnapshotLocked(c uint8, entry bool) []byte {
	if c >= uint8(len(kcWorld)) {
		return nil
	}
	world := kcWorld[c]
	maximum := kcMaxKill()
	count := min(world.count, maximum)
	var remaining uint32
	if !world.resetAt.IsZero() {
		remaining = uint32(max(time.Until(world.resetAt).Milliseconds(), 0))
	}
	w := packet.New(packet.Opcode(proto.GXBlessingSnapshot))
	w.U8(c)
	var flags uint8
	if entry {
		flags = 1
	}
	w.U8(flags)
	w.U32(count)
	w.U32(maximum)
	w.U32(remaining)
	// The native tooltip selects by count. Actual stats keep debit hysteresis.
	index := kcTierFor(count)
	for typ := 1; typ < kcTypes; typ++ {
		var value uint16
		if index >= 0 {
			value = uint16(kcTiers[index].Status[typ])
		}
		w.U16(value)
	}
	return w.MustFinish()
}

// kcStatus is the country's current tier types (zero below the first).
func kcStatus(c uint8) [kcTypes]int32 {
	kcMu.Lock()
	defer kcMu.Unlock()
	if c > 1 || kcWorld[c].index < 0 || kcWorld[c].index >= len(kcTiers) {
		return [kcTypes]int32{}
	}
	return kcTiers[kcWorld[c].index].Status
}

// kcAbility maps the flat KC types onto SetSkillAbility kinds (same field).
var kcAbility = map[int]uint8{4: 4, 5: 5, 6: 6, 7: 7, 8: 8, 9: 9, 16: 18, 17: 20, 20: 31, 21: 29, 22: 30, 23: 31}

// applyKCStatus adds the country tier's user types to the stats.
func applyKCStatus(e *EquipmentStats, c uint8) {
	st := kcStatus(c)
	for typ, kind := range kcAbility {
		if v := st[typ]; v != 0 {
			applyAbility(e, typeEffectBuff, kind, v)
		}
	}
	e.KCPoolPct = [3]int32{st[1], st[2], st[3]}       // HP, MP, SP
	e.KCRecoverPct = [3]int32{st[10], st[11], st[12]} // HP, SP, MP (sit and stand)
	e.KCCombatPct = [3]int32{st[13], st[14], st[15]}  // HP, SP, MP (in combat)
	e.KCGuardPct = [2]int32{st[18], st[19]}
}

// kcPercent is UpdateKCStatus' (int)((float)(base * value) * 0.01).
func kcPercent(base uint32, v int32) uint32 {
	if v == 0 {
		return 0
	}
	scaled := math.Trunc(float64(float32(int64(base)*int64(v))) * float64(float32(0.01)))
	return uint32(min(max(scaled, 0), math.MaxUint32))
}

// kcWorldType is a country-wide type (24 gem chance, 25 exp loss cut).
func kcWorldType(c uint8, typ int) int32 { return kcStatus(c)[typ] }

// SubKillCount is the obelisk capture debit. Active blessing freezes both
// countries, including subtraction; counts cannot wrap below zero.
func SubKillCount(c uint8, n uint32, log logPrinter) {
	if c > 1 {
		return
	}
	kcMu.Lock()
	if !kcWorld[0].resetAt.IsZero() || !kcWorld[1].resetAt.IsZero() {
		kcMu.Unlock()
		return
	}
	now := time.Now()
	w := &kcWorld[c]
	w.count -= min(w.count, n)
	// Native 0x413750 only reselects after crossing below the preceding
	// tier's threshold. Preserve that hysteresis rather than treating a debit
	// as SetKillCount; below the first table row GX safely applies no bonuses.
	if w.index > 0 && w.count < kcTiers[w.index-1].Kill {
		w.index = kcTierFor(w.count)
		w.next = kcTiers[w.index+1].Kill
		w.broadcastAt = now.Add(kcBroadcastEvery)
	}
	light, dark := kcWorld[0].count, kcWorld[1].count
	kcMu.Unlock()
	kcApplyCountry(int(c))
	kcBroadcast(int(c), true)
	kcSave(light, dark, log)
}

// Keep threshold/hold semantics identical to individual kill credits while
// reducing a burst to at most one update per country tier.
func addKillCountBatch(country uint8, count uint32, log logPrinter) {
	if country > 1 {
		return
	}
	for count > 0 {
		kcMu.Lock()
		if !kcWorld[0].resetAt.IsZero() || !kcWorld[1].resetAt.IsZero() {
			kcMu.Unlock()
			return
		}
		w := kcWorld[country]
		chunk := count
		if w.next > w.count {
			chunk = min(chunk, w.next-w.count)
		}
		kcMu.Unlock()
		AddKillCount(country, chunk, log)
		count -= chunk
	}
}
