# Protocol: Combat, skills and character resources

> Generated from Go source by `tools/generate_server_reference.py`; do not edit by hand.

[All opcodes](server-opcodes.md) · [Wire conventions](server-protocol.md) · [Server architecture](server-architecture.md)

Client codecs operate on opcode + body; server packet.Writer emits length + opcode + body. Excerpts preserve the respective construction API. Go wire methods specify widths, not host struct alignment. `Count` is special: see the stack-width compatibility rules. Loops, conditions, helper calls and fixed string sizes are significant; do not flatten them into a single fixed-size struct.

## Opcode 0501

**`0x0501` — CharacterMove**

Declarations: [packet.CharacterMove](../../server/internal/packet/packet.go#L92), [proto.CharacterMove](../../internal/proto/opcodes.go#L76).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L198)

**S→C serializers:** [buildCharacterMove (server/internal/world/character.go)](../../server/internal/world/character.go#L533)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1399)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildCharacterMove — server/internal/world/character.go</summary>

[Source definition](../../server/internal/world/character.go#L533)

```go
func buildCharacterMove(ch *db.Character, motion uint8, direction uint16) []byte {
	x, y, z := charPos(ch)
	w := packet.New(packet.CharacterMove)
	w.U32(charID(ch))
	w.U8(motion)
	w.U16(direction)
	w.F32(x)
	w.F32(y)
	w.F32(z)
	return w.MustFinish()
}
```

</details>

<details><summary>MovePacket — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L71)

```go
func MovePacket(motion uint8, dir uint16, x, y, z float32) []byte {
	for _, v := range []float32{x, y, z} {
		if v != v || v > 3.4e38 || v < -3.4e38 {
			return nil
		}
	}
	w := NewWriter(CharacterMove)
	w.U8(motion)
	w.U16(dir)
	w.F32(x)
	w.F32(y)
	w.F32(z)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeUserMove — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L241)

```go
func DecodeUserMove(p []byte) (m UserMove, err error) {
	r := NewReader(p)
	m.ID, m.Motion, m.Dir = r.U32(), r.U8(), r.U16()
	m.X, m.Y, m.Z = r.F32(), r.F32(), r.F32()
	return m, r.Err()
}
```

```go
type UserMove struct {
	ID      uint32
	Motion  uint8
	Dir     uint16
	X, Y, Z float32
}
```

</details>

## Opcode 0502

**`0x0502` — CharacterUserAutoAttack**

Declarations: [packet.CharacterUserAutoAttack](../../server/internal/packet/packet.go#L94), [proto.CharacterUserAutoAttack](../../internal/proto/opcodes.go#L78).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L23), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L243)

**S→C serializers:** [buildCharacterUserAttack (server/internal/world/pvp.go)](../../server/internal/world/pvp.go#L479), [idPacket (internal/proto/world.go)](../../internal/proto/world.go#L39)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1115)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildCharacterUserAttack — server/internal/world/pvp.go</summary>

[Source definition](../../server/internal/world/pvp.go#L479)

```go
func buildCharacterUserAttack(attackerID, targetID uint32, result uint8, damage uint16) []byte {
	w := packet.New(packet.CharacterUserAutoAttack)
	w.U8(result)
	w.U32(attackerID)
	w.U32(targetID)
	w.U16(damage)
	w.U16(0)
	w.U16(0)
	return w.MustFinish()
}
```

</details>

<details><summary>idPacket — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L39)

```go
func idPacket(op Opcode, id uint32) []byte {
	w := NewWriter(op)
	w.U32(id)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeAttackResult — internal/proto/combat.go</summary>

[Source definition](../../internal/proto/combat.go#L32)

```go
func DecodeAttackResult(p []byte) (m AttackResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	return m, r.Err()
}
```

```go
type AttackResult struct {
	Result         uint8
	Source, Target uint32
	HP, SP, MP     uint16
}
```

</details>

## Opcode 0503

**`0x0503` — CharacterMobAutoAttack**

Declarations: [packet.CharacterMobAutoAttack](../../server/internal/packet/packet.go#L95), [proto.CharacterMobAutoAttack](../../internal/proto/opcodes.go#L79).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L23), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L234)

**S→C serializers:** [buildCharacterMobAttack (server/internal/world/entities.go)](../../server/internal/world/entities.go#L1410), [idPacket (internal/proto/world.go)](../../internal/proto/world.go#L39)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1115)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildCharacterMobAttack — server/internal/world/entities.go</summary>

[Source definition](../../server/internal/world/entities.go#L1410)

```go
func buildCharacterMobAttack(characterID, mobID uint32, result uint8, damage uint16) []byte {
	w := packet.New(packet.CharacterMobAutoAttack)
	w.U8(result)
	w.U32(characterID)
	w.U32(mobID)
	w.U16(damage)
	w.U16(0)
	w.U16(0)
	return w.MustFinish()
}
```

</details>

<details><summary>idPacket — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L39)

```go
func idPacket(op Opcode, id uint32) []byte {
	w := NewWriter(op)
	w.U32(id)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeAttackResult — internal/proto/combat.go</summary>

[Source definition](../../internal/proto/combat.go#L32)

```go
func DecodeAttackResult(p []byte) (m AttackResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	return m, r.Err()
}
```

```go
type AttackResult struct {
	Result         uint8
	Source, Target uint32
	HP, SP, MP     uint16
}
```

</details>

## Opcode 0504

**`0x0504` — CharacterDeath**

Declarations: [packet.CharacterDeath](../../server/internal/packet/packet.go#L96), [proto.CharacterDeath](../../internal/proto/opcodes.go#L80).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildCharacterDeath (server/internal/world/entities.go)](../../server/internal/world/entities.go#L1432)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1121)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildCharacterDeath — server/internal/world/entities.go</summary>

[Source definition](../../server/internal/world/entities.go#L1432)

```go
func buildCharacterDeath(characterID uint32, deathType uint8, killerID uint32) []byte {
	w := packet.New(packet.CharacterDeath)
	w.U32(characterID)
	w.U8(deathType)
	w.U32(killerID)
	return w.MustFinish()
}
```

</details>

<details><summary>NewReader — internal/proto/rw.go</summary>

[Source definition](../../internal/proto/rw.go#L28)

```go
func NewReader(payload []byte) *Reader {
	r := &Reader{b: payload, off: 2}
	if len(payload) >= 2 {
		r.wide = itemstack.Wide(binary.LittleEndian.Uint16(payload))
	}
	if len(payload) < 2 {
		r.err = ErrShort
	}
	return r
}
```

```go
type Reader struct {
	wide bool
	b    []byte
	off  int
	err  error
}
```

</details>

## Opcode 0505

**`0x0505` — CharacterReviveHitpoints**

Declarations: [packet.CharacterReviveHitpoints](../../server/internal/packet/packet.go#L93), [proto.CharacterReviveHitpoints](../../internal/proto/opcodes.go#L77).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildCharacterReviveHitpoints (server/internal/world/entities.go)](../../server/internal/world/entities.go#L1467)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1089)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildCharacterReviveHitpoints — server/internal/world/entities.go</summary>

[Source definition](../../server/internal/world/entities.go#L1467)

```go
func buildCharacterReviveHitpoints(characterID, hp, mp, sp uint32) []byte {
	w := packet.New(packet.CharacterReviveHitpoints)
	w.U32(characterID)
	w.U32(hp)
	w.U32(mp)
	w.U32(sp)
	return w.MustFinish()
}
```

</details>

<details><summary>NewReader — internal/proto/rw.go</summary>

[Source definition](../../internal/proto/rw.go#L28)

```go
func NewReader(payload []byte) *Reader {
	r := &Reader{b: payload, off: 2}
	if len(payload) >= 2 {
		r.wide = itemstack.Wide(binary.LittleEndian.Uint16(payload))
	}
	if len(payload) < 2 {
		r.err = ErrShort
	}
	return r
}
```

```go
type Reader struct {
	wide bool
	b    []byte
	off  int
	err  error
}
```

</details>

## Opcode 0506

**`0x0506` — CharacterMotion**

Declarations: [packet.CharacterMotion](../../server/internal/packet/packet.go#L97), [proto.CharacterMotion](../../internal/proto/opcodes.go#L81).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L23), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L200)

**S→C serializers:** [buildCharacterMotion (server/internal/world/character.go)](../../server/internal/world/character.go#L545)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L268)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildCharacterMotion — server/internal/world/character.go</summary>

[Source definition](../../server/internal/world/character.go#L545)

```go
func buildCharacterMotion(characterID uint32, motion uint8) []byte {
	w := packet.New(packet.CharacterMotion)
	w.U32(characterID)
	w.U8(motion)
	return w.MustFinish()
}
```

</details>

<details><summary>MotionPacket — internal/proto/motion.go</summary>

[Source definition](../../internal/proto/motion.go#L31)

```go
func MotionPacket(motion uint8) []byte {
	w := NewWriter(CharacterMotion)
	w.U8(motion)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeCharacterMotion — internal/proto/motion.go</summary>

[Source definition](../../internal/proto/motion.go#L37)

```go
func DecodeCharacterMotion(p []byte) (CharacterMotionResult, error) {
	r := NewReader(p)
	m := CharacterMotionResult{ID: r.U32(), Motion: r.U8()}
	if r.Err() != nil {
		return CharacterMotionResult{}, r.Err()
	}
	if r.Remaining() != 0 {
		return CharacterMotionResult{}, fmt.Errorf("motion: trailing bytes")
	}
	return m, nil
}
```

```go
type CharacterMotionResult struct {
	ID     uint32
	Motion uint8
}
```

</details>

<details><summary>GestureAction — internal/proto/motion.go</summary>

[Source definition](../../internal/proto/motion.go#L16)

```go
func GestureAction(motion uint8) (int, bool) {
	if motion < 116 || motion > 125 {
		return 0, false
	}
	if motion >= 124 {
		return int(motion), true
	}
	return int(motion ^ 1), true
}
```

</details>

## Opcode 0507

**`0x0507` — CharacterEquipment**

Declarations: [packet.CharacterEquipment](../../server/internal/packet/packet.go#L98), [proto.CharacterEquipment](../../internal/proto/opcodes.go#L82).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildCharacterEquipment (server/internal/world/character.go)](../../server/internal/world/character.go#L574)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1366)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildCharacterEquipment — server/internal/world/character.go</summary>

[Source definition](../../server/internal/world/character.go#L574)

```go
func buildCharacterEquipment(ch *db.Character, items []db.Item, slot uint8) []byte {
	w := packet.New(packet.CharacterEquipment)
	w.U32(ch.ID)
	w.U8(slot)
	it := equippedItem(items, slot)
	if it != nil {
		w.U8(it.Type)
		w.U8(it.TypeID)
		w.U8(enchantLevel(it))
		// The client always reads 6 more bytes (0x0058f980): the cloak
		// badge, zero for other slots.
		if slot == 7 {
			w.Bytes(it.Gems[:])
		} else {
			w.Bytes(make([]byte, 6))
		}
	} else {
		w.Bytes(make([]byte, 9))
	}
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeEquipment — internal/proto/entities.go</summary>

[Source definition](../../internal/proto/entities.go#L79)

```go
func DecodeEquipment(p []byte) (m EquipmentChange, err error) {
	r := NewReader(p)
	m.ID = r.U32()
	m.Slot, m.Type, m.TypeID, m.Enchant = r.U8(), r.U8(), r.U8(), r.U8()
	for i := range m.Cloak {
		m.Cloak[i] = r.U8()
	}
	return m, r.Err()
}
```

```go
type EquipmentChange struct {
	ID                          uint32
	Slot, Type, TypeID, Enchant uint8
	Cloak                       [6]uint8
}
```

</details>

## Opcode 0508

**`0x0508` — LevelUpSelf**

Declarations: [packet.LevelUpSelf](../../server/internal/packet/packet.go#L73), [proto.LevelUpSelf](../../internal/proto/opcodes.go#L57).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildLevelUp (server/internal/world/combat.go)](../../server/internal/world/combat.go#L610)

**Client receive reducers:** [questMapStateChanged (internal/game/quest_map.go)](../../internal/game/quest_map.go#L13), [Apply (internal/game/state.go)](../../internal/game/state.go#L938)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildLevelUp — server/internal/world/combat.go</summary>

[Source definition](../../server/internal/world/combat.go#L610)

```go
func buildLevelUp(op packet.Opcode, ch *db.Character) []byte {
	prev, next := expBounds(ch)
	w := packet.New(op)
	w.U32(ch.ID)
	w.U16(ch.Level)
	w.U16(ch.StatPoint)
	w.U16(ch.SkillPoint)
	w.U32(prev)
	w.U32(next)
	return w.MustFinish()
}
```

</details>

<details><summary>NewReader — internal/proto/rw.go</summary>

[Source definition](../../internal/proto/rw.go#L28)

```go
func NewReader(payload []byte) *Reader {
	r := &Reader{b: payload, off: 2}
	if len(payload) >= 2 {
		r.wide = itemstack.Wide(binary.LittleEndian.Uint16(payload))
	}
	if len(payload) < 2 {
		r.err = ErrShort
	}
	return r
}
```

```go
type Reader struct {
	wide bool
	b    []byte
	off  int
	err  error
}
```

</details>

## Opcode 050a

**`0x050A` — UseItem**

Declarations: [packet.UseItem](../../server/internal/packet/packet.go#L114), [proto.UseItem](../../internal/proto/opcodes.go#L98).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L23), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L81)

**S→C serializers:** [buildUseItemResult (server/internal/world/inventory.go)](../../server/internal/world/inventory.go#L104)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L751)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildUseItemResult — server/internal/world/inventory.go</summary>

[Source definition](../../server/internal/world/inventory.go#L104)

```go
func buildUseItemResult(characterID uint32, it *db.Item) []byte {
	w := packet.New(packet.UseItem)
	w.U32(characterID)
	if it == nil {
		w.Bytes(make([]byte, 4))
		w.Count(0)
	} else {
		w.Bytes([]byte{it.Bag, it.Slot, it.Type, it.TypeID})
		w.Count(it.Count)
	}
	return w.MustFinish()
}
```

</details>

<details><summary>UseItemPacket — internal/proto/inventory.go</summary>

[Source definition](../../internal/proto/inventory.go#L86)

```go
func UseItemPacket(bag, slot uint8) []byte {
	w := NewWriter(UseItem)
	w.U8(bag)
	w.U8(slot)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeItemUse — internal/proto/inventory.go</summary>

[Source definition](../../internal/proto/inventory.go#L99)

```go
func DecodeItemUse(p []byte) (m ItemUseResult, err error) {
	r := NewReader(p)
	m.ID = r.U32()
	m.Bag, m.Slot, m.Type, m.TypeID, m.Count = r.U8(), r.U8(), r.U8(), r.U8(), r.Count()
	return m, r.Err()
}
```

```go
type ItemUseResult struct {
	ID                      uint32
	Bag, Slot, Type, TypeID uint8
	Count                   uint16
}
```

</details>

## Opcode 050b

**`0x050B` — CharacterMaxEnergy**

Declarations: [packet.CharacterMaxEnergy](../../server/internal/packet/packet.go#L115), [proto.CharacterMaxEnergy](../../internal/proto/opcodes.go#L99).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildMaxEnergy (server/internal/world/character.go)](../../server/internal/world/character.go#L485)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L722)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildMaxEnergy — server/internal/world/character.go</summary>

[Source definition](../../server/internal/world/character.go#L485)

```go
func buildMaxEnergy(ch *db.Character, hitpointType uint8, value uint32) []byte {
	w := packet.New(packet.CharacterMaxEnergy)
	w.U32(charID(ch))
	w.U8(hitpointType)
	w.U32(value)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeMaxEnergy — internal/proto/stats.go</summary>

[Source definition](../../internal/proto/stats.go#L71)

```go
func DecodeMaxEnergy(p []byte) (s MaxEnergy, err error) {
	r := NewReader(p)
	s.ID, s.Kind, s.Value = r.U32(), r.U8(), r.U32()
	return s, r.Err()
}
```

```go
type MaxEnergy struct {
	ID    uint32
	Kind  uint8
	Value uint32
}
```

</details>

## Opcode 050c

**`0x050C` — SkillUsedSpMp**

Declarations: [packet.SkillUsedSpMp](../../server/internal/packet/packet.go#L116), [proto.SkillUsedSpMp](../../internal/proto/opcodes.go#L100).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildSkillUsedSpMp (server/internal/world/skillcast.go)](../../server/internal/world/skillcast.go#L125)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L922)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildSkillUsedSpMp — server/internal/world/skillcast.go</summary>

[Source definition](../../server/internal/world/skillcast.go#L125)

```go
func buildSkillUsedSpMp(sp, mp uint32) []byte {
	w := packet.New(packet.SkillUsedSpMp)
	w.U32(sp)
	w.U32(mp)
	return w.MustFinish()
}
```

</details>

<details><summary>NewReader — internal/proto/rw.go</summary>

[Source definition](../../internal/proto/rw.go#L28)

```go
func NewReader(payload []byte) *Reader {
	r := &Reader{b: payload, off: 2}
	if len(payload) >= 2 {
		r.wide = itemstack.Wide(binary.LittleEndian.Uint16(payload))
	}
	if len(payload) < 2 {
		r.err = ErrShort
	}
	return r
}
```

```go
type Reader struct {
	wide bool
	b    []byte
	off  int
	err  error
}
```

</details>

## Opcode 050d

**`0x050D` — BuffAdd**

Declarations: [packet.BuffAdd](../../server/internal/packet/packet.go#L118), [proto.BuffAdd](../../internal/proto/opcodes.go#L102).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildBuffAdd (server/internal/world/playereffects.go)](../../server/internal/world/playereffects.go#L99)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L679)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildBuffAdd — server/internal/world/playereffects.go</summary>

[Source definition](../../server/internal/world/playereffects.go#L99)

```go
func buildBuffAdd(id uint32, skillID uint16, level uint8) []byte {
	w := packet.New(packet.BuffAdd)
	w.U32(id)
	w.U16(skillID)
	w.U8(level)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeBuffAdd — internal/proto/buffs.go</summary>

[Source definition](../../internal/proto/buffs.go#L27)

```go
func DecodeBuffAdd(p []byte) (b ActiveBuff, err error) {
	r := NewReader(p)
	b.Instance, b.Skill, b.Level = r.U32(), r.U16(), r.U8()
	return b, r.Err()
}
```

```go
type ActiveBuff struct {
	Instance     uint32
	Skill        uint16
	Level        uint8
	Remaining    uint32
	HasRemaining bool
}
```

</details>

## Opcode 050e

**`0x050E` — BuffRemove**

Declarations: [packet.BuffRemove](../../server/internal/packet/packet.go#L119), [proto.BuffRemove](../../internal/proto/opcodes.go#L103).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [removeEffectsInternal (server/internal/world/playereffects.go)](../../server/internal/world/playereffects.go#L205), [clearBuffsOnDeath (server/internal/world/skillattack.go)](../../server/internal/world/skillattack.go#L310)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L688)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>removeEffectsInternal — server/internal/world/playereffects.go</summary>

[Source definition](../../server/internal/world/playereffects.go#L205)

```go
func (s *Session) removeEffectsInternal(match func(*ActiveBuff) bool, why string, includeMapCurses bool) {
	before := s.speeds()
	kept := s.buffs[:0]
	var removed []ActiveBuff
	for i := range s.buffs {
		if (includeMapCurses || !s.buffs[i].MapCurse) && match(&s.buffs[i]) {
			removed = append(removed, s.buffs[i])
			continue
		}
		kept = append(kept, s.buffs[i])
	}
	s.buffs = kept
	if len(removed) == 0 {
		return
	}
	for _, b := range removed {
		s.removeElement(b)
		w := packet.New(packet.BuffRemove)
		w.U32(b.ID)
		s.send(w.MustFinish(), "0x050E applied skill removed "+why)
	}
	s.syncVisibility()
	s.sendBuffStats(why)
	if after := s.speeds(); after != before {
		s.broadcastSpeed(after)
	}
}
```

</details>

<details><summary>clearBuffsOnDeath — server/internal/world/skillattack.go</summary>

[Source definition](../../server/internal/world/skillattack.go#L310)

```go
func (s *Session) clearBuffsOnDeath() {
	s.deathLoss = s.deathExpLoss()
	duelDeath := s.duelDeathMark
	if duelDeath {
		// A lost duel (m_nVehicleStatus -1): no exp loss, no ultimate
		// death, and skills above 200 (not level 100) stay.
		s.deathLoss, s.duelDeathMark = -1, false
	}
	s.startRebirthTimer()
	kept := s.buffs[:0]
	var removed []uint32
	for _, b := range s.buffs {
		if b.MapCurse || (duelDeath && b.SkillID > 200 && b.SkillLevel != db.NPCSkillWireLevel) {
			kept = append(kept, b)
			continue
		}
		if d := db.MustDefs().EffectSkill(b.SkillID, b.SkillLevel); d != nil && d.KeepOnDeath {
			kept = append(kept, b)
			continue
		}
		removed = append(removed, b.ID)
	}
	s.buffs = kept
	s.syncVisibility()
	for _, id := range removed {
		w := packet.New(packet.BuffRemove)
		w.U32(id)
		s.send(w.MustFinish(), "0x050E buff removed by death")
	}
	if len(removed) > 0 {
		s.sendBuffStats("after death")
	}
}
```

</details>

<details><summary>DecodeU32 — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L190)

```go
func DecodeU32(p []byte) (uint32, error) { r := NewReader(p); v := r.U32(); return v, r.Err() }
```

</details>

## Opcode 050f

**`0x050F` — SkillKeep**

Declarations: [packet.SkillKeep](../../server/internal/packet/packet.go#L120), [proto.SkillKeep](../../internal/proto/opcodes.go#L104).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [applyFixedHit (server/internal/world/firethorn.go)](../../server/internal/world/firethorn.go#L99), [tickDebuffs (server/internal/world/playereffects.go)](../../server/internal/world/playereffects.go#L262), [processBuffs (server/internal/world/skillcast.go)](../../server/internal/world/skillcast.go#L392)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L892)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>applyFixedHit — server/internal/world/firethorn.go</summary>

[Source definition](../../server/internal/world/firethorn.go#L99)

```go
func (s *Session) applyFixedHit(h pendingUserHit) bool {
	ch := s.char
	killed := s.takeUserSkillDamage(h.AttackerID, h.Damage)
	shown := h.Damage
	if shown[0] > 0 { // Health Assistant moved part of it to SP / MP
		shown = [3]uint32{s.assisted[0], shown[1] + s.assisted[1], shown[2] + s.assisted[2]}
	}
	w := packet.New(packet.SkillKeep)
	w.U32(ch.ID)
	w.U16(h.Skill.SkillID)
	w.U8(h.Skill.SkillLevel)
	for _, v := range shown {
		w.U16(uint16(min(v, 65535)))
	}
	sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius, w.MustFinish(), "0x050F fire thorn damage")
	s.afterUserDamage(h.AttackerID, killed)
	return killed
}
```

</details>

<details><summary>tickDebuffs — server/internal/world/playereffects.go</summary>

[Source definition](../../server/internal/world/playereffects.go#L262)

```go
func (s *Session) tickDebuffs(now time.Time) bool {
	if s.invulnerable() {
		return false
	}
	ch := s.char
	effects := append([]ActiveBuff(nil), s.buffs...)
	for i := range effects {
		b := &effects[i]
		if b.Effect != typeEffectDebuff || (b.Detail != detailTimeDamage && b.Detail != detailDeathSentence) {
			continue
		}
		def := db.MustDefs().EffectSkill(b.SkillID, b.SkillLevel)
		if def == nil {
			continue
		}
		expiring := !b.MapCurse && !b.EndsAt.After(now)
		var damage [3]int32
		switch {
		case b.Detail == detailTimeDamage && (expiring || now.Sub(b.prevTick) >= effectTick):
			st := s.stats()
			damage = playerTimeDamage(def, &st, now.Sub(b.prevTick))
			b.prevTick = now
			for j := range s.buffs {
				if s.buffs[j].ID == b.ID {
					s.buffs[j].prevTick = now
					break
				}
			}
		case b.Detail == detailDeathSentence && expiring:
			damage[0] = int32(min(s.stats().CurrentHP, 65535))
		default:
			continue
		}
		if damage[0] > 0 || damage[1] > 0 || damage[2] > 0 {
			s.cancelTravelCast()
			s.breakStealth()
		}
		hp := uint32(max(damage[0], 0))
		sp, mp := uint32(max(damage[1], 0)), uint32(max(damage[2], 0))
		if sp > 0 || mp > 0 {
			s.spendDelta(mp, sp)
			if hp == 0 {
				s.persist()
			}
		}
		var killed bool
		if b.CasterUser != 0 {
			killed = s.takeUserDamage(b.CasterUser, hp)
		} else {
			killed = s.takeMobDamage(pendingDamage{MobGlobalID: b.CasterMob}, hp, nil)
		}
		if hp > 0 { // Health Assistant moved part of it to SP / MP
			hp = s.assisted[0]
			damage[1], damage[2] = damage[1]+int32(s.assisted[1]), damage[2]+int32(s.assisted[2])
		}
		w := packet.New(packet.SkillKeep)
		w.U32(ch.ID)
		w.U16(b.SkillID)
		w.U8(b.SkillLevel)
		w.U16(uint16(min(hp, 65535)))
		w.U16(uint16(min(max(damage[1], 0), 65535)))
		w.U16(uint16(min(max(damage[2], 0), 65535)))
		sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius, w.MustFinish(), "0x050F debuff damage")
		if killed {
			if b.CasterUser != 0 {
				s.afterUserDamage(b.CasterUser, true)
			} else {
				s.afterMobDamage(pendingDamage{MobGlobalID: b.CasterMob}, true)
			}
			s.dead, s.autoAttack, s.cast = true, false, nil
			s.clearBuffsOnDeath()
			return true
		}
		st := s.stats()
		s.send(buildCurrentHitpoints(&st), "0x0521 current hp after debuff damage")
	}
	return false
}
```

</details>

<details><summary>processBuffs — server/internal/world/skillcast.go</summary>

[Source definition](../../server/internal/world/skillcast.go#L392)

```go
func (s *Session) processBuffs() {
	if len(s.buffs) == 0 || s.char == nil || s.dead {
		return
	}
	now := time.Now()
	if s.tickDebuffs(now) {
		return
	}
	for i := range s.buffs {
		b := &s.buffs[i]
		expiring := !b.MapCurse && !b.EndsAt.After(now)
		hasHeal := b.HealHP+b.HealSP+b.HealMP > 0
		due := !b.nextTickAt.IsZero() && !b.nextTickAt.After(now)
		if hasHeal && (due || expiring) {
			hp, mp, sp := s.healDelta(b.HealHP, b.HealMP, b.HealSP)
			s.persist()
			w := packet.New(packet.SkillKeep)
			w.U32(s.char.ID)
			w.U16(b.SkillID)
			w.U8(b.SkillLevel)
			w.U16(uint16(min(hp, 65535)))
			w.U16(uint16(min(mp, 65535)))
			w.U16(uint16(min(sp, 65535)))
			s.send(w.MustFinish(), "0x050F buff heal tick")
			st := s.stats()
			s.send(buildCurrentHitpoints(&st), "0x0521 current hp/mp/sp after buff heal tick")
			if !expiring {
				b.nextTickAt = b.nextTickAt.Add(periodicHealTick)
			}
		}
	}
	s.removeEffects(func(b *ActiveBuff) bool {
		if b.EndsAt.After(now) {
			return false
		}
		s.log.Printf("GX Server World applied skill expired skill_id=%d level=%d", b.SkillID, b.SkillLevel)
		return true
	}, "after expiry")
}
```

</details>

<details><summary>DecodePeriodicEffect — internal/proto/periodic.go</summary>

[Source definition](../../internal/proto/periodic.go#L13)

```go
func DecodePeriodicEffect(p []byte) (v PeriodicEffect, err error) {
	r := NewReader(p)
	v.Target, v.Skill, v.Level = r.U32(), r.U16(), r.U8()
	v.HP, v.Resource1, v.Resource2 = r.U16(), r.U16(), r.U16()
	return v, r.Err()
}
```

```go
type PeriodicEffect struct {
	Target                   uint32
	Skill                    uint16
	Level                    uint8
	HP, Resource1, Resource2 uint16
}
```

</details>

## Opcode 0510

**`0x0510` — SkillCastStarted**

Declarations: [packet.SkillCastStarted](../../server/internal/packet/packet.go#L117), [proto.SkillCastStarted](../../internal/proto/opcodes.go#L101).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [handleUseSkill (server/internal/world/skillcast.go)](../../server/internal/world/skillcast.go#L158)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L802)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>handleUseSkill — server/internal/world/skillcast.go</summary>

[Source definition](../../server/internal/world/skillcast.go#L158)

```go
func (s *Session) handleUseSkill(op packet.Opcode, p []byte) {
	if len(p) < 7 || s.char == nil {
		return
	}
	s.cancelMove()
	ch := s.char
	number := p[2]
	if op == packet.SkillUseNoTarget && number == 0xff {
		id, _ := packet.U32(p[3:])
		s.cancelBuff(id)
		return
	}
	target, _ := packet.U32(p[3:])
	s.cancelVehicleCast()
	if s.ridingVehicle() || s.cast != nil || s.cannotAct() {
		s.log.Printf("GX Server World rejected skill use (mounted, casting, stunned or asleep) character_id=%d", ch.ID)
		return
	}
	k, def, ok := s.knownSkill(number)
	if !ok {
		s.log.Printf("GX Server World rejected skill use: no skill in slot number=%d", number)
		return
	}
	s.autoAttack = false
	if !s.skillActionReady(def, time.Now()) {
		if !s.queueSpinCombo(def, op, p) {
			s.sendSkillMessage(k.SkillID, 215)
		}
		return
	}
	if (def.TypeEffect == typeEffectAttack || def.TypeEffect == typeEffectDebuff) && !s.attackMovementAllowed(time.Now()) {
		return
	}
	target = s.friendlySkillRecipient(def, target)
	castTime := time.Duration(def.ReadyTime) * 250 * time.Millisecond
	if castTime == 0 {
		s.resolveSkill(target, number, op, nil)
		return
	}
	if !s.skillReady(k.SkillID) {
		s.sendSkillMessage(k.SkillID, 215)
		return
	}
	if !s.previousSkillCooling(def) {
		s.sendSkillMessage(k.SkillID, msgPreviousSkillRequired)
		return
	}
	stats := s.stats()
	if s.cannotAct() || stats.Disabled[min(int(def.Kind), 2)] > 0 {
		s.log.Printf("GX Server World rejected skill use: category disabled skill_id=%d", k.SkillID)
		return
	}
	if sp, mp := s.skillCosts(def, &stats); stats.CurrentMP < mp || stats.CurrentSP < sp {
		s.log.Printf("GX Server World rejected skill use: not enough MP/SP skill_id=%d", k.SkillID)
		return
	}
	castTarget := target
	if def.TypeEffect == typeEffectHeal || def.TypeEffect == typeEffectBuff {
		targets, _, ok := s.friendlyTargets(def, target)
		if !ok {
			s.sendFriendlySkillFailure(k, def, target)
			return
		}
		castTarget = ch.ID
		if def.TargetType == targetTypeOther || def.TargetType == targetTypeOtherAlt {
			castTarget = targets[0].CharacterID
		}
	}
	var pos *[2]float32
	if def.TypeEffect == typeEffectOnce {
		p, ok := s.userSkillTarget(def, target)
		if !ok {
			return
		}
		if !s.inUserSkillRange(p, def) {
			s.sendUserSkillRangeFailure(k, p)
			return
		}
		castTarget = p.CharacterID
	} else if op == opUserSkillPvP && (def.TypeEffect == typeEffectAttack || def.TypeEffect == typeEffectDebuff) {
		// A hostile skill on a user: ChkTargetUser before the cast.
		if def.TargetType != targetTypeEnemiesNearCaster {
			p := s.hostileUser(target)
			if p == nil {
				s.send(buildSkillUseResult(packet.SkillUseNoTarget, skillResultNotTarget, ch.ID, 0, k.SkillID, k.Level, 0, 0, 0),
					"0x0511 skill target refused")
				return
			}
			if !s.inUserSkillRange(p, def) {
				s.sendUserSkillRangeFailure(k, p)
				return
			}
			x, z := p.PositionXZ()
			castTarget, pos = p.CharacterID, &[2]float32{x, z}
		}
	} else if (def.TypeEffect == typeEffectAttack || def.TypeEffect == typeEffectDebuff) && castTarget != ch.ID && def.TargetType != targetTypeEnemiesNearCaster {
		if def.TypeEffect == typeEffectDebuff && def.TypeDetail == detailDegeneration {
			s.sendSkillMessage(k.SkillID, 317) // it cannot be cast upon a monster
			return
		}
		mob, ok := MobByID(ch.MapID, target)
		if !ok {
			return
		}
		if mobFriendly(db.MustDefs().Mobs[mob.MobID], s.faction) {
			s.sendSkillMessage(k.SkillID, 317)
			return
		}
		if !s.inSkillRange(&mob, def, &stats) {
			s.sendMobSkillRangeFailure(k, &mob)
			return
		}
		pos = &[2]float32{mob.PosX, mob.PosZ}
	}
	if def.TypeEffect == typeEffectAttack || def.TypeEffect == typeEffectDebuff {
		s.breakStealth()
	}
	s.send(packet.New(packet.AutoAttackStop).MustFinish(), "0x0212 auto attack stop (casting)")
	w := packet.New(packet.SkillCastStarted)
	w.U32(ch.ID)
	w.U32(castTarget)
	w.U16(k.SkillID)
	w.U8(k.Level)
	s.broadcastNear(w.MustFinish(), "0x0510 skill cast started", true)
	if pos == nil {
		pos = &[2]float32{}
	}
	now := time.Now()
	s.lastActionAt = now
	s.cast = &pendingCast{target, number, op, now.Add(castTime), pos}
	s.autoAttack = false
	s.syncPlayerSpeed(false)
	s.log.Printf("GX Server World skill cast started skill_id=%d level=%d cast_time=%s", k.SkillID, k.Level, castTime)
}
```

</details>

<details><summary>DecodeSkillCast — internal/proto/skills.go</summary>

[Source definition](../../internal/proto/skills.go#L111)

```go
func DecodeSkillCast(p []byte) (m SkillCast, err error) {
	r := NewReader(p)
	m.Source, m.Target, m.ID, m.Level = r.U32(), r.U32(), r.U16(), r.U8()
	return m, r.Err()
}
```

```go
type SkillCast struct {
	Source, Target uint32
	ID             uint16
	Level          uint8
}
```

</details>

## Opcode 0511

**`0x0511` — SkillUseNoTarget**

Declarations: [packet.SkillUseNoTarget](../../server/internal/packet/packet.go#L141), [proto.SkillUseNoTarget](../../internal/proto/opcodes.go#L126).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L23), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L255)

**S→C serializers:** [buildSkillUseResult (server/internal/world/skillcast.go)](../../server/internal/world/skillcast.go#L112)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L904)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildSkillUseResult — server/internal/world/skillcast.go</summary>

[Source definition](../../server/internal/world/skillcast.go#L112)

```go
func buildSkillUseResult(op packet.Opcode, result uint8, sender, target uint32, skillID uint16, level uint8, hp, sp, mp uint16) []byte {
	w := packet.New(op)
	w.U8(result)
	w.U32(sender)
	w.U32(target)
	w.U16(skillID)
	w.U8(level)
	w.U16(hp)
	w.U16(sp)
	w.U16(mp)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeSkillResult — internal/proto/skills.go</summary>

[Source definition](../../internal/proto/skills.go#L117)

```go
func DecodeSkillResult(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	if Op(p) == MobSkillUse || Op(p) == MobSkillDrain {
		m.AttackSlot = r.U8()
	}
	m.ID, m.Level = r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if Op(p) == SkillUserDrain || Op(p) == SkillMobDrain || Op(p) == MobSkillDrain {
		m.HealedHP, m.HealedSP, m.HealedMP = r.U16(), r.U16(), r.U16()
	}
	if err := r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill result: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type SkillResult struct {
	Result                       uint8
	AttackSlot                   uint8
	Source, Target               uint32
	ID                           uint16
	Level                        uint8
	HP, SP, MP                   uint16
	HealedHP, HealedSP, HealedMP uint16
}
```

</details>

## Opcode 0512

**`0x0512` — SkillUserDrain**

Declarations: [proto.SkillUserDrain](../../internal/proto/opcodes.go#L127).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L904)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>DecodeSkillResult — internal/proto/skills.go</summary>

[Source definition](../../internal/proto/skills.go#L117)

```go
func DecodeSkillResult(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	if Op(p) == MobSkillUse || Op(p) == MobSkillDrain {
		m.AttackSlot = r.U8()
	}
	m.ID, m.Level = r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if Op(p) == SkillUserDrain || Op(p) == SkillMobDrain || Op(p) == MobSkillDrain {
		m.HealedHP, m.HealedSP, m.HealedMP = r.U16(), r.U16(), r.U16()
	}
	if err := r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill result: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type SkillResult struct {
	Result                       uint8
	AttackSlot                   uint8
	Source, Target               uint32
	ID                           uint16
	Level                        uint8
	HP, SP, MP                   uint16
	HealedHP, HealedSP, HealedMP uint16
}
```

</details>

## Opcode 0513

**`0x0513` — SkillPartyMember**

Declarations: [packet.SkillPartyMember](../../server/internal/packet/packet.go#L142), [proto.SkillPartyMember](../../internal/proto/opcodes.go#L128).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildSkillUseResult (server/internal/world/skillcast.go)](../../server/internal/world/skillcast.go#L112)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L904)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildSkillUseResult — server/internal/world/skillcast.go</summary>

[Source definition](../../server/internal/world/skillcast.go#L112)

```go
func buildSkillUseResult(op packet.Opcode, result uint8, sender, target uint32, skillID uint16, level uint8, hp, sp, mp uint16) []byte {
	w := packet.New(op)
	w.U8(result)
	w.U32(sender)
	w.U32(target)
	w.U16(skillID)
	w.U8(level)
	w.U16(hp)
	w.U16(sp)
	w.U16(mp)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeSkillResult — internal/proto/skills.go</summary>

[Source definition](../../internal/proto/skills.go#L117)

```go
func DecodeSkillResult(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	if Op(p) == MobSkillUse || Op(p) == MobSkillDrain {
		m.AttackSlot = r.U8()
	}
	m.ID, m.Level = r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if Op(p) == SkillUserDrain || Op(p) == SkillMobDrain || Op(p) == MobSkillDrain {
		m.HealedHP, m.HealedSP, m.HealedMP = r.U16(), r.U16(), r.U16()
	}
	if err := r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill result: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type SkillResult struct {
	Result                       uint8
	AttackSlot                   uint8
	Source, Target               uint32
	ID                           uint16
	Level                        uint8
	HP, SP, MP                   uint16
	HealedHP, HealedSP, HealedMP uint16
}
```

</details>

## Opcode 0514

**`0x0514` — SkillUserExtraDamage**

Declarations: [proto.SkillUserExtraDamage](../../internal/proto/opcodes.go#L129).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildUserSkillExtra (server/internal/world/pvpskill.go)](../../server/internal/world/pvpskill.go#L297)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L910)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildUserSkillExtra — server/internal/world/pvpskill.go</summary>

[Source definition](../../server/internal/world/pvpskill.go#L297)

```go
func buildUserSkillExtra(caster, target uint32, skillID uint16, level uint8, add [3]uint32) []byte {
	w := packet.New(opUserSkillExtra)
	w.U32(caster)
	w.U32(target)
	w.U16(skillID)
	w.U8(level)
	for _, v := range add {
		w.U16(uint16(v))
	}
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeSkillExtra — internal/proto/skills.go</summary>

[Source definition](../../internal/proto/skills.go#L139)

```go
func DecodeSkillExtra(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Source, m.Target, m.ID, m.Level = r.U32(), r.U32(), r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if err = r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill extra: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type SkillResult struct {
	Result                       uint8
	AttackSlot                   uint8
	Source, Target               uint32
	ID                           uint16
	Level                        uint8
	HP, SP, MP                   uint16
	HealedHP, HealedSP, HealedMP uint16
}
```

</details>

## Opcode 0515

**`0x0515` — SkillUserReflectedDamage**

Declarations: [proto.SkillUserReflectedDamage](../../internal/proto/opcodes.go#L130).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [applyUserHit (server/internal/world/pvp.go)](../../server/internal/world/pvp.go#L348)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L916)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>applyUserHit — server/internal/world/pvp.go</summary>

[Source definition](../../server/internal/world/pvp.go#L348)

```go
func (s *Session) applyUserHit(h pendingUserHit) bool {
	if s.invulnerable() {
		return false
	}
	ch := s.char
	if ch == nil {
		return false
	}
	stats := s.stats()
	if stats.CurrentHP == 0 {
		return false
	}
	s.lastCombatAt = time.Now()
	if h.DrainHeal {
		s.applyDrainHeal(h)
		return false
	}
	if h.GemAttack {
		s.applyGemAttack(h.AttackerID)
		return false
	}
	if h.Trap && h.Skill != nil {
		return s.applyTrapHit(h)
	}
	if h.Fixed && h.Skill != nil {
		return s.applyFixedHit(h)
	}
	if h.Skill != nil {
		return s.applyUserSkillHit(h, stats)
	}
	attacker := FindPlayerByID(h.AttackerID)
	if h.Reflected {
		killed := s.takeUserSkillDamage(h.AttackerID, h.Damage)
		w := packet.New(0x0515)
		w.U32(h.AttackerID) // the mirror's owner
		w.U32(ch.ID)
		for _, v := range h.Damage {
			w.U16(uint16(min(v, 65535)))
		}
		sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius, w.MustFinish(), "0x0515 mirrored hit")
		s.afterUserDamage(h.AttackerID, killed)
		return killed
	}
	s.cancelMove() // StatusCastingStop on the target
	if s.blocksAttack(h.Kind) || !userHits(&h.Stats, &stats, h.Kind) {
		sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius,
			buildCharacterUserAttack(h.AttackerID, ch.ID, attackResultMiss, 0), "0x0502 user attack (miss)")
		return false
	}
	duelHitBy(s.player, h.AttackerID) // PvPChkLeaveByAttack
	closeRanged := DistanceXZ(h.X, h.Z, ch.PosX, ch.PosZ) <= closeRangeDistance
	mirrored := s.reflects(h.Kind, nil)
	value, crit := userDamage(&h.Stats, &stats, h.Kind, closeRanged)
	damage := uint32(max(value, 0))
	if stats.Absorption > 0 {
		damage = max(satSub(damage, stats.Absorption), 1)
	}
	code := uint8(attackSuccessNormal)
	if crit {
		code = attackResultCritical
	}
	killed, shown := false, damage
	if mirrored {
		shown = 0
	} else {
		killed = s.takeUserDamage(h.AttackerID, damage)
		shown = s.assisted[0]
	}
	sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius,
		buildCharacterUserAttack(h.AttackerID, ch.ID, code, uint16(min(shown, 65535))), "0x0502 user attack")
	if mirrored && attacker != nil {
		attacker.QueueUserHit(pendingUserHit{AttackerID: ch.ID, Reflected: true, Damage: [3]uint32{damage, 0, 0}})
	}
	s.log.Printf("GX Server World user attack attacker_id=%d target_id=%d kind=%d damage=%d crit=%t mirrored=%t killed=%t",
		h.AttackerID, ch.ID, h.Kind, damage, crit, mirrored, killed)
	s.afterUserDamage(h.AttackerID, killed)
	if !killed {
		s.itemOptionsAfterUserHit(h)
	}
	return killed
}
```

</details>

<details><summary>DecodeReflectedDamage — internal/proto/skills.go</summary>

[Source definition](../../internal/proto/skills.go#L158)

```go
func DecodeReflectedDamage(p []byte) (m ReflectedDamage, err error) {
	r := NewReader(p)
	m.Source, m.Target = r.U32(), r.U32()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if err = r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("reflected damage: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type ReflectedDamage struct {
	Source, Target uint32
	HP, SP, MP     uint16
}
```

</details>

## Opcode 0517

**`0x0517` — SkillUse**

Declarations: [packet.SkillUse](../../server/internal/packet/packet.go#L192), [proto.SkillUse](../../internal/proto/opcodes.go#L179).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L23), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L255)

**S→C serializers:** [buildSkillUseResult (server/internal/world/skillcast.go)](../../server/internal/world/skillcast.go#L112)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L904)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildSkillUseResult — server/internal/world/skillcast.go</summary>

[Source definition](../../server/internal/world/skillcast.go#L112)

```go
func buildSkillUseResult(op packet.Opcode, result uint8, sender, target uint32, skillID uint16, level uint8, hp, sp, mp uint16) []byte {
	w := packet.New(op)
	w.U8(result)
	w.U32(sender)
	w.U32(target)
	w.U16(skillID)
	w.U8(level)
	w.U16(hp)
	w.U16(sp)
	w.U16(mp)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeSkillResult — internal/proto/skills.go</summary>

[Source definition](../../internal/proto/skills.go#L117)

```go
func DecodeSkillResult(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	if Op(p) == MobSkillUse || Op(p) == MobSkillDrain {
		m.AttackSlot = r.U8()
	}
	m.ID, m.Level = r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if Op(p) == SkillUserDrain || Op(p) == SkillMobDrain || Op(p) == MobSkillDrain {
		m.HealedHP, m.HealedSP, m.HealedMP = r.U16(), r.U16(), r.U16()
	}
	if err := r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill result: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type SkillResult struct {
	Result                       uint8
	AttackSlot                   uint8
	Source, Target               uint32
	ID                           uint16
	Level                        uint8
	HP, SP, MP                   uint16
	HealedHP, HealedSP, HealedMP uint16
}
```

</details>

## Opcode 0518

**`0x0518` — SkillMobDrain**

Declarations: [proto.SkillMobDrain](../../internal/proto/opcodes.go#L131).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildDrainPacket (server/internal/world/skillattack.go)](../../server/internal/world/skillattack.go#L286)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L904)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildDrainPacket — server/internal/world/skillattack.go</summary>

[Source definition](../../server/internal/world/skillattack.go#L286)

```go
func buildDrainPacket(op packet.Opcode, result uint8, sender, target uint32, skillID uint16, level uint8, damage, healed [3]uint16) []byte {
	w := packet.New(op)
	w.U8(result)
	w.U32(sender)
	w.U32(target)
	w.U16(skillID)
	w.U8(level)
	for _, v := range damage {
		w.U16(v)
	}
	for _, v := range healed {
		w.U16(v)
	}
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeSkillResult — internal/proto/skills.go</summary>

[Source definition](../../internal/proto/skills.go#L117)

```go
func DecodeSkillResult(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	if Op(p) == MobSkillUse || Op(p) == MobSkillDrain {
		m.AttackSlot = r.U8()
	}
	m.ID, m.Level = r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if Op(p) == SkillUserDrain || Op(p) == SkillMobDrain || Op(p) == MobSkillDrain {
		m.HealedHP, m.HealedSP, m.HealedMP = r.U16(), r.U16(), r.U16()
	}
	if err := r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill result: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type SkillResult struct {
	Result                       uint8
	AttackSlot                   uint8
	Source, Target               uint32
	ID                           uint16
	Level                        uint8
	HP, SP, MP                   uint16
	HealedHP, HealedSP, HealedMP uint16
}
```

</details>

## Opcode 0519

**`0x0519` — SkillMobOther**

Declarations: [proto.SkillMobOther](../../internal/proto/opcodes.go#L132).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildSkillUseResult (server/internal/world/skillcast.go)](../../server/internal/world/skillcast.go#L112)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L904)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildSkillUseResult — server/internal/world/skillcast.go</summary>

[Source definition](../../server/internal/world/skillcast.go#L112)

```go
func buildSkillUseResult(op packet.Opcode, result uint8, sender, target uint32, skillID uint16, level uint8, hp, sp, mp uint16) []byte {
	w := packet.New(op)
	w.U8(result)
	w.U32(sender)
	w.U32(target)
	w.U16(skillID)
	w.U8(level)
	w.U16(hp)
	w.U16(sp)
	w.U16(mp)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeSkillResult — internal/proto/skills.go</summary>

[Source definition](../../internal/proto/skills.go#L117)

```go
func DecodeSkillResult(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	if Op(p) == MobSkillUse || Op(p) == MobSkillDrain {
		m.AttackSlot = r.U8()
	}
	m.ID, m.Level = r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if Op(p) == SkillUserDrain || Op(p) == SkillMobDrain || Op(p) == MobSkillDrain {
		m.HealedHP, m.HealedSP, m.HealedMP = r.U16(), r.U16(), r.U16()
	}
	if err := r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill result: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type SkillResult struct {
	Result                       uint8
	AttackSlot                   uint8
	Source, Target               uint32
	ID                           uint16
	Level                        uint8
	HP, SP, MP                   uint16
	HealedHP, HealedSP, HealedMP uint16
}
```

</details>

## Opcode 051a

**`0x051A` — SkillExtraDamage**

Declarations: [packet.SkillExtraDamage](../../server/internal/packet/packet.go#L193), [proto.SkillExtraDamage](../../internal/proto/opcodes.go#L180).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [skillHit (server/internal/world/skillattack.go)](../../server/internal/world/skillattack.go#L180)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L910)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>skillHit — server/internal/world/skillattack.go</summary>

[Source definition](../../server/internal/world/skillattack.go#L180)

```go
func (s *Session) skillHit(target uint32, def *db.SkillDef, k db.CharacterSkill, first bool) bool {
	ch := s.char
	mob, ok := MobByID(ch.MapID, target)
	if !ok || mob.CurrentHP == 0 {
		return false
	}
	md := db.MustDefs().Mobs[mob.MobID]
	if md == nil || mobFriendly(md, s.faction) {
		return false
	}
	s.breakStealth()
	stats := s.stats()
	mc, _ := MobCombatByID(ch.MapID, target)
	drain := first && def.TypeDetail == detailDrain
	op := opSkillOther
	switch {
	case drain:
		op = opSkillDrain
	case first:
		op = packet.SkillUse
	}
	lapisDebuff := s.itemAttacking && def.TypeEffect == typeEffectDebuff
	SetMobAggro(ch.MapID, target, ch.ID)
	s.lastCombatAt = time.Now()
	if !playerSkillHits(&stats, &mc, md, def) {
		recordMobAIMiss(ch.MapID, mob.ID, ch.ID)
		if lapisDebuff {
			return true
		}
		s.log.Printf("GX Server World skill miss skill_id=%d level=%d mob_global_id=%d mob_id=%d", k.SkillID, k.Level, mob.ID, mob.MobID)
		frame := buildSkillUseResult(op, attackResultMiss, ch.ID, mob.ID, k.SkillID, k.Level, 0, 0, 0)
		if drain {
			frame = buildSkillDrainResult(attackResultMiss, ch.ID, mob.ID, k.SkillID, k.Level, [3]uint16{}, [3]uint16{})
		}
		sendNear(ch.MapID, mob.PosX, mob.PosZ, CombatRadius, frame, "skill use (miss)")
		return true
	}
	t := skillTarget{MaxHP: mob.MaxHP, HP: mob.CurrentHP, MaxMP: md.MaxMP, MaxSP: md.MaxSP, MP: md.MaxMP, SP: md.MaxSP}
	caster := [3]uint32{stats.CurrentHP, stats.CurrentSP, stats.CurrentMP}
	close := DistanceXZ(ch.PosX, ch.PosZ, mob.PosX, mob.PosZ) <= closeRangeDistance
	damage, crit := playerSkillDamage(&stats, &mc, def, t, caster, close)
	r, ok := DamageMobFrom(ch.MapID, target, uint32(max(damage[0], 0)), ch.ID)
	if !ok {
		return false
	}
	AddMobAggro(ch.MapID, target, ch.ID, hitAggro(&stats, def.Aggro))
	code := uint8(attackSuccessNormal)
	if crit {
		code = attackResultCritical
	}
	dealt := [3]uint16{r.DamageHP, uint16(min(max(damage[1], 0), 65535)), uint16(min(max(damage[2], 0), 65535))}
	frame := buildSkillUseResult(op, code, ch.ID, r.Mob.ID, k.SkillID, k.Level, dealt[0], dealt[1], dealt[2])
	if drain {
		// The caster heals what it drained (CUser::Heal), HP/SP/MP.
		hp, mp, sp := s.healDelta(uint32(dealt[0]), uint32(dealt[2]), uint32(dealt[1]))
		frame = buildSkillDrainResult(code, ch.ID, r.Mob.ID, k.SkillID, k.Level, dealt,
			[3]uint16{uint16(min(hp, 65535)), uint16(min(sp, 65535)), uint16(min(mp, 65535))})
		s.persist()
	}
	if !lapisDebuff {
		sendNear(ch.MapID, r.Mob.PosX, r.Mob.PosZ, CombatRadius, frame, "skill use")
	}
	if drain {
		st := s.stats()
		s.send(buildCurrentHitpoints(&st), "0x0521 current hp after drain")
	}
	alive := s.resolveMobDamage(&r)
	if alive && !drain && (def.AddDamageHP != 0 || def.AddDamageSP != 0 || def.AddDamageMP != 0) {
		if x, ok := DamageMobFrom(ch.MapID, target, uint32(uint16(def.AddDamageHP)), ch.ID); ok {
			w := packet.New(packet.SkillExtraDamage)
			w.U32(ch.ID)
			w.U32(x.Mob.ID)
			w.U16(k.SkillID)
			w.U8(k.Level)
			w.U16(x.DamageHP)
			w.U16(uint16(def.AddDamageSP))
			w.U16(uint16(def.AddDamageMP))
			sendNear(ch.MapID, x.Mob.PosX, x.Mob.PosZ, CombatRadius, w.MustFinish(), "0x051A skill extra damage")
			alive = s.resolveMobDamage(&x)
		}
	}
	// TypeEffect 3 leaves the skill on the mob (CMob::AddApplySkill).
	if alive && def.TypeEffect == typeEffectDebuff {
		if speed, ok := ApplyMobEffect(ch.MapID, target, ch.ID, def); ok {
			if lapisDebuff {
				sendNear(ch.MapID, r.Mob.PosX, r.Mob.PosZ, CombatRadius, frame, "skill use")
			}
			if speed != nil {
				sendNear(ch.MapID, r.Mob.PosX, r.Mob.PosZ, BroadcastRadius, speed, "0x0609 mob speed")
			}
		}
	}
	if alive {
		s.gemCastsOnMob(target, weaponGemSkills(s.items)) // AttackItemOption(CMob); a gem cast skips it
	}
	return alive
}
```

</details>

<details><summary>DecodeSkillExtra — internal/proto/skills.go</summary>

[Source definition](../../internal/proto/skills.go#L139)

```go
func DecodeSkillExtra(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Source, m.Target, m.ID, m.Level = r.U32(), r.U32(), r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if err = r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill extra: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type SkillResult struct {
	Result                       uint8
	AttackSlot                   uint8
	Source, Target               uint32
	ID                           uint16
	Level                        uint8
	HP, SP, MP                   uint16
	HealedHP, HealedSP, HealedMP uint16
}
```

</details>

## Opcode 051b

**`0x051B` — SkillMobReflectedDamage**

Declarations: [proto.SkillMobReflectedDamage](../../internal/proto/opcodes.go#L181).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [reflectToMob (server/internal/world/playereffects.go)](../../server/internal/world/playereffects.go#L454)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L916)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>reflectToMob — server/internal/world/playereffects.go</summary>

[Source definition](../../server/internal/world/playereffects.go#L454)

```go
func (s *Session) reflectToMob(mobGlobalID uint32, damage [3]uint16) {
	ch := s.char
	r, ok := DamageMobFrom(ch.MapID, mobGlobalID, uint32(damage[0]), ch.ID)
	if !ok {
		return
	}
	w := packet.New(0x051B)
	w.U32(ch.ID)
	w.U32(mobGlobalID)
	for _, v := range damage {
		w.U16(v)
	}
	sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius, w.MustFinish(), "0x051B reflected hit")
	s.resolveMobDamage(&r)
}
```

</details>

<details><summary>DecodeReflectedDamage — internal/proto/skills.go</summary>

[Source definition](../../internal/proto/skills.go#L158)

```go
func DecodeReflectedDamage(p []byte) (m ReflectedDamage, err error) {
	r := NewReader(p)
	m.Source, m.Target = r.U32(), r.U32()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if err = r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("reflected damage: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
```

```go
type ReflectedDamage struct {
	Source, Target uint32
	HP, SP, MP     uint16
}
```

</details>

## Opcode 051c

**`0x051C` — CharacterDeathState**

Declarations: [packet.CharacterDeathState](../../server/internal/packet/packet.go#L195), [proto.CharacterDeathState](../../internal/proto/opcodes.go#L183).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildCharacterDeathState (server/internal/world/entities.go)](../../server/internal/world/entities.go#L1440), [buildCharacterSpeed (server/internal/world/playereffects.go)](../../server/internal/world/playereffects.go#L157)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L737)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildCharacterDeathState — server/internal/world/entities.go</summary>

[Source definition](../../server/internal/world/entities.go#L1440)

```go
func buildCharacterDeathState(characterID uint32) []byte {
	w := packet.New(packet.CharacterDeathState)
	w.U32(characterID)
	w.U8(0x07)
	w.U8(0x02)
	return w.MustFinish()
}
```

</details>

<details><summary>buildCharacterSpeed — server/internal/world/playereffects.go</summary>

[Source definition](../../server/internal/world/playereffects.go#L157)

```go
func buildCharacterSpeed(id uint32, sp playerSpeeds) []byte {
	w := packet.New(packet.CharacterDeathState)
	w.U32(id)
	w.U8(sp.attack)
	w.U8(sp.move)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeCharacterSpeed — internal/proto/speed.go</summary>

[Source definition](../../internal/proto/speed.go#L10)

```go
func DecodeCharacterSpeed(p []byte) (m CharacterSpeed, err error) {
	r := NewReader(p)
	m.ID, m.Attack, m.Move = r.U32(), r.U8(), r.U8()
	return m, r.Err()
}
```

```go
type CharacterSpeed struct {
	ID           uint32
	Attack, Move uint8
}
```

</details>

## Opcode 051d

**`0x051D` — CharacterMountShape**

Declarations: [packet.CharacterMountShape](../../server/internal/packet/packet.go#L194), [proto.CharacterMountShape](../../internal/proto/opcodes.go#L182).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [enterFrames (server/internal/world/view.go)](../../server/internal/world/view.go#L318)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L456)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>enterFrames — server/internal/world/view.go</summary>

[Source definition](../../server/internal/world/view.go#L318)

```go
func (p *ConnectedPlayer) enterFrames() [][]byte {
	frames := [][]byte{p.EnteredMapPacket(), p.ShapePacket(), p.MovePacket(), p.visibilityPacket(), buildVehicleState(p.vehicleState())}
	if mob := p.transformMob.Load(); mob != 0 {
		frames = append(frames, buildTransform(p.CharacterID, uint16(mob)))
	}
	p.mu.Lock()
	sp, known := p.speeds, p.speedKnown
	ladderIndex, ladderMode := p.ladderIndex, p.ladderMode
	p.mu.Unlock()
	if ladderMode != 0 {
		frames = append(frames, buildLadderState(p.CharacterID, ladderIndex, ladderMode))
	}
	if known {
		frames = append(frames, buildCharacterSpeed(p.CharacterID, sp))
	}
	if shape := p.mountShape(); shape != 0 {
		w := packet.New(packet.CharacterMountShape)
		w.U32(p.CharacterID)
		w.U8(shape)
		frames = append(frames, w.MustFinish())
	}
	if p.IsSitting() {
		frames = append(frames, p.SitMotionPacket())
	}
	return frames
}
```

</details>

<details><summary>DecodeMountShape — internal/proto/vehicle.go</summary>

[Source definition](../../internal/proto/vehicle.go#L24)

```go
func DecodeMountShape(p []byte) (MountShape, error) {
	r := NewReader(p)
	v := MountShape{ID: r.U32(), Shape: r.U8()}
	return v, r.Err()
}
```

```go
type MountShape struct {
	ID    uint32
	Shape uint8
}
```

</details>

## Opcode 051e

**`0x051E` — LevelUpOther**

Declarations: [packet.LevelUpOther](../../server/internal/packet/packet.go#L74), [proto.LevelUpOther](../../internal/proto/opcodes.go#L58).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildLevelUp (server/internal/world/combat.go)](../../server/internal/world/combat.go#L610)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L938)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildLevelUp — server/internal/world/combat.go</summary>

[Source definition](../../server/internal/world/combat.go#L610)

```go
func buildLevelUp(op packet.Opcode, ch *db.Character) []byte {
	prev, next := expBounds(ch)
	w := packet.New(op)
	w.U32(ch.ID)
	w.U16(ch.Level)
	w.U16(ch.StatPoint)
	w.U16(ch.SkillPoint)
	w.U32(prev)
	w.U32(next)
	return w.MustFinish()
}
```

</details>

<details><summary>NewReader — internal/proto/rw.go</summary>

[Source definition](../../internal/proto/rw.go#L28)

```go
func NewReader(payload []byte) *Reader {
	r := &Reader{b: payload, off: 2}
	if len(payload) >= 2 {
		r.wide = itemstack.Wide(binary.LittleEndian.Uint16(payload))
	}
	if len(payload) < 2 {
		r.err = ErrShort
	}
	return r
}
```

```go
type Reader struct {
	wide bool
	b    []byte
	off  int
	err  error
}
```

</details>

## Opcode 051f

**`0x051F` — CharacterMaxHpMpSp**

Declarations: [packet.CharacterMaxHpMpSp](../../server/internal/packet/packet.go#L121), [proto.CharacterMaxHpMpSp](../../internal/proto/opcodes.go#L105).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildMaxHPMPSP (server/internal/world/character.go)](../../server/internal/world/character.go#L475)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L963)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildMaxHPMPSP — server/internal/world/character.go</summary>

[Source definition](../../server/internal/world/character.go#L475)

```go
func buildMaxHPMPSP(ch *db.Character, s *CharacterStats) []byte {
	w := packet.New(packet.CharacterMaxHpMpSp)
	w.U32(charID(ch))
	w.U32(s.MaxHP)
	w.U32(s.MaxMP)
	w.U32(s.MaxSP)
	return w.MustFinish()
}
```

</details>

<details><summary>NewReader — internal/proto/rw.go</summary>

[Source definition](../../internal/proto/rw.go#L28)

```go
func NewReader(payload []byte) *Reader {
	r := &Reader{b: payload, off: 2}
	if len(payload) >= 2 {
		r.wide = itemstack.Wide(binary.LittleEndian.Uint16(payload))
	}
	if len(payload) < 2 {
		r.err = ErrShort
	}
	return r
}
```

```go
type Reader struct {
	wide bool
	b    []byte
	off  int
	err  error
}
```

</details>

## Opcode 0520

**`0x0520` — PartyBadge**

Declarations: [packet.PartyBadge](../../server/internal/packet/packet.go#L51), [proto.PartyBadge](../../internal/proto/opcodes.go#L35).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [partyBadge (server/internal/world/party.go)](../../server/internal/world/party.go#L123)

**Client receive reducers:** [applyParty (internal/game/party.go)](../../internal/game/party.go#L145), [Apply (internal/game/state.go)](../../internal/game/state.go#L660)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>partyBadge — server/internal/world/party.go</summary>

[Source definition](../../server/internal/world/party.go#L123)

```go
func partyBadge(p *ConnectedPlayer) {
	kind := uint8(0)
	if pt := partyOf(p); pt != nil {
		kind = 1
		if pt.leaderPlayer() == p {
			kind = 2
		}
	}
	w := packet.New(packet.PartyBadge)
	w.U32(p.CharacterID)
	w.U8(kind)
	frame := w.MustFinish()
	x, z := p.PositionXZ()
	sendNear(p.MapID(), x, z, BroadcastRadius, frame, "0x0520 party badge")
}
```

</details>

<details><summary>NewReader — internal/proto/rw.go</summary>

[Source definition](../../internal/proto/rw.go#L28)

```go
func NewReader(payload []byte) *Reader {
	r := &Reader{b: payload, off: 2}
	if len(payload) >= 2 {
		r.wide = itemstack.Wide(binary.LittleEndian.Uint16(payload))
	}
	if len(payload) < 2 {
		r.err = ErrShort
	}
	return r
}
```

```go
type Reader struct {
	wide bool
	b    []byte
	off  int
	err  error
}
```

</details>

## Opcode 0521

**`0x0521` — CharacterCurrentHitpoints**

Declarations: [packet.CharacterCurrentHitpoints](../../server/internal/packet/packet.go#L122), [proto.CharacterCurrentHitpoints](../../internal/proto/opcodes.go#L106).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildCurrentHitpoints (server/internal/world/character.go)](../../server/internal/world/character.go#L467)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1099)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildCurrentHitpoints — server/internal/world/character.go</summary>

[Source definition](../../server/internal/world/character.go#L467)

```go
func buildCurrentHitpoints(s *CharacterStats) []byte {
	w := packet.New(packet.CharacterCurrentHitpoints)
	w.U32(s.CurrentHP)
	w.U32(s.CurrentMP)
	w.U32(s.CurrentSP)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodePools — internal/proto/combat.go</summary>

[Source definition](../../internal/proto/combat.go#L6)

```go
func DecodePools(p []byte) (m Pools, err error) {
	r := NewReader(p)
	m.HP, m.MP, m.SP = r.U32(), r.U32(), r.U32()
	return m, r.Err()
}
```

```go
type Pools struct{ HP, MP, SP uint32 }
```

</details>

## Opcode 0522

**`0x0522` — CharacterPvPKills**

Declarations: [proto.CharacterPvPKills](../../internal/proto/opcodes.go#L107).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [gmSet (server/internal/world/gm.go)](../../server/internal/world/gm.go#L448), [killPCToCount (server/internal/world/pvpdeath.go)](../../server/internal/world/pvpdeath.go#L153)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L390)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>gmSet — server/internal/world/gm.go</summary>

[Source definition](../../server/internal/world/gm.go#L448)

```go
func (s *Session) gmSet(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("%s", gmSetUsage)
	}
	field := strings.ToLower(args[0])
	var cap uint64
	switch field {
	case "money":
		cap = db.MoneyCap
	case "kill":
		cap = pvpKillCap
	case "str", "dex", "rec", "int", "wis", "luc":
		cap = db.BaseStatCap
	default:
		return fmt.Errorf("%s", gmSetUsage)
	}
	// Saturate while parsing so even an arbitrarily large decimal cannot overflow.
	var value uint64
	if args[1] == "" {
		return fmt.Errorf("amount must be a non-negative integer")
	}
	for _, digit := range args[1] {
		if digit < '0' || digit > '9' {
			return fmt.Errorf("amount must be a non-negative integer")
		}
		value = min(value*10+uint64(digit-'0'), cap)
	}
	next := *s.char
	switch field {
	case "money":
		next.Money = uint32(value)
	case "kill":
		next.Kills = uint32(value)
	case "str":
		next.Str = uint16(value)
	case "dex":
		next.Dex = uint16(value)
	case "rec":
		next.Rec = uint16(value)
	case "int":
		next.Int = uint16(value)
	case "wis":
		next.Wis = uint16(value)
	case "luc":
		next.Luc = uint16(value)
	}
	stat := field != "money" && field != "kill"
	if stat {
		normalize(&next)
	}
	if err := db.SaveCharacter(&next); err != nil {
		return err
	}
	*s.char = next
	s.syncPlayer()
	if field == "money" {
		s.send(buildSetMoney(next.Money), "GM set money")
	} else if field == "kill" {
		s.send(buildKillCountUpdate(0, next.Kills), "GM set kills")
		w := packet.New(0x0522)
		w.U32(next.ID)
		w.U32(next.Kills)
		frame := w.MustFinish()
		s.send(frame, "GM set rank")
		s.sendToViewers(frame, "GM set rank (view)")
	} else {
		s.send(buildStatsReset(next.StatPoint, next.Str, next.Rec, next.Int, next.Wis, next.Dex, next.Luc), "GM set base stats")
		// 0214 resets the client's displayed claim stage; restore the actual
		// unchanged stages immediately without reopening any reward.
		s.send(buildPvPRewardStatus(&next), "GM set unchanged PvP rewards")
		s.sendStatPackets()
		s.partyPools()
	}
	s.gmReply(fmt.Sprintf("GM: %s set to %d.", field, value))
	return nil
}
```

</details>

<details><summary>killPCToCount — server/internal/world/pvpdeath.go</summary>

[Source definition](../../server/internal/world/pvpdeath.go#L153)

```go
func (s *Session) killPCToCount(count uint32) {
	if count == 0 {
		return
	}
	ch := s.char
	for range min(count, uint32(255*len(s.quests))) {
		s.questKillPC()
	}
	if ch.Kills >= pvpKillCap {
		return
	}
	added := min(count, pvpKillCap-ch.Kills)
	ch.Kills += added
	s.persist()
	addKillCountBatch(s.faction, added, s.log) // CWorld::AddKillCount for the killer's country
	s.send(buildKillCountUpdate(0, ch.Kills), "0x020E kills")
	w := packet.New(0x0522)
	w.U32(ch.ID)
	w.U32(ch.Kills)
	frame := w.MustFinish()
	s.send(frame, "0x0522 kill count")
	s.sendToViewers(frame, "0x0522 kill count (view)")
}
```

</details>

<details><summary>DecodeVisiblePvPKills — internal/proto/pvp_count.go</summary>

[Source definition](../../internal/proto/pvp_count.go#L27)

```go
func DecodeVisiblePvPKills(p []byte) (VisiblePvPKills, error) {
	r := NewReader(p)
	v := VisiblePvPKills{ID: r.U32(), Kills: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("visible pvp kills: %d trailing bytes", r.Remaining())
	}
	return v, nil
}
```

```go
type VisiblePvPKills struct{ ID, Kills uint32 }
```

</details>

## Opcode 0526

**`0x0526` — CharacterAdditionalStats**

Declarations: [packet.CharacterAdditionalStats](../../server/internal/packet/packet.go#L123), [proto.CharacterAdditionalStats](../../internal/proto/opcodes.go#L108).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildAdditionalStats (server/internal/world/character.go)](../../server/internal/world/character.go#L449)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L716)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildAdditionalStats — server/internal/world/character.go</summary>

[Source definition](../../server/internal/world/character.go#L449)

```go
func buildAdditionalStats(s *CharacterStats) []byte {
	w := packet.New(packet.CharacterAdditionalStats)
	for _, v := range []uint32{s.Str, s.Rec, s.Int, s.Wis, s.Dex, s.Luc, s.MinAttack, s.MaxAttack,
		s.MinMagic, s.MaxMagic, s.Defense, s.Resistance} {
		w.U32(v)
	}
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeAdditionalStats — internal/proto/stats.go</summary>

[Source definition](../../internal/proto/stats.go#L39)

```go
func DecodeAdditionalStats(p []byte) (s AdditionalStats, err error) {
	r := NewReader(p)
	s.Str, s.Rec, s.Int, s.Wis, s.Dex, s.Luc = r.U32(), r.U32(), r.U32(), r.U32(), r.U32(), r.U32()
	s.MinAttack, s.MaxAttack, s.MinMagic, s.MaxMagic, s.Defense, s.Resistance = r.U32(), r.U32(), r.U32(), r.U32(), r.U32(), r.U32()
	return s, r.Err()
}
```

```go
type AdditionalStats struct {
	Str, Rec, Int, Wis, Dex, Luc             uint32
	MinAttack, MaxAttack, MinMagic, MaxMagic uint32
	Defense, Resistance                      uint32
}
```

</details>

## Opcode 0551

**`0x0551` — CharacterResurrect**

Declarations: [packet.CharacterResurrect](../../server/internal/packet/packet.go#L191), [proto.CharacterResurrect](../../internal/proto/opcodes.go#L178).

Source notes: charId:u32, 5-byte blob 04 46 AF 37 01 (verbatim), x,y,z:f32.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildCharacterResurrect (server/internal/world/entities.go)](../../server/internal/world/entities.go#L1456)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1068)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildCharacterResurrect — server/internal/world/entities.go</summary>

[Source definition](../../server/internal/world/entities.go#L1456)

```go
func buildCharacterResurrect(characterID uint32, kind uint8, exp uint32, x, y, z float32) []byte {
	w := packet.New(packet.CharacterResurrect)
	w.U32(characterID)
	w.U8(kind)
	w.U32(exp)
	w.F32(x)
	w.F32(y)
	w.F32(z)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeResurrection — internal/proto/combat.go</summary>

[Source definition](../../internal/proto/combat.go#L48)

```go
func DecodeResurrection(p []byte) (m Resurrection, err error) {
	r := NewReader(p)
	m.ID, m.Kind, m.Exp = r.U32(), r.U8(), r.U32()
	m.X, m.Y, m.Z = r.F32(), r.F32(), r.F32()
	return m, r.Err()
}
```

```go
type Resurrection struct {
	ID      uint32
	Kind    uint8
	Exp     uint32
	X, Y, Z float32
}
```

</details>

## Opcode 0553

**`0x0553` — ResurrectRequest**

Declarations: [packet.ResurrectRequest](../../server/internal/packet/packet.go#L143), [proto.ResurrectRequest](../../internal/proto/opcodes.go#L133).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L220)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>ReviveChoicePacket — internal/proto/combat.go</summary>

[Source definition](../../internal/proto/combat.go#L70)

```go
func ReviveChoicePacket(mode ReviveMode) []byte {
	w := NewWriter(ResurrectRequest)
	w.U8(uint8(mode))
	return w.Bytes()
}
```

```go
type ReviveMode uint8
```

</details>

## Opcode 0554

**`0x0554` — ItemRepair**

Declarations: [packet.ItemRepair](../../server/internal/packet/packet.go#L144), [proto.ItemRepair](../../internal/proto/opcodes.go#L134).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L222)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L532)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>ItemRepairPacket — internal/proto/repair.go</summary>

[Source definition](../../internal/proto/repair.go#L22)

```go
func ItemRepairPacket(bag, slot uint8) []byte {
	w := NewWriter(ItemRepair)
	w.Raw([]byte{bag, slot})
	return w.Bytes()
}
```

</details>

<details><summary>DecodeItemRepair — internal/proto/repair.go</summary>

[Source definition](../../internal/proto/repair.go#L10)

```go
func DecodeItemRepair(p []byte) (ItemRepairResult, error) {
	r := NewReader(p)
	v := ItemRepairResult{Bag: r.U8(), Slot: r.U8(), Money: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if !(v.Bag == 255 && v.Slot == 255) && (v.Bag > 5 || v.Bag == 0 && v.Slot >= 13 || v.Bag != 0 && v.Slot >= 24) {
		return v, fmt.Errorf("invalid repair position")
	}
	return v, nil
}
```

```go
type ItemRepairResult struct {
	Bag, Slot uint8
	Money     uint32
}
```

</details>

## Opcode 0556

**`0x0556` — ReturnToTown**

Declarations: [packet.ReturnToTown](../../server/internal/packet/packet.go#L145), [proto.ReturnToTown](../../internal/proto/opcodes.go#L135).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L23), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L224)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

The shape is decoded inline, delegated, or only declared. Use the linked adapter/reducer and the codec appendix below; there is no inferred fixed layout here.

## Opcode 0558

**`0x0558` — SkillDetectInfo**

Declarations: [packet.SkillDetectInfo](../../server/internal/packet/packet.go#L189), [proto.SkillDetectInfo](../../internal/proto/opcodes.go#L177).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildDetectInfo (server/internal/world/skillonce.go)](../../server/internal/world/skillonce.go#L75)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L898)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildDetectInfo — server/internal/world/skillonce.go</summary>

[Source definition](../../server/internal/world/skillonce.go#L75)

```go
func buildDetectInfo(attr uint32, level uint16, mode uint8) []byte {
	w := packet.New(packet.SkillDetectInfo)
	w.U32(attr)
	w.U16(level)
	w.U8(mode)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeDetectInfo — internal/proto/detect.go</summary>

[Source definition](../../internal/proto/detect.go#L10)

```go
func DecodeDetectInfo(p []byte) (v DetectInfo, err error) {
	r := NewReader(p)
	v.DefenseElement, v.Level, v.Mode = r.U32(), r.U16(), r.U8()
	return v, r.Err()
}
```

```go
type DetectInfo struct {
	DefenseElement uint32
	Level          uint16
	Mode           uint8
}
```

</details>
