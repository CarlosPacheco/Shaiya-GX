# Protocol: Chat and legacy administration

> Generated from Go source by `tools/generate_server_reference.py`; do not edit by hand.

[All opcodes](server-opcodes.md) · [Wire conventions](server-protocol.md) · [Server architecture](server-architecture.md)

Client codecs operate on opcode + body; server packet.Writer emits length + opcode + body. Excerpts preserve the respective construction API. Go wire methods specify widths, not host struct alignment. `Count` is special: see the stack-width compatibility rules. Loops, conditions, helper calls and fixed string sizes are significant; do not flatten them into a single fixed-size struct.

## Opcode 1101

**`0x1101` — ChatNormal**

Declarations: [packet.ChatNormal](../../server/internal/packet/packet.go#L133), [proto.ChatNormal](../../internal/proto/opcodes.go#L118).

**C→S receive adapters:** [handlePublicChat (server/internal/world/chat.go)](../../server/internal/world/chat.go#L71), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L70)

**S→C serializers:** [buildChatByID (server/internal/world/protocol.go)](../../server/internal/world/protocol.go#L144)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1389)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildChatByID — server/internal/world/protocol.go</summary>

[Source definition](../../server/internal/world/protocol.go#L144)

```go
func buildChatByID(op packet.Opcode, senderID uint32, message string) []byte {
	w := packet.New(op)
	w.U32(senderID)
	writeChatText(w, message)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeChatByID — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L354)

```go
func DecodeChatByID(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.SenderID, m.Text = r.U32(), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
	Items     ChatItems
}
```

</details>

## Opcode 1102

**`0x1102` — ChatWhisper**

Declarations: [packet.ChatWhisper](../../server/internal/packet/packet.go#L179), [proto.ChatWhisper](../../internal/proto/opcodes.go#L167).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L73)

**S→C serializers:** [buildWhisper (server/internal/world/chat.go)](../../server/internal/world/chat.go#L136)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1403)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildWhisper — server/internal/world/chat.go</summary>

[Source definition](../../server/internal/world/chat.go#L136)

```go
func buildWhisper(direction uint8, name, message string) []byte {
	w := packet.New(packet.ChatWhisper)
	w.U8(direction)
	w.FixedString(name, 21)
	writeChatText(w, message)
	return w.MustFinish()
}
```

</details>

<details><summary>WhisperPacket — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L115)

```go
func WhisperPacket(target, text string) []byte {
	if target == "" || text == "" || len(EncodeText(target)) > 20 {
		return nil
	}
	w := NewWriter(ChatWhisper)
	w.Fixed(target, 21, true)
	w.Text(text)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeWhisper — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L368)

```go
func DecodeWhisper(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.Direction, m.Sender, m.Text = r.U8(), r.Fixed(21), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
	Items     ChatItems
}
```

</details>

## Opcode 1103

**`0x1103` — ChatWorld**

Declarations: [packet.ChatWorld](../../server/internal/packet/packet.go#L134), [proto.ChatWorld](../../internal/proto/opcodes.go#L119).

**C→S receive adapters:** [handlePublicChat (server/internal/world/chat.go)](../../server/internal/world/chat.go#L74), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L70)

**S→C serializers:** [buildChatByName (server/internal/world/protocol.go)](../../server/internal/world/protocol.go#L155)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1396)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildChatByName — server/internal/world/protocol.go</summary>

[Source definition](../../server/internal/world/protocol.go#L155)

```go
func buildChatByName(op packet.Opcode, sender, message string) []byte {
	w := packet.New(op)
	name := sender
	if len(name) > 21 {
		name = textwire.Prefix(name, 21)
	}
	w.FixedString(name, 21)
	writeChatText(w, message)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeChatByName — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L361)

```go
func DecodeChatByName(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.Sender, m.Text = r.Fixed(21), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
	Items     ChatItems
}
```

</details>

## Opcode 1104

**`0x1104` — ChatGuild**

Declarations: [packet.ChatGuild](../../server/internal/packet/packet.go#L178), [proto.ChatGuild](../../internal/proto/opcodes.go#L166).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L148)

**S→C serializers:** [guildChat (server/internal/world/guild.go)](../../server/internal/world/guild.go#L452)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1396)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>guildChat — server/internal/world/guild.go</summary>

[Source definition](../../server/internal/world/guild.go#L452)

```go
func (s *Session) guildChat(message string, items proto.ChatItems) {
	guildMu.Lock()
	defer guildMu.Unlock()
	g := s.player.guild.guild
	if g == nil {
		return
	}
	w := packet.New(packet.ChatGuild)
	w.FixedString(s.char.Name, 21)
	writeChatText(w, message)
	g.send(chatWithItems(w.MustFinish(), items), "0x1104 guild chat", nil)
}
```

</details>

<details><summary>DecodeChatByName — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L361)

```go
func DecodeChatByName(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.Sender, m.Text = r.Fixed(21), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
	Items     ChatItems
}
```

</details>

## Opcode 1105

**`0x1105` — ChatParty**

Declarations: [packet.ChatParty](../../server/internal/packet/packet.go#L70), [proto.ChatParty](../../internal/proto/opcodes.go#L54).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L140)

**S→C serializers:** [buildChatByID (server/internal/world/protocol.go)](../../server/internal/world/protocol.go#L144)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1389)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildChatByID — server/internal/world/protocol.go</summary>

[Source definition](../../server/internal/world/protocol.go#L144)

```go
func buildChatByID(op packet.Opcode, senderID uint32, message string) []byte {
	w := packet.New(op)
	w.U32(senderID)
	writeChatText(w, message)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeChatByID — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L354)

```go
func DecodeChatByID(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.SenderID, m.Text = r.U32(), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
	Items     ChatItems
}
```

</details>

## Opcode 1106

**`0x1106` — ChatError**

Declarations: [packet.ChatError](../../server/internal/packet/packet.go#L135), [proto.ChatError](../../internal/proto/opcodes.go#L120).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildChatError (server/internal/world/protocol.go)](../../server/internal/world/protocol.go#L166)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1409)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildChatError — server/internal/world/protocol.go</summary>

[Source definition](../../server/internal/world/protocol.go#L166)

```go
func buildChatError(result uint8) []byte {
	w := packet.New(packet.ChatError)
	w.U8(result)
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

## Opcode 1107

**`0x1107` — ChatShout**

Declarations: [packet.ChatShout](../../server/internal/packet/packet.go#L180), [proto.ChatShout](../../internal/proto/opcodes.go#L168).

**C→S receive adapters:** [handlePublicChat (server/internal/world/chat.go)](../../server/internal/world/chat.go#L77), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L70)

**S→C serializers:** [buildChatByID (server/internal/world/protocol.go)](../../server/internal/world/protocol.go#L144)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1389)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildChatByID — server/internal/world/protocol.go</summary>

[Source definition](../../server/internal/world/protocol.go#L144)

```go
func buildChatByID(op packet.Opcode, senderID uint32, message string) []byte {
	w := packet.New(op)
	w.U32(senderID)
	writeChatText(w, message)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeChatByID — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L354)

```go
func DecodeChatByID(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.SenderID, m.Text = r.U32(), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
	Items     ChatItems
}
```

</details>

## Opcode 1108

**`0x1108` — ChatWorldShout**

Declarations: [packet.ChatWorldShout](../../server/internal/packet/packet.go#L181), [proto.ChatWorldShout](../../internal/proto/opcodes.go#L169).

**C→S receive adapters:** [handlePublicChat (server/internal/world/chat.go)](../../server/internal/world/chat.go#L84), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L70)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1396)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>DecodeChatByName — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L361)

```go
func DecodeChatByName(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.Sender, m.Text = r.Fixed(21), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
	Items     ChatItems
}
```

</details>

## Opcode 1109

**`0x1109` — MobSpeechText**

Declarations: [proto.MobSpeechText](../../internal/proto/mob_speech.go#L6).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildMobAIMessage (server/internal/world/mob_ai_speech.go)](../../server/internal/world/mob_ai_speech.go#L49)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1379)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildMobAIMessage — server/internal/world/mob_ai_speech.go</summary>

[Source definition](../../server/internal/world/mob_ai_speech.go#L49)

```go
func buildMobAIMessage(owner uint32, index uint16, text string, indexed bool) []byte {
	if indexed {
		w := packet.New(0x110a)
		w.U32(owner)
		w.U16(index)
		return w.MustFinish()
	}
	w := packet.New(0x1109)
	w.U8(1)
	w.U32(owner)
	b := encodeText(text)
	for len(b) > 127 {
		text = string([]rune(text)[:len([]rune(text))-1])
		b = encodeText(text)
	}
	w.U8(uint8(len(b)))
	w.Bytes(b)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeMobSpeech — internal/proto/mob_speech.go</summary>

[Source definition](../../internal/proto/mob_speech.go#L16)

```go
func DecodeMobSpeech(p []byte) (MobSpeech, error) {
	r := NewReader(p)
	var speech MobSpeech
	switch Op(p) {
	case MobSpeechText:
		if r.U8() != 1 {
			return speech, fmt.Errorf("invalid mob speech type")
		}
		speech.MobID = r.U32()
		speech.Text = r.Text()
	case MobSpeechIndex:
		speech.MobID = r.U32()
		speech.MessageID = r.U16()
	default:
		return speech, fmt.Errorf("invalid mob speech opcode")
	}
	if e := r.Err(); e != nil {
		return speech, e
	}
	if r.Remaining() != 0 {
		return speech, fmt.Errorf("trailing mob speech bytes")
	}
	return speech, nil
}
```

```go
type MobSpeech struct {
	MobID     uint32
	MessageID uint16
	Text      string
}
```

</details>

## Opcode 110a

**`0x110A` — MobSpeechIndex**

Declarations: [proto.MobSpeechIndex](../../internal/proto/mob_speech.go#L7).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildMobAIMessage (server/internal/world/mob_ai_speech.go)](../../server/internal/world/mob_ai_speech.go#L49)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1379)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildMobAIMessage — server/internal/world/mob_ai_speech.go</summary>

[Source definition](../../server/internal/world/mob_ai_speech.go#L49)

```go
func buildMobAIMessage(owner uint32, index uint16, text string, indexed bool) []byte {
	if indexed {
		w := packet.New(0x110a)
		w.U32(owner)
		w.U16(index)
		return w.MustFinish()
	}
	w := packet.New(0x1109)
	w.U8(1)
	w.U32(owner)
	b := encodeText(text)
	for len(b) > 127 {
		text = string([]rune(text)[:len([]rune(text))-1])
		b = encodeText(text)
	}
	w.U8(uint8(len(b)))
	w.Bytes(b)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeMobSpeech — internal/proto/mob_speech.go</summary>

[Source definition](../../internal/proto/mob_speech.go#L16)

```go
func DecodeMobSpeech(p []byte) (MobSpeech, error) {
	r := NewReader(p)
	var speech MobSpeech
	switch Op(p) {
	case MobSpeechText:
		if r.U8() != 1 {
			return speech, fmt.Errorf("invalid mob speech type")
		}
		speech.MobID = r.U32()
		speech.Text = r.Text()
	case MobSpeechIndex:
		speech.MobID = r.U32()
		speech.MessageID = r.U16()
	default:
		return speech, fmt.Errorf("invalid mob speech opcode")
	}
	if e := r.Err(); e != nil {
		return speech, e
	}
	if r.Remaining() != 0 {
		return speech, fmt.Errorf("trailing mob speech bytes")
	}
	return speech, nil
}
```

```go
type MobSpeech struct {
	MobID     uint32
	MessageID uint16
	Text      string
}
```

</details>

## Opcode 1111

**`0x1111` — ChatMap**

Declarations: [packet.ChatMap](../../server/internal/packet/packet.go#L136), [proto.ChatMap](../../internal/proto/opcodes.go#L121).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L70)

**S→C serializers:** [buildChatByName (server/internal/world/protocol.go)](../../server/internal/world/protocol.go#L155)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1396)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildChatByName — server/internal/world/protocol.go</summary>

[Source definition](../../server/internal/world/protocol.go#L155)

```go
func buildChatByName(op packet.Opcode, sender, message string) []byte {
	w := packet.New(op)
	name := sender
	if len(name) > 21 {
		name = textwire.Prefix(name, 21)
	}
	w.FixedString(name, 21)
	writeChatText(w, message)
	return w.MustFinish()
}
```

</details>

<details><summary>DecodeChatByName — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L361)

```go
func DecodeChatByName(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.Sender, m.Text = r.Fixed(21), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
	Items     ChatItems
}
```

</details>

## Opcode f101

**`0xF101` — ChatNormalAdmin**

Declarations: [packet.ChatNormalAdmin](../../server/internal/packet/packet.go#L202), [proto.ChatNormalAdmin](../../internal/proto/opcodes.go#L190).

**C→S receive adapters:** [handlePublicChat (server/internal/world/chat.go)](../../server/internal/world/chat.go#L71), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L70)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1389)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>DecodeChatByID — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L354)

```go
func DecodeChatByID(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.SenderID, m.Text = r.U32(), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
	Items     ChatItems
}
```

</details>

## Opcode f103

**`0xF103` — ChatWorldAdmin**

Declarations: [packet.ChatWorldAdmin](../../server/internal/packet/packet.go#L203), [proto.ChatWorldAdmin](../../internal/proto/opcodes.go#L191).

**C→S receive adapters:** [handlePublicChat (server/internal/world/chat.go)](../../server/internal/world/chat.go#L74), [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L70)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1396)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>DecodeChatByName — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L361)

```go
func DecodeChatByName(p []byte) (m Chat, err error) {
	r := NewReader(p)
	m.Sender, m.Text = r.Fixed(21), r.Text()
	m.Items = readChatItems(r)
	return m, r.Err()
}
```

```go
type Chat struct {
	SenderID  uint32
	Sender    string
	Text      string
	Direction uint8
	Items     ChatItems
}
```

</details>

## Opcode f501

**`0xF501` — GmCommandError**

Declarations: [packet.GmCommandError](../../server/internal/packet/packet.go#L204), [proto.GmCommandError](../../internal/proto/opcodes.go#L192).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

The shape is decoded inline, delegated, or only declared. Use the linked adapter/reducer and the codec appendix below; there is no inferred fixed layout here.

## Opcode f702

**`0xF702` — GmCommandGetItem**

Declarations: [packet.GmCommandGetItem](../../server/internal/packet/packet.go#L205), [proto.GmCommandGetItem](../../internal/proto/opcodes.go#L193).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

The shape is decoded inline, delegated, or only declared. Use the linked adapter/reducer and the codec appendix below; there is no inferred fixed layout here.
