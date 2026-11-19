package world

import (
	"context"
	"math/rand/v2"
	"sort"
	"sync"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
)

// Mob loot as the original server rolls it (CMob loot path, recovered by
// Eternity's mob_drops): MobItems orders 1-6 use a per-cent rate and 7-9 a
// per-million rate (order 0 is unused); each hit picks a random item of
// that grade, subject to the item's Drop time limit. World drops use a
// separate roll per eligible rule. Gold drops 30% of the time as
// Money1 + rand(Money2).

func chanceSucceeds(percent uint32) bool {
	switch {
	case percent == 0:
		return false
	case percent >= 100:
		return true
	}
	return rand.Uint32N(100) < percent
}

// dropLimitIntervals are the original Drop selectors (ms per window).
var dropLimitIntervals = [7]uint32{0, 1440000, 14400000, 28800000, 43200000, 86400000, 259200000}

type gradeEntry struct {
	def *db.ItemDef
	// Drop limit: at most one drop per window, the window start
	// randomized within its last tenth.
	limited          bool
	delay, randDelay time.Duration
	nextAt           time.Time
}

var dropCatalog struct {
	mu         sync.Mutex
	grades     map[uint16][]*gradeEntry
	worldRules []worlddrops.Rule
}

func isQuestItemType(t uint8) bool { return t == 27 || t == 28 || t == 29 }

// buildDropCatalog groups items by grade, unrestricted items first, then
// those with a Server or Drop restriction, as CGameData's init does.
func buildDropCatalog() {
	defs := db.MustDefs()
	var all []*db.ItemDef
	for _, d := range defs.Items {
		if d.Grade > 0 && d.Grade <= 1024 && d.Type != 0 && d.TypeID != 0 && !isQuestItemType(d.Type) {
			all = append(all, d)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Type != all[j].Type {
			return all[i].Type < all[j].Type
		}
		return all[i].TypeID < all[j].TypeID
	})
	grades := map[uint16][]*gradeEntry{}
	for pass := 0; pass < 2; pass++ {
		for _, d := range all {
			restricted := (d.Server != 0 && d.Server < 10) || d.Drop != 0
			if restricted != (pass == 1) {
				continue
			}
			e := &gradeEntry{def: d}
			if d.Drop != 0 && int(d.Drop) < len(dropLimitIntervals) {
				interval := time.Duration(dropLimitIntervals[d.Drop]) * time.Millisecond
				e.limited, e.randDelay = true, interval/10
				e.delay = interval - e.randDelay
			}
			grades[d.Grade] = append(grades[d.Grade], e)
		}
	}
	dropCatalog.mu.Lock()
	dropCatalog.grades = grades
	dropCatalog.mu.Unlock()
}

func loadWorldDrops(log *logx.Logger) error {
	pool, err := db.Pool(db.GameDefs)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()
	rules, err := worlddrops.Read(ctx, pool)
	if err != nil {
		return err
	}
	dropCatalog.mu.Lock()
	dropCatalog.worldRules = rules
	dropCatalog.mu.Unlock()
	log.Printf("GX Server World loaded world drop rules=%d", len(rules))
	return nil
}

// pickGradeItem is MobDropCatalog::select: a random item of the grade,
// dropped only if its time limit allows it.
func pickGradeItem(grade uint16) *db.ItemDef {
	dropCatalog.mu.Lock()
	defer dropCatalog.mu.Unlock()
	list := dropCatalog.grades[grade]
	if len(list) == 0 {
		return nil
	}
	e := list[rand.IntN(len(list))]
	if e.limited {
		now := time.Now()
		if now.Before(e.nextAt) {
			return nil
		}
		jitter := time.Duration(0)
		if e.randDelay > 0 {
			jitter = time.Duration(rand.Int64N(int64(e.randDelay)))
		}
		e.nextAt = now.Add(e.delay + jitter)
	}
	return e.def
}

func rollWorldDrops(level uint16, roll func(uint64) uint64, drop func(worlddrops.Rule)) {
	dropCatalog.mu.Lock()
	rules := dropCatalog.worldRules
	dropCatalog.mu.Unlock()
	for _, rule := range rules {
		if level >= rule.MinLevel && level <= rule.MaxLevel && rule.DropChance > 0 &&
			roll(worlddrops.RollRange) < rule.DropChance {
			drop(rule)
		}
	}
}

// dropPosition is CMob::GetDropPosRand: within half a unit of the mob, with
// Y 0 (the server has no terrain heights; the client places it).
func dropPosition(x, z float32) (float32, float32) {
	return x + float32(rand.IntN(1000))/1000 - 0.5, z + float32(rand.IntN(1000))/1000 - 0.5
}

// rollMobDrops rolls one kill's loot, registers it on the map and returns
// the new map items.
//
// With two or more party members in range, the gold is split evenly among
// them instead of dropped (at least 1 each).
func rollMobDrops(mapID uint16, mob *MapMob, ownerID uint32, party []*ConnectedPlayer, log logPrinter) []MapItem {
	defs := db.MustDefs()
	var packets []MapItem
	place := func(def *db.ItemDef, money uint32) {
		x, z := dropPosition(mob.PosX, mob.PosZ)
		mi := MapItem{ID: nextMapItemID(), OwnerID: ownerID, FromMob: true, DroppedAt: time.Now(), PosX: x, PosZ: z}
		if def != nil {
			mi.Item = newDroppedItem(def)
		} else {
			mi.Item, mi.Money = db.Item{Type: moneyItemType, Count: 1}, money
		}
		addMapItem(mapID, mi)
		packets = append(packets, mi)
	}
	for _, d := range defs.MobDrops[mob.MobID] {
		if d.ItemOrder < 1 || d.ItemOrder > 9 {
			continue
		}
		var roll uint32
		if d.ItemOrder < 7 {
			roll = rand.Uint32N(100)
		} else {
			roll = rand.Uint32N(1_000_000)
		}
		if roll >= d.DropRate {
			continue
		}
		if def := pickGradeItem(d.Grade); def != nil {
			place(def, 0)
			log.Printf("GX Server World mob drop mob_id=%d order=%d grade=%d item=%d/%d", mob.MobID, d.ItemOrder, d.Grade, def.Type, def.TypeID)
		}
	}
	rollWorldDrops(mob.Level, rand.Uint64N, func(rule worlddrops.Rule) {
		if def := pickGradeItem(rule.Grade); def != nil {
			place(def, 0)
			log.Printf("GX Server World world drop mob_id=%d rule=%d grade=%d item=%d/%d", mob.MobID, rule.Index, rule.Grade, def.Type, def.TypeID)
		}
	})
	if grbKill(mapID, mob, ownerID) {
		return packets
	}
	if rand.IntN(100) < 30 {
		amount := mob.MoneyMin
		if mob.MoneyMax > 0 {
			amount += rand.Uint32N(mob.MoneyMax)
		}
		switch {
		case amount == 0:
		case len(party) > 1:
			share := max(amount/uint32(len(party)), 1)
			for _, m := range party {
				m.QueueGold(share)
			}
		default:
			place(nil, amount)
		}
	}
	return packets
}

func newDroppedItem(def *db.ItemDef) db.Item {
	it := db.Item{Type: def.Type, TypeID: def.TypeID, Quality: def.Quality, Count: 1}
	if def.Server > 0 && def.ReqWis > 0 {
		it.CraftName = rollCraftName(min(def.Server, 9), uint8(min(def.ReqWis, 99)), rollsStatsOnly(def.Type), "")
	}
	return it
}
