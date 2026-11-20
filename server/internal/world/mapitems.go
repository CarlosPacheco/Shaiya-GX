package world

import (
	"sort"
	"sync"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const (
	// A dropped item is reserved for its owner for 30 s, open to anyone
	// until 60 s, then despawns.
	mapItemOwnerWindow = 30 * time.Second
	mapItemDespawnAge  = 60 * time.Second
	moneyItemType      = 26
)

// MapItem is an item or gold lying on a map.
type MapItem struct {
	ID               uint32
	Item             db.Item
	Money            uint32
	OwnerID          uint32
	NoParty          bool // a player's drop: its party does not share it
	FromMob          bool // mob loot (drop type 1): gold can receive the mob-gold bonus
	DroppedAt        time.Time
	PosX, PosY, PosZ float32
}

type mapItemEntry struct {
	item    MapItem
	claimed bool
}
type itemClaim struct {
	shard *mapShard
	entry *mapItemEntry
	once  sync.Once
}

func nextMapItemID() uint32 { return max(realm.nextMapItem.Add(1), 1) }
func addMapItem(mapID uint16, it MapItem) {
	s := shard(mapID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items == nil {
		s.items = map[uint32]*mapItemEntry{}
	}
	s.items[it.ID] = &mapItemEntry{item: it}
}
func findMapItem(mapID uint16, id uint32) (MapItem, bool) {
	s := shard(mapID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry := s.items[id]; entry != nil {
		return entry.item, true
	}
	return MapItem{}, false
}

// Claim reserves one ground object across concurrent players without holding a
// map mutex during SQLite IO. Failed persistence releases the unchanged object.
func claimMapItem(mapID uint16, id uint32) (MapItem, *itemClaim, bool) {
	s := shard(mapID)
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.items[id]
	if entry == nil || entry.claimed {
		return MapItem{}, nil, false
	}
	entry.claimed = true
	return entry.item, &itemClaim{shard: s, entry: entry}, true
}
func (c *itemClaim) Abort() {
	c.once.Do(func() { c.shard.mu.Lock(); c.entry.claimed = false; c.shard.mu.Unlock() })
}
func (c *itemClaim) Finish(left uint16) (updated MapItem) {
	c.once.Do(func() {
		s := c.shard
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.items[c.entry.item.ID] != c.entry {
			return
		}
		if left == 0 {
			delete(s.items, c.entry.item.ID)
		} else {
			c.entry.item.Item.Count = left
			c.entry.claimed = false
			updated = c.entry.item
		}
	})
	return
}
func itemsOnMap(mapID uint16) []MapItem {
	s := shard(mapID)
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]MapItem, 0, len(s.items))
	for _, entry := range s.items {
		result = append(result, entry.item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

type expiredItem struct {
	MapID uint16
	Item  MapItem
}

func sweepExpiredMapItems() []expiredItem {
	var result []expiredItem
	for _, id := range shardIDs() {
		s := shard(id)
		s.mu.Lock()
		for key, entry := range s.items {
			if !entry.claimed && time.Since(entry.item.DroppedAt) >= mapItemDespawnAge {
				result = append(result, expiredItem{id, entry.item})
				delete(s.items, key)
			}
		}
		s.mu.Unlock()
	}
	return result
}

func buildMapAddItem(m *MapItem) []byte {
	w := packet.New(packet.MapAddItem)
	w.U32(m.ID)
	w.U8(1)
	if m.Money > 0 {
		w.Bytes([]byte{moneyItemType, 0})
		w.Count(1)
	} else {
		w.Bytes([]byte{m.Item.Type, m.Item.TypeID})
		w.Count(m.Item.Count)
	}
	w.F32(m.PosX)
	w.F32(m.PosY)
	w.F32(m.PosZ) // the client reads no owner (0x0058f320)
	return w.MustFinish()
}

func buildMapRemoveItem(id uint32) []byte {
	w := packet.New(packet.MapRemoveItem)
	w.U32(id)
	return w.MustFinish()
}

// clearMapItems drops every item of a zone (instance teardown).
func clearMapItems(zone uint16) {
	realm.shardsMu.Lock()
	s := realm.shards[zone]
	realm.shardsMu.Unlock()
	if s == nil {
		return
	}
	s.mu.Lock()
	s.items = nil
	s.mu.Unlock()
}
