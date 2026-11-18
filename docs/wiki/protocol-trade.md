# Protocol: Player exchange

> Generated from Go source by `tools/generate_server_reference.py`; do not edit by hand.

[All opcodes](server-opcodes.md) · [Wire conventions](server-protocol.md) · [Server architecture](server-architecture.md)

Client codecs operate on opcode + body; server packet.Writer emits length + opcode + body. Excerpts preserve the respective construction API. Go wire methods specify widths, not host struct alignment. `Count` is special: see the stack-width compatibility rules. Loops, conditions, helper calls and fixed string sizes are significant; do not flatten them into a single fixed-size struct.

## Opcode 0a01

**`0x0A01` — ExchangeRequest**

Declarations: [packet.ExchangeRequest](../../server/internal/packet/packet.go#L212), [proto.ExchangeRequest](../../internal/proto/opcodes.go#L200).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L162)

**S→C serializers:** [handleTradeRequest (server/internal/world/trade.go)](../../server/internal/world/trade.go#L157)

**Client receive reducers:** [applyExchange (internal/game/exchange.go)](../../internal/game/exchange.go#L37), [Apply (internal/game/state.go)](../../internal/game/state.go#L569)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>handleTradeRequest — server/internal/world/trade.go</summary>

[Source definition](../../server/internal/world/trade.go#L157)

```go
func (s *Session) handleTradeRequest(target uint32) {
	me := s.player
	if me == nil || target == me.CharacterID || tradeFor(me.CharacterID) != nil {
		return
	}
	// CUser::PacketExchange 0x0A01: a user of the same zone and country
	// that is not trading.
	other := FindPlayerByID(target)
	if other == nil || !me.canSee(other) || other.MapID() != me.MapID() || other.Faction() != me.Faction() || tradeFor(target) != nil {
		return
	}
	tradeMu.Lock()
	reciprocal := tradeRequests[target] == me.CharacterID
	if reciprocal {
		delete(tradeRequests, target)
		delete(tradeRequests, me.CharacterID)
	} else {
		tradeRequests[me.CharacterID] = target
	}
	tradeMu.Unlock()
	if reciprocal {
		startTrade(me, other)
		return
	}
	w := packet.New(packet.ExchangeRequest)
	w.U32(me.CharacterID)
	other.Send(w.MustFinish(), "0x0A01 exchange request")
}
```

</details>

<details><summary>ExchangeRequestPacket — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L64)

```go
func ExchangeRequestPacket(id uint32) []byte {
	w := NewWriter(ExchangeRequest)
	w.U32(id)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeExchangeID — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L24)

```go
func DecodeExchangeID(p []byte) (uint32, error) { r := NewReader(p); id := r.U32(); return id, r.Err() }
```

</details>

## Opcode 0a02

**`0x0A02` — ExchangeAnswer**

Declarations: [packet.ExchangeAnswer](../../server/internal/packet/packet.go#L213), [proto.ExchangeAnswer](../../internal/proto/opcodes.go#L201).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L167)

**S→C serializers:** [tradePacket (server/internal/world/trade.go)](../../server/internal/world/trade.go#L76)

**Client receive reducers:** [applyExchange (internal/game/exchange.go)](../../internal/game/exchange.go#L44), [Apply (internal/game/state.go)](../../internal/game/state.go#L569)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>tradePacket — server/internal/world/trade.go</summary>

[Source definition](../../server/internal/world/trade.go#L76)

```go
func tradePacket(op packet.Opcode, b ...byte) []byte {
	w := packet.New(op)
	w.Bytes(b)
	return w.MustFinish()
}
```

</details>

<details><summary>ExchangeAnswerPacket — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L69)

```go
func ExchangeAnswerPacket(accept bool) []byte {
	w := NewWriter(ExchangeAnswer)
	if accept {
		w.U8(0)
	} else {
		w.U8(1)
	}
	return w.Bytes()
}
```

</details>

<details><summary>DecodeExchangeByte — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L31)

```go
func DecodeExchangeByte(p []byte) (uint8, error) { r := NewReader(p); v := r.U8(); return v, r.Err() }
```

</details>

## Opcode 0a03

**`0x0A03` — ExchangeStart**

Declarations: [packet.ExchangeStart](../../server/internal/packet/packet.go#L214), [proto.ExchangeStart](../../internal/proto/opcodes.go#L202).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildTradeStart (server/internal/world/trade.go)](../../server/internal/world/trade.go#L136)

**Client receive reducers:** [applyExchange (internal/game/exchange.go)](../../internal/game/exchange.go#L51), [Apply (internal/game/state.go)](../../internal/game/state.go#L569)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildTradeStart — server/internal/world/trade.go</summary>

[Source definition](../../server/internal/world/trade.go#L136)

```go
func buildTradeStart(partner uint32) []byte {
	w := packet.New(packet.ExchangeStart)
	w.U32(partner)
	w.U8(0)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeExchangeStart — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L25)

```go
func DecodeExchangeStart(p []byte) (id uint32, result uint8, err error) {
	r := NewReader(p)
	id = r.U32()
	result = r.U8()
	return id, result, r.Err()
}
```

</details>

## Opcode 0a04

**`0x0A04` — ExchangeEnd**

Declarations: [packet.ExchangeEnd](../../server/internal/packet/packet.go#L215), [proto.ExchangeEnd](../../internal/proto/opcodes.go#L203).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [tradePacket (server/internal/world/trade.go)](../../server/internal/world/trade.go#L76)

**Client receive reducers:** [applyExchange (internal/game/exchange.go)](../../internal/game/exchange.go#L61), [Apply (internal/game/state.go)](../../internal/game/state.go#L569)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>tradePacket — server/internal/world/trade.go</summary>

[Source definition](../../server/internal/world/trade.go#L76)

```go
func tradePacket(op packet.Opcode, b ...byte) []byte {
	w := packet.New(op)
	w.Bytes(b)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeExchangeByte — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L31)

```go
func DecodeExchangeByte(p []byte) (uint8, error) { r := NewReader(p); v := r.U8(); return v, r.Err() }
```

</details>

## Opcode 0a05

**`0x0A05` — ExchangeReady**

Declarations: [packet.ExchangeReady](../../server/internal/packet/packet.go#L216), [proto.ExchangeReady](../../internal/proto/opcodes.go#L204).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L171)

**S→C serializers:** [tradePacket (server/internal/world/trade.go)](../../server/internal/world/trade.go#L76)

**Client receive reducers:** [applyExchange (internal/game/exchange.go)](../../internal/game/exchange.go#L74), [Apply (internal/game/state.go)](../../internal/game/state.go#L569)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>tradePacket — server/internal/world/trade.go</summary>

[Source definition](../../server/internal/world/trade.go#L76)

```go
func tradePacket(op packet.Opcode, b ...byte) []byte {
	w := packet.New(op)
	w.Bytes(b)
	return w.MustFinish()
}
```

</details>

<details><summary>ExchangeReadyPacket — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L78)

```go
func ExchangeReadyPacket(value uint8) []byte {
	w := NewWriter(ExchangeReady)
	w.U8(value)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeExchangeUpdate — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L32)

```go
func DecodeExchangeUpdate(p []byte) (u ExchangeUpdate, err error) {
	r := NewReader(p)
	u = ExchangeUpdate{r.U8(), r.U8()}
	return u, r.Err()
}
```

```go
type ExchangeUpdate struct{ Kind, Value uint8 }
```

</details>

<details><summary>Op — internal/proto/rw.go</summary>

[Source definition](../../internal/proto/rw.go#L161)

```go
func Op(payload []byte) Opcode {
	if len(payload) < 2 {
		return 0
	}
	return Opcode(itemstack.Base(binary.LittleEndian.Uint16(payload)))
}
```

</details>

## Opcode 0a06

**`0x0A06` — ExchangeAdd**

Declarations: [packet.ExchangeAdd](../../server/internal/packet/packet.go#L217), [proto.ExchangeAdd](../../internal/proto/opcodes.go#L205).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L175)

**S→C serializers:** [handleTradeAdd (server/internal/world/trade.go)](../../server/internal/world/trade.go#L261)

**Client receive reducers:** [applyExchange (internal/game/exchange.go)](../../internal/game/exchange.go#L110), [Apply (internal/game/state.go)](../../internal/game/state.go#L569)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>handleTradeAdd — server/internal/world/trade.go</summary>

[Source definition](../../server/internal/world/trade.go#L261)

```go
func (s *Session) handleTradeAdd(bag, slot uint8, count uint16, dest uint8) {
	if dest >= tradeSlots || bag == 0 || !isValidItemPosition(bag, slot) {
		return
	}
	i := findItem(s.items, bag, slot)
	if i < 0 {
		return
	}
	// 0x0A06: count 1..stack, not a quest item (real type 27), not offered
	// twice.
	it := s.items[i]
	if count == 0 || count > it.Count || realType(it.Type) == 27 {
		return
	}
	if t := tradeFor(s.char.ID); t != nil {
		if off := t.offer(s.char.ID); off != nil {
			for _, x := range off.items {
				if x != nil && x.Item.UID == it.UID {
					return
				}
			}
		}
	}
	n := count
	o := &offeredItem{Item: it, Count: n}
	_, partner, ok := s.mutateOffer(func(off *tradeOffer) { off.items[dest] = o })
	if !ok {
		return
	}
	w := packet.New(packet.ExchangeAdd)
	w.U8(bag)
	w.U8(slot)
	w.Count(n)
	w.U8(dest)
	s.send(w.MustFinish(), "0x0A06 exchange add (self)")
	if partner != nil {
		partner.Send(buildTradeAddDest(dest, o), "0x0A09 exchange add dest")
	}
}
```

</details>

<details><summary>ExchangeAddPacket — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L83)

```go
func ExchangeAddPacket(bag, slot uint8, count uint16, dest uint8) []byte {
	w := NewWriter(ExchangeAdd)
	w.Raw([]byte{bag, slot})
	w.Count(count)
	w.U8(dest)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeExchangeOwnItem — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L42)

```go
func DecodeExchangeOwnItem(p []byte) (item ExchangeOwnItem, err error) {
	r := NewReader(p)
	item = ExchangeOwnItem{Bag: r.U8(), Slot: r.U8(), Count: r.Count(), Destination: r.U8()}
	if err = r.Err(); err == nil && item.Destination >= ExchangeSlots {
		err = fmt.Errorf("exchange: invalid slot %d", item.Destination)
	}
	return
}
```

```go
type ExchangeOwnItem struct {
	Bag, Slot, Destination uint8
	Count                  uint16
}
```

</details>

## Opcode 0a07

**`0x0A07` — ExchangeRemove**

Declarations: [packet.ExchangeRemove](../../server/internal/packet/packet.go#L218), [proto.ExchangeRemove](../../internal/proto/opcodes.go#L206).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L183)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>ExchangeRemovePacket — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L90)

```go
func ExchangeRemovePacket(slot uint8) []byte {
	w := NewWriter(ExchangeRemove)
	w.U8(slot)
	return w.Bytes()
}
```

</details>

## Opcode 0a08

**`0x0A08` — ExchangeAddMoney**

Declarations: [packet.ExchangeAddMoney](../../server/internal/packet/packet.go#L219), [proto.ExchangeAddMoney](../../internal/proto/opcodes.go#L207).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L187)

**S→C serializers:** [buildTradeMoney (server/internal/world/trade.go)](../../server/internal/world/trade.go#L315)

**Client receive reducers:** [applyExchange (internal/game/exchange.go)](../../internal/game/exchange.go#L97), [Apply (internal/game/state.go)](../../internal/game/state.go#L569)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildTradeMoney — server/internal/world/trade.go</summary>

[Source definition](../../server/internal/world/trade.go#L315)

```go
func buildTradeMoney(kind uint8, money uint32) []byte {
	w := packet.New(packet.ExchangeAddMoney)
	w.U8(kind)
	w.U32(money)
	return w.MustFinish()
}
```

</details>

<details><summary>ExchangeMoneyPacket — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L95)

```go
func ExchangeMoneyPacket(money uint32) []byte {
	w := NewWriter(ExchangeAddMoney)
	w.U32(money)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeExchangeMoney — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L37)

```go
func DecodeExchangeMoney(p []byte) (m ExchangeMoneyUpdate, err error) {
	r := NewReader(p)
	m = ExchangeMoneyUpdate{r.U8(), r.U32()}
	return m, r.Err()
}
```

```go
type ExchangeMoneyUpdate struct {
	Kind  uint8
	Money uint32
}
```

</details>

## Opcode 0a09

**`0x0A09` — ExchangeAddDest**

Declarations: [packet.ExchangeAddDest](../../server/internal/packet/packet.go#L220), [proto.ExchangeAddDest](../../internal/proto/opcodes.go#L208).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildTradeAddDest (server/internal/world/trade.go)](../../server/internal/world/trade.go#L232)

**Client receive reducers:** [applyExchange (internal/game/exchange.go)](../../internal/game/exchange.go#L120), [Apply (internal/game/state.go)](../../internal/game/state.go#L569)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildTradeAddDest — server/internal/world/trade.go</summary>

[Source definition](../../server/internal/world/trade.go#L232)

```go
func buildTradeAddDest(slot uint8, o *offeredItem) []byte {
	w := packet.New(packet.ExchangeAddDest)
	w.U8(slot)
	writeOffered(w, o)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeExchangePartnerItem — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L50)

```go
func DecodeExchangePartnerItem(p []byte) (slot uint8, item ExchangeItem, err error) {
	r := NewReader(p)
	slot = r.U8()
	item.Type, item.TypeID, item.Count = r.U8(), r.U8(), r.Count()
	item.Quality = r.U16()
	for i := range item.Gems {
		item.Gems[i] = r.U8()
	}
	item.Craft = r.Fixed(21)
	if err = r.Err(); err == nil && slot >= ExchangeSlots {
		err = fmt.Errorf("exchange: invalid slot %d", slot)
	}
	return
}
```

```go
type ExchangeItem struct {
	Type, TypeID uint8
	Count        uint16
	Quality      uint16
	Gems         [6]uint8
	Craft        string
}
```

</details>

## Opcode 0a0a

**`0x0A0A` — ExchangeConfirm**

Declarations: [packet.ExchangeConfirm](../../server/internal/packet/packet.go#L221), [proto.ExchangeConfirm](../../internal/proto/opcodes.go#L209).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L192)

**S→C serializers:** [tradePacket (server/internal/world/trade.go)](../../server/internal/world/trade.go#L76)

**Client receive reducers:** [applyExchange (internal/game/exchange.go)](../../internal/game/exchange.go#L74), [Apply (internal/game/state.go)](../../internal/game/state.go#L569)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>tradePacket — server/internal/world/trade.go</summary>

[Source definition](../../server/internal/world/trade.go#L76)

```go
func tradePacket(op packet.Opcode, b ...byte) []byte {
	w := packet.New(op)
	w.Bytes(b)
	return w.MustFinish()
}
```

</details>

<details><summary>ExchangeConfirmPacket — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L100)

```go
func ExchangeConfirmPacket(confirm bool) []byte {
	w := NewWriter(ExchangeConfirm)
	if confirm {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

</details>

<details><summary>DecodeExchangeUpdate — internal/proto/exchange.go</summary>

[Source definition](../../internal/proto/exchange.go#L32)

```go
func DecodeExchangeUpdate(p []byte) (u ExchangeUpdate, err error) {
	r := NewReader(p)
	u = ExchangeUpdate{r.U8(), r.U8()}
	return u, r.Err()
}
```

```go
type ExchangeUpdate struct{ Kind, Value uint8 }
```

</details>

<details><summary>Op — internal/proto/rw.go</summary>

[Source definition](../../internal/proto/rw.go#L161)

```go
func Op(payload []byte) Opcode {
	if len(payload) < 2 {
		return 0
	}
	return Opcode(itemstack.Base(binary.LittleEndian.Uint16(payload)))
}
```

</details>
