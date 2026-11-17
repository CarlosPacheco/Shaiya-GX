# Protocol: Quest lifecycle

> Generated from Go source by `tools/generate_server_reference.py`; do not edit by hand.

[All opcodes](server-opcodes.md) · [Wire conventions](server-protocol.md) · [Server architecture](server-architecture.md)

Client codecs operate on opcode + body; server packet.Writer emits length + opcode + body. Excerpts preserve the respective construction API. Go wire methods specify widths, not host struct alignment. `Count` is special: see the stack-width compatibility rules. Loops, conditions, helper calls and fixed string sizes are significant; do not flatten them into a single fixed-size struct.

## Opcode 0901

**`0x0901` — QuestList**

Declarations: [packet.QuestList](../../server/internal/packet/packet.go#L124), [proto.QuestList](../../internal/proto/opcodes.go#L109).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [questListPackets (server/internal/world/quests.go)](../../server/internal/world/quests.go#L230)

**Client receive reducers:** [questMapStateChanged (internal/game/quest_map.go)](../../internal/game/quest_map.go#L13), [Apply (internal/game/state.go)](../../internal/game/state.go#L568)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>questListPackets — server/internal/world/quests.go</summary>

[Source definition](../../server/internal/world/quests.go#L230)

```go
func (s *Session) questListPackets() [][]byte {
	var out [][]byte
	for start := 0; start == 0 || start < len(s.quests); start += 255 {
		end := min(start+255, len(s.quests))
		w := packet.New(packet.QuestList)
		w.U8(uint8(end - start))
		for _, q := range s.quests[start:end] {
			w.U16(q.ID)
			left := uint16(0)
			if q.def.MaxMinutes != 0 {
				left = uint16(max(time.Until(q.endAt), 0) / time.Minute)
			}
			w.U16(left)
			w.Bytes(q.counters[:])
		}
		out = append(out, w.MustFinish())
	}
	for start := 0; start == 0 || start < len(s.finishedQuests); start += 255 {
		end := min(start+255, len(s.finishedQuests))
		w := packet.New(packet.QuestFinishedList)
		w.U8(uint8(end - start))
		for _, f := range s.finishedQuests[start:end] {
			w.U16(f.QuestID)
			w.U8(f.Success)
		}
		out = append(out, w.MustFinish())
	}
	return out
}
```

</details>

<details><summary>DecodeQuestList — internal/proto/quest.go</summary>

[Source definition](../../internal/proto/quest.go#L20)

```go
func DecodeQuestList(p []byte) ([]ActiveQuest, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/7 {
		return nil, ErrShort
	}
	out := make([]ActiveQuest, n)
	for i := range out {
		q := &out[i]
		q.ID = r.U16()
		q.MinutesLeft = r.U16()
		for j := range q.Counters {
			q.Counters[j] = r.U8()
		}
	}
	return out, r.Err()
}
```

```go
type ActiveQuest struct {
	ID, MinutesLeft uint16
	Counters        [3]uint8
}
```

</details>

## Opcode 0902

**`0x0902` — QuestStart**

Declarations: [packet.QuestStart](../../server/internal/packet/packet.go#L125), [proto.QuestStart](../../internal/proto/opcodes.go#L110).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L99)

**S→C serializers:** [startQuest (server/internal/world/quests.go)](../../server/internal/world/quests.go#L406)

**Client receive reducers:** [questMapStateChanged (internal/game/quest_map.go)](../../internal/game/quest_map.go#L13), [Apply (internal/game/state.go)](../../internal/game/state.go#L590)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>startQuest — server/internal/world/quests.go</summary>

[Source definition](../../server/internal/world/quests.go#L406)

```go
func (s *Session) startQuest(q *questDef, id uint16, npcID uint32) {
	now := time.Now()
	aq := &activeQuest{ID: id, def: q, startAt: now}
	if q.MaxMinutes != 0 {
		aq.endAt = now.Add(time.Duration(q.MaxMinutes) * time.Minute)
	}
	if q.MinMinutes != 0 {
		aq.successAt = now.Add(time.Duration(q.MinMinutes) * time.Minute)
	}
	if err := db.AddCharacterQuest(s.char.ID, db.CharQuest{QuestID: id, Delay: int16(min(q.MaxMinutes, 32767))}); err != nil {
		s.log.Printf("GX Server World quest %d start failed: %v", id, err)
		return
	}
	s.quests = append(s.quests, aq)
	w := packet.New(packet.QuestStart)
	w.U32(npcID)
	w.U16(id)
	s.send(w.MustFinish(), "0x0902 quest start")
	for _, it := range q.StartItems {
		if it.Type != 0 {
			s.giveQuestItem(it.Type, it.TypeID, max(it.Count, 1))
		}
	}
	s.log.Printf("GX Server World quest %d %q started", id, q.Name)
}
```

</details>

<details><summary>QuestStartPacket — internal/proto/quest.go</summary>

[Source definition](../../internal/proto/quest.go#L80)

```go
func QuestStartPacket(npc uint32, id uint16) []byte {
	w := NewWriter(QuestStart)
	w.U32(npc)
	w.U16(id)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeQuestStart — internal/proto/quest.go</summary>

[Source definition](../../internal/proto/quest.go#L49)

```go
func DecodeQuestStart(p []byte) (npc uint32, id uint16, err error) {
	r := NewReader(p)
	npc = r.U32()
	id = r.U16()
	return npc, id, r.Err()
}
```

</details>

## Opcode 0903

**`0x0903` — QuestEnd**

Declarations: [packet.QuestEnd](../../server/internal/packet/packet.go#L126), [proto.QuestEnd](../../internal/proto/opcodes.go#L111).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L101)

**S→C serializers:** [completeQuest (server/internal/world/quests.go)](../../server/internal/world/quests.go#L576), [awardQuest (server/internal/world/quests.go)](../../server/internal/world/quests.go#L653)

**Client receive reducers:** [questMapStateChanged (internal/game/quest_map.go)](../../internal/game/quest_map.go#L13), [Apply (internal/game/state.go)](../../internal/game/state.go#L617)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>completeQuest — server/internal/world/quests.go</summary>

[Source definition](../../server/internal/world/quests.go#L576)

```go
func (s *Session) completeQuest(q *activeQuest, ctx uint32, success bool) {
	d := q.def
	if !success {
		w := packet.New(packet.QuestEnd)
		w.U32(ctx)
		w.U16(q.ID)
		w.Bytes(make([]byte, 14))
		s.send(w.MustFinish(), "0x0903 quest failed")
		s.removeQuest(q, 0)
		s.log.Printf("GX Server World quest %d failed", q.ID)
		return
	}
	if d.MinMinutes != 0 && time.Now().Before(q.successAt) {
		return
	}
	if d.ResultType == 1 {
		s.awardQuest(q, 0, ctx)
		return
	}
	if d.UserSelect != 0 {
		q.success = 1
		w := packet.New(packet.QuestChoose)
		w.U16(q.ID)
		s.send(w.MustFinish(), "0x0907 quest reward choice")
		return
	}
	for v := 2; v >= 0; v-- {
		var sel, req uint16
		switch d.EndType {
		case 1:
			sel, req = uint16(d.Results[v].NeedItemID), uint16(d.Results[v].NeedItemCount)
			if sel > 3 {
				sel = 0
			}
		case 2:
			sel, req = d.Results[v].NeedMobID, uint16(d.Results[v].NeedMobCount)
			if sel > 2 {
				sel = 0
			}
		}
		if v == 0 || (sel != 0 && req != 0 && uint16(q.counters[sel-1]) >= req) {
			s.awardQuest(q, uint8(v), ctx)
			return
		}
	}
}
```

</details>

<details><summary>awardQuest — server/internal/world/quests.go</summary>

[Source definition](../../server/internal/world/quests.go#L653)

```go
func (s *Session) awardQuest(q *activeQuest, variant uint8, ctx uint32) {
	if s.char == nil || q == nil || s.findQuest(q.ID) != q || variant >= 3 {
		return
	}
	res := q.def.Results[variant]
	exp := res.Exp
	if q.def.Mode <= 2 {
		exp += exp
	}
	change := db.CharacterChange{Before: *s.char, After: *s.char, ItemsBefore: s.items, ItemsAfter: append([]db.Item(nil), s.items...)}
	var packets [][]byte
	if q.def.EndType == 1 {
		plannedQuest := *q
		var ok bool
		change.ItemsAfter, packets, ok = planQuestItems(&plannedQuest, s.items, true)
		if !ok {
			return
		}
	}
	var reward *db.Item
	if res.ItemType != 0 {
		def := db.MustDefs().Item(res.ItemType, res.ItemTypeID)
		if def == nil {
			s.log.Printf("GX Server World quest %d has invalid reward definition", q.ID)
			return
		}
		tpl := db.Item{Type: res.ItemType, TypeID: res.ItemTypeID, Quality: def.Quality, MakeType: "Q"}
		adds, left := planGiveItem(&change.ItemsAfter, &tpl, 1, def.MaxStack())
		if left != 0 {
			s.send(buildAddItemError(1), "quest reward inventory full")
			return
		}
		packets = append(packets, adds...)
		for i := len(change.ItemsAfter) - 1; i >= 0; i-- {
			it := &change.ItemsAfter[i]
			if it.Type == tpl.Type && it.TypeID == tpl.TypeID {
				reward = it
				break
			}
		}
	}
	change.After.Money = uint32(min(uint64(change.After.Money)+uint64(res.Money), db.MoneyCap))
	levels := applyExpGain(&change.After, exp)
	if levels > 0 {
		st := StatsFor(&change.After, s.equipment())
		change.After.CurrentHP, change.After.CurrentMP, change.After.CurrentSP = st.MaxHP, st.MaxMP, st.MaxSP
	}
	completion := db.QuestCompletion{CharacterID: s.char.ID, QuestID: q.ID, Repeatable: q.def.TickReceive != 0}
	if err := db.CommitGameplay([]db.CharacterChange{change}, &completion); err != nil {
		s.log.Printf("GX Server World quest %d transaction failed: %v", q.ID, err)
		return
	}
	// Publishing is deliberately after commit: failures retain the active quest,
	// required items, character stats, gold and reward-choice state unchanged.
	*s.char = change.After
	s.items = change.ItemsAfter
	s.removeQuestFromMemory(q, 1)
	s.syncPlayer()
	s.partyPools()
	if s.player != nil {
		s.player.UpdateItems(s.items)
	}
	for _, p := range packets {
		s.send(p, "quest committed item")
	}
	if res.Money != 0 {
		s.send(buildSetMoney(s.char.Money), "0x0213 quest gold")
	}
	w := packet.New(packet.QuestEnd)
	w.U32(ctx)
	w.U16(q.ID)
	w.U8(1)
	w.U8(variant)
	w.U32(exp)
	w.U32(res.Money)
	if reward != nil {
		w.Bytes([]byte{reward.Bag, reward.Slot, reward.Type, reward.TypeID})
	} else {
		w.Bytes(make([]byte, 4))
	}
	s.send(w.MustFinish(), "0x0903 quest completed")
	s.notifyExpGain(exp, levels)
	s.log.Printf("GX Server World quest %d committed variant %d exp %d money %d", q.ID, variant, exp, res.Money)
}
```

</details>

<details><summary>QuestEndPacket — internal/proto/quest.go</summary>

[Source definition](../../internal/proto/quest.go#L86)

```go
func QuestEndPacket(npc uint32, id uint16) []byte {
	w := NewWriter(QuestEnd)
	w.U32(npc)
	w.U16(id)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeQuestEnd — internal/proto/quest.go</summary>

[Source definition](../../internal/proto/quest.go#L62)

```go
func DecodeQuestEnd(p []byte) (q QuestEndResult, err error) {
	r := NewReader(p)
	q.NPC = r.U32()
	q.ID = r.U16()
	q.Success = r.U8()
	q.Variant = r.U8()
	q.Experience = r.U32()
	q.Money = r.U32()
	for i := range q.Item {
		q.Item[i] = r.U8()
	}
	return q, r.Err()
}
```

```go
type QuestEndResult struct {
	NPC               uint32
	ID                uint16
	Success, Variant  uint8
	Experience, Money uint32
	Item              [4]uint8
}
```

</details>

## Opcode 0904

**`0x0904` — QuestStartAlt**

Declarations: [packet.QuestStartAlt](../../server/internal/packet/packet.go#L127), [proto.QuestStartAlt](../../internal/proto/opcodes.go#L112).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L99)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

The shape is decoded inline, delegated, or only declared. Use the linked adapter/reducer and the codec appendix below; there is no inferred fixed layout here.

## Opcode 0905

**`0x0905` — QuestCounter**

Declarations: [packet.QuestCounter](../../server/internal/packet/packet.go#L128), [proto.QuestCounter](../../internal/proto/opcodes.go#L113).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [bumpQuestCounter (server/internal/world/quests.go)](../../server/internal/world/quests.go#L766), [questKillCount (server/internal/world/quests.go)](../../server/internal/world/quests.go#L785)

**Client receive reducers:** [questMapStateChanged (internal/game/quest_map.go)](../../internal/game/quest_map.go#L13), [Apply (internal/game/state.go)](../../internal/game/state.go#L599)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>bumpQuestCounter — server/internal/world/quests.go</summary>

[Source definition](../../server/internal/world/quests.go#L766)

```go
func (s *Session) bumpQuestCounter(q *activeQuest, slot int) {
	if q.counters[slot] == 255 {
		return
	}
	q.counters[slot]++
	if err := db.SaveCharacterQuest(s.char.ID, q.row()); err != nil {
		q.counters[slot]--
		s.logErr("save quest counter", err)
		return
	}
	w := packet.New(packet.QuestCounter)
	w.U16(q.ID)
	w.U8(uint8(slot))
	w.U8(q.counters[slot])
	s.send(w.MustFinish(), "0x0905 quest counter")
}
```

</details>

<details><summary>questKillCount — server/internal/world/quests.go</summary>

[Source definition](../../server/internal/world/quests.go#L785)

```go
func (s *Session) questKillCount(mobID uint16, count uint8) bool {
	if s.char == nil {
		return true
	}
	type update struct {
		quest    *activeQuest
		counters [3]uint8
	}
	var changes []update
	var rows []db.CharQuest
	var packets [][]byte
	now := time.Now()
	for _, id := range mobQuestLinks[mobID] {
		q := s.findQuest(id)
		if q == nil || q.def.EndType != 2 || (q.def.MinMinutes != 0 && now.After(q.successAt)) {
			continue
		}
		planned := *q
		changed := false
		for slot := 0; slot < 2; slot++ {
			if q.def.MobIDs[slot] != mobID {
				continue
			}
			value := uint8(min(uint16(planned.counters[slot])+uint16(count), 255))
			if value == planned.counters[slot] {
				continue
			}
			planned.counters[slot] = value
			changed = true
			w := packet.New(packet.QuestCounter)
			w.U16(q.ID)
			w.U8(uint8(slot))
			w.U8(value)
			packets = append(packets, w.MustFinish())
		}
		if changed {
			rows = append(rows, planned.row())
			changes = append(changes, update{q, planned.counters})
		}
	}
	if err := db.SaveCharacterQuests(s.char.ID, rows); err != nil {
		s.logErr("quest kill transaction", err)
		return false
	}
	for _, change := range changes {
		change.quest.counters = change.counters
	}
	for _, frame := range packets {
		s.send(frame, "0x0905 committed quest counter")
	}
	for _, change := range changes {
		q := change.quest
		if s.findQuest(q.ID) == q && q.def.EndNpcType == 0 && q.counters[0] >= q.def.MobCounts[0] && q.counters[1] >= q.def.MobCounts[1] {
			s.completeQuest(q, 0, true)
		}
	}
	return true
}
```

</details>

<details><summary>DecodeQuestCounter — internal/proto/quest.go</summary>

[Source definition](../../internal/proto/quest.go#L55)

```go
func DecodeQuestCounter(p []byte) (id uint16, slot, value uint8, err error) {
	r := NewReader(p)
	id = r.U16()
	slot = r.U8()
	value = r.U8()
	return id, slot, value, r.Err()
}
```

</details>

## Opcode 0906

**`0x0906` — QuestFinishedList**

Declarations: [packet.QuestFinishedList](../../server/internal/packet/packet.go#L132), [proto.QuestFinishedList](../../internal/proto/opcodes.go#L117).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [questListPackets (server/internal/world/quests.go)](../../server/internal/world/quests.go#L230)

**Client receive reducers:** [questMapStateChanged (internal/game/quest_map.go)](../../internal/game/quest_map.go#L13), [Apply (internal/game/state.go)](../../internal/game/state.go#L579)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>questListPackets — server/internal/world/quests.go</summary>

[Source definition](../../server/internal/world/quests.go#L230)

```go
func (s *Session) questListPackets() [][]byte {
	var out [][]byte
	for start := 0; start == 0 || start < len(s.quests); start += 255 {
		end := min(start+255, len(s.quests))
		w := packet.New(packet.QuestList)
		w.U8(uint8(end - start))
		for _, q := range s.quests[start:end] {
			w.U16(q.ID)
			left := uint16(0)
			if q.def.MaxMinutes != 0 {
				left = uint16(max(time.Until(q.endAt), 0) / time.Minute)
			}
			w.U16(left)
			w.Bytes(q.counters[:])
		}
		out = append(out, w.MustFinish())
	}
	for start := 0; start == 0 || start < len(s.finishedQuests); start += 255 {
		end := min(start+255, len(s.finishedQuests))
		w := packet.New(packet.QuestFinishedList)
		w.U8(uint8(end - start))
		for _, f := range s.finishedQuests[start:end] {
			w.U16(f.QuestID)
			w.U8(f.Success)
		}
		out = append(out, w.MustFinish())
	}
	return out
}
```

</details>

<details><summary>DecodeFinishedQuests — internal/proto/quest.go</summary>

[Source definition](../../internal/proto/quest.go#L37)

```go
func DecodeFinishedQuests(p []byte) ([]FinishedQuest, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/3 {
		return nil, ErrShort
	}
	out := make([]FinishedQuest, n)
	for i := range out {
		out[i] = FinishedQuest{r.U16(), r.U8()}
	}
	return out, r.Err()
}
```

```go
type FinishedQuest struct {
	ID      uint16
	Success uint8
}
```

</details>

## Opcode 0907

**`0x0907` — QuestChoose**

Declarations: [packet.QuestChoose](../../server/internal/packet/packet.go#L129), [proto.QuestChoose](../../internal/proto/opcodes.go#L114).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L103)

**S→C serializers:** [completeQuest (server/internal/world/quests.go)](../../server/internal/world/quests.go#L576)

**Client receive reducers:** [questMapStateChanged (internal/game/quest_map.go)](../../internal/game/quest_map.go#L13), [Apply (internal/game/state.go)](../../internal/game/state.go#L611)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>completeQuest — server/internal/world/quests.go</summary>

[Source definition](../../server/internal/world/quests.go#L576)

```go
func (s *Session) completeQuest(q *activeQuest, ctx uint32, success bool) {
	d := q.def
	if !success {
		w := packet.New(packet.QuestEnd)
		w.U32(ctx)
		w.U16(q.ID)
		w.Bytes(make([]byte, 14))
		s.send(w.MustFinish(), "0x0903 quest failed")
		s.removeQuest(q, 0)
		s.log.Printf("GX Server World quest %d failed", q.ID)
		return
	}
	if d.MinMinutes != 0 && time.Now().Before(q.successAt) {
		return
	}
	if d.ResultType == 1 {
		s.awardQuest(q, 0, ctx)
		return
	}
	if d.UserSelect != 0 {
		q.success = 1
		w := packet.New(packet.QuestChoose)
		w.U16(q.ID)
		s.send(w.MustFinish(), "0x0907 quest reward choice")
		return
	}
	for v := 2; v >= 0; v-- {
		var sel, req uint16
		switch d.EndType {
		case 1:
			sel, req = uint16(d.Results[v].NeedItemID), uint16(d.Results[v].NeedItemCount)
			if sel > 3 {
				sel = 0
			}
		case 2:
			sel, req = d.Results[v].NeedMobID, uint16(d.Results[v].NeedMobCount)
			if sel > 2 {
				sel = 0
			}
		}
		if v == 0 || (sel != 0 && req != 0 && uint16(q.counters[sel-1]) >= req) {
			s.awardQuest(q, uint8(v), ctx)
			return
		}
	}
}
```

</details>

<details><summary>QuestChoosePacket — internal/proto/quest.go</summary>

[Source definition](../../internal/proto/quest.go#L92)

```go
func QuestChoosePacket(id uint16, choice uint8) []byte {
	w := NewWriter(QuestChoose)
	w.U16(id)
	w.U8(choice)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeQuestChoice — internal/proto/quest.go</summary>

[Source definition](../../internal/proto/quest.go#L75)

```go
func DecodeQuestChoice(p []byte) (uint16, error) {
	r := NewReader(p)
	id := r.U16()
	return id, r.Err()
}
```

</details>

## Opcode 0908

**`0x0908` — QuestAbandon**

Declarations: [packet.QuestAbandon](../../server/internal/packet/packet.go#L130), [proto.QuestAbandon](../../internal/proto/opcodes.go#L115).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L105)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>QuestAbandonPacket — internal/proto/quest.go</summary>

[Source definition](../../internal/proto/quest.go#L98)

```go
func QuestAbandonPacket(id uint16) []byte { w := NewWriter(QuestAbandon); w.U16(id); return w.Bytes() }
```

</details>
