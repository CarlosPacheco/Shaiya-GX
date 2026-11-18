# Protocol: GX protocol extensions

> Generated from Go source by `tools/generate_server_reference.py`; do not edit by hand.

[All opcodes](server-opcodes.md) · [Wire conventions](server-protocol.md) · [Server architecture](server-architecture.md)

Client codecs operate on opcode + body; server packet.Writer emits length + opcode + body. Excerpts preserve the respective construction API. Go wire methods specify widths, not host struct alignment. `Count` is special: see the stack-width compatibility rules. Loops, conditions, helper calls and fixed string sizes are significant; do not flatten them into a single fixed-size struct.

## Opcode 7f01

**`0x7F01` — GXGMState**

Declarations: [proto.GXGMState](../../internal/proto/gm.go#L11).

Source notes: GXGMState is a GX-only server response: self ID:u32, GM:u8, turbo:u8, message byte length:u16, UTF-8 message. Map teleport requests use GXGMMapTeleport; other requests use chat commands.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [gmReply (server/internal/world/gm.go)](../../server/internal/world/gm.go#L18)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1224)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>gmReply — server/internal/world/gm.go</summary>

[Source definition](../../server/internal/world/gm.go#L18)

```go
func (s *Session) gmReply(message string) {
	if s.char == nil {
		return
	}
	w := packet.New(gxGMState)
	w.U32(s.char.ID)
	role, turbo := uint8(0), uint8(0)
	if s.isAdmin {
		role = 1
		if s.gmTurbo {
			turbo = 1
		}
	}
	w.U8(role)
	w.U8(turbo)
	w.U16(uint16(len(message)))
	w.Bytes([]byte(message))
	s.send(w.MustFinish(), "GX GM state")
}
```

</details>

<details><summary>DecodeGMState — internal/proto/gm.go</summary>

[Source definition](../../internal/proto/gm.go#L22)

```go
func DecodeGMState(p []byte) (GMState, error) {
	if len(p) < 10 {
		return GMState{}, fmt.Errorf("short GX GM state")
	}
	n := int(binary.LittleEndian.Uint16(p[8:10]))
	if len(p) != 10+n || p[6] > 1 || p[7] > 1 || (p[6] == 0 && p[7] != 0) || !utf8.Valid(p[10:]) {
		return GMState{}, fmt.Errorf("invalid GX GM state")
	}
	return GMState{ID: binary.LittleEndian.Uint32(p[2:6]), GM: p[6] == 1, Turbo: p[7] == 1, Message: string(p[10:])}, nil
}
```

```go
type GMState struct {
	ID        uint32
	GM, Turbo bool
	Message   string
}
```

</details>

## Opcode 7f02

**`0x7F02` — GXVisibility**

Declarations: [proto.GXVisibility](../../internal/proto/gm.go#L14).

Source notes: GXVisibility: character ID:u32, concealed:u8. Sent only to authorized viewers.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [visibilityPacket (server/internal/world/visibility.go)](../../server/internal/world/visibility.go#L25)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1214)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>visibilityPacket — server/internal/world/visibility.go</summary>

[Source definition](../../server/internal/world/visibility.go#L25)

```go
func (p *ConnectedPlayer) visibilityPacket() []byte {
	w := packet.New(gxVisibility)
	w.U32(p.CharacterID)
	hidden := uint8(0)
	if p.hidden.Load() {
		hidden = 1
	}
	w.U8(hidden)
	return w.MustFinish()
}
```

</details>

## Opcode 7f03

**`0x7F03` — GXForgeQuote**

Declarations: [packet.GXForgeQuote](../../server/internal/packet/packet.go#L325), [proto.GXForgeQuote](../../internal/proto/forge_quote.go#L6).

Source notes: GXForgeQuote is a read-only preview; execution still validates live inventory.

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L275)

**S→C serializers:** [handleForgeQuote (server/internal/world/forge_quote.go)](../../server/internal/world/forge_quote.go#L103)

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>handleForgeQuote — server/internal/world/forge_quote.go</summary>

[Source definition](../../server/internal/world/forge_quote.go#L103)

```go
func (s *Session) handleForgeQuote(p []byte) {
	q, err := proto.DecodeForgeQuoteRequest(p)
	if err != nil {
		return
	}
	out := s.forgeQuote(q)
	w := packet.New(packet.GXForgeQuote)
	w.U32(out.Serial)
	w.U8(out.Kind)
	w.Bool(out.Valid)
	w.U32(out.Rate)
	w.U32(out.Gold)
	w.U8(out.Lapis)
	w.U32(out.Material)
	s.send(w.MustFinish(), "GX forge quote")
}
```

</details>

<details><summary>ForgeQuotePacket — internal/proto/forge_quote.go</summary>

[Source definition](../../internal/proto/forge_quote.go#L28)

```go
func ForgeQuotePacket(q ForgeQuoteRequest) []byte {
	w := NewWriter(GXForgeQuote)
	w.U32(q.Serial)
	w.U8(q.Kind)
	for _, k := range [][2]uint8{q.Item, q.Lapis, q.Material, q.Hammer} {
		w.Raw(k[:])
	}
	w.U8(q.Count)
	return w.Bytes()
}
```

```go
type ForgeQuoteRequest struct {
	Serial                        uint32
	Kind                          uint8
	Item, Lapis, Material, Hammer [2]uint8
	Count                         uint8
}
```

</details>

## Opcode 7f04

**`0x7F04` — GXQuestNPCs**

Declarations: [proto.GXQuestNPCs](../../internal/proto/quest_map.go#L9).

Source notes: GXQuestNPCs supplies map metadata, not visible/interactive world entities.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [questNPCFrames (server/internal/world/quest_map.go)](../../server/internal/world/quest_map.go#L50)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L246)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>questNPCFrames — server/internal/world/quest_map.go</summary>

[Source definition](../../server/internal/world/quest_map.go#L50)

```go
func questNPCFrames(mapID uint16, rows []proto.QuestNPC) [][]byte {
	// length/opcode 4 + map/flags/count 5 + 11 bytes per NPC.
	const batch = proto.QuestNPCChunkLimit
	var frames [][]byte
	for start := 0; ; {
		end := min(start+batch, len(rows))
		flags := uint8(0)
		if start == 0 {
			flags |= proto.QuestNPCBegin
		}
		if end == len(rows) {
			flags |= proto.QuestNPCEnd
		}
		w := packet.New(packet.Opcode(proto.GXQuestNPCs))
		w.U16(mapID)
		w.U8(flags)
		w.U16(uint16(end - start))
		for _, n := range rows[start:end] {
			w.U8(n.Type)
			w.U16(n.TypeID)
			w.F32(n.X)
			w.F32(n.Z)
		}
		frames = append(frames, w.MustFinish())
		if end == len(rows) {
			return frames
		}
		start = end
	}
}
```

</details>

## Opcode 7f05

**`0x7F05` — GXMobEnter**

Declarations: [proto.GXMobEnter](../../internal/proto/world.go#L250).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1341)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>DecodeMobEnter — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L265)

```go
func DecodeMobEnter(p []byte) (m MobSpawn, err error) {
	r := NewReader(p)
	m.ID, m.New, m.MobID, m.X = r.U32(), r.Bool(), r.U16(), r.F32()
	m.HeightKnown = len(p) >= 2 && Opcode(binary.LittleEndian.Uint16(p[:2])) == GXMobEnter
	if m.HeightKnown {
		m.Y = r.F32()
	}
	m.Z = r.F32()
	if m.HeightKnown && r.Remaining() > 0 {
		if r.Remaining() != 2 {
			return m, fmt.Errorf("invalid GX mob metadata length")
		}
		m.Country = r.U8()
		height := r.U8()
		if m.Country > 2 || height > 1 {
			return m, fmt.Errorf("invalid GX mob metadata")
		}
		m.CountryKnown = true
		m.HeightKnown = height == 1
	}
	return m, r.Err()
}
```

```go
type MobSpawn struct {
	Country      uint8
	CountryKnown bool
	ID           uint32
	New          bool
	MobID        uint16
	X, Y, Z      float32
	HeightKnown  bool
}
```

</details>

## Opcode 7f06

**`0x7F06` — GXMobMove**

Declarations: [proto.GXMobMove](../../internal/proto/world.go#L251).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1347)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>DecodeMobMove — internal/proto/world.go</summary>

[Source definition](../../internal/proto/world.go#L297)

```go
func DecodeMobMove(p []byte) (m MobMotion, err error) {
	r := NewReader(p)
	m.ID, m.Motion, m.X = r.U32(), r.U8(), r.F32()
	m.HeightKnown = len(p) >= 2 && Opcode(binary.LittleEndian.Uint16(p[:2])) == GXMobMove
	if m.HeightKnown {
		m.Y = r.F32()
	}
	m.Z = r.F32()
	if m.HeightKnown {
		m.DurationMS = r.U32()
	}
	return m, r.Err()
}
```

```go
type MobMotion struct {
	ID          uint32
	Motion      uint8
	X, Y, Z     float32
	HeightKnown bool
	DurationMS  uint32
}
```

</details>

## Opcode 7f07

**`0x7F07` — GXPhysicsClock**

Declarations: [proto.GXPhysicsClock](../../internal/proto/world.go#L248).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendPhysicsClock (server/internal/world/player_movement.go)](../../server/internal/world/player_movement.go#L371)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1204)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendPhysicsClock — server/internal/world/player_movement.go</summary>

[Source definition](../../server/internal/world/player_movement.go#L371)

```go
func (s *Session) sendPhysicsClock() {
	if s.char == nil {
		return
	}
	w := packet.New(packet.Opcode(proto.GXPhysicsClock))
	w.U16(realMap(s.char.MapID))
	w.U64(math.Float64bits(mapgeometry.PhysicsSeconds()))
	s.send(w.MustFinish(), "GX physics clock")
}
```

</details>

## Opcode 7f08

**`0x7F08` — GXPlayerCorrection**

Declarations: [proto.GXPlayerCorrection](../../internal/proto/player_physics.go#L8).

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [correctMovement (server/internal/world/player_movement.go)](../../server/internal/world/player_movement.go#L254)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1194)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>correctMovement — server/internal/world/player_movement.go</summary>

[Source definition](../../server/internal/world/player_movement.go#L254)

```go
func (s *Session) correctMovement(reason string) {
	m := &s.movement
	w := packet.New(packet.Opcode(proto.GXPlayerCorrection))
	w.U32(s.char.ID)
	w.F32(s.char.PosX)
	w.F32(s.char.PosY)
	w.F32(s.char.PosZ)
	w.U8(m.motion)
	w.U16(m.direction)
	w.Bool(m.jumping)
	w.F32(m.jumpFall)
	w.F32(m.fall)
	w.F32(m.slope.Frames)
	s.send(w.MustFinish(), "movement correction: "+reason)
}
```

</details>

<details><summary>DecodePlayerCorrection — internal/proto/player_physics.go</summary>

[Source definition](../../internal/proto/player_physics.go#L20)

```go
func DecodePlayerCorrection(p []byte) (c PlayerCorrection, err error) {
	if len(p) != 34 || p[21] > 1 {
		return c, fmt.Errorf("invalid player physics correction")
	}
	r := NewReader(p)
	c.ID = r.U32()
	c.X, c.Y, c.Z = r.F32(), r.F32(), r.F32()
	c.Motion, c.Direction, c.Jumping = r.U8(), r.U16(), r.Bool()
	c.JumpFall, c.Fall = r.F32(), r.F32()
	c.SlopeFrames = r.F32()
	if c.SlopeFrames < 0 {
		return PlayerCorrection{}, fmt.Errorf("negative slope pressure")
	}
	for _, v := range []float32{c.X, c.Y, c.Z, c.JumpFall, c.Fall, c.SlopeFrames} {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return PlayerCorrection{}, fmt.Errorf("non-finite physics correction")
		}
	}
	return c, r.Err()
}
```

```go
type PlayerCorrection struct {
	ID             uint32
	X, Y, Z        float32
	Motion         uint8
	Direction      uint16
	Jumping        bool
	JumpFall, Fall float32
	SlopeFrames    float32
}
```

</details>

## Opcode 7f09

**`0x7F09` — GXGMMapTeleport**

Declarations: [proto.GXGMMapTeleport](../../internal/proto/gm_map.go#L10).

Source notes: GXGMMapTeleport requests an exact map-image coordinate. The server owns Y.

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L273)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>GMMapTeleportPacket — internal/proto/gm_map.go</summary>

[Source definition](../../internal/proto/gm_map.go#L12)

```go
func GMMapTeleportPacket(mapID uint16, u, v float32) []byte {
	w := NewWriter(GXGMMapTeleport)
	w.U16(mapID)
	w.F32(u)
	w.F32(v)
	return w.Bytes()
}
```

</details>

## Opcode 7f0a

**`0x7F0A` — GXOJLimits**

Declarations: [proto.GXOJLimits](../../internal/proto/oj_limits.go#L9).

Source notes: GXOJLimits publishes static equipment recreation limits from the server DB.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendOJLimits (server/internal/world/oj_limits.go)](../../server/internal/world/oj_limits.go#L32)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L235)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendOJLimits — server/internal/world/oj_limits.go</summary>

[Source definition](../../server/internal/world/oj_limits.go#L32)

```go
func (s *Session) sendOJLimits() {
	limits := equipmentOJLimits(db.MustDefs())
	for len(limits) > 0 {
		n := min(len(limits), 256)
		payload := proto.OJLimitsPacket(limits[:n])
		w := packet.New(packet.Opcode(proto.GXOJLimits))
		w.Bytes(payload[2:])
		s.send(w.MustFinish(), "GX equipment OJ limits")
		limits = limits[n:]
	}
}
```

</details>

<details><summary>OJLimitsPacket — internal/proto/oj_limits.go</summary>

[Source definition](../../internal/proto/oj_limits.go#L13)

```go
func OJLimitsPacket(limits []OJLimit) []byte {
	w := NewWriter(GXOJLimits)
	w.U16(uint16(len(limits)))
	for _, v := range limits {
		w.U8(v.Type)
		w.U8(v.TypeID)
		w.U8(v.Stats)
		w.U8(v.Value)
	}
	return w.Bytes()
}
```

```go
type OJLimit struct{ Type, TypeID, Stats, Value uint8 }
```

</details>

<details><summary>DecodeOJLimits — internal/proto/oj_limits.go</summary>

[Source definition](../../internal/proto/oj_limits.go#L24)

```go
func DecodeOJLimits(p []byte) ([]OJLimit, error) {
	if len(p) < 4 || Op(p) != GXOJLimits || len(p) != 4+4*int(binary.LittleEndian.Uint16(p[2:4])) {
		return nil, fmt.Errorf("invalid OJ limits")
	}
	out := make([]OJLimit, 0, (len(p)-4)/4)
	for i := 4; i < len(p); i += 4 {
		v := OJLimit{p[i], p[i+1], p[i+2], p[i+3]}
		if v.Type == 0 || v.TypeID == 0 || v.Stats > 9 || v.Value > 99 || (v.Stats == 0) != (v.Value == 0) {
			return nil, fmt.Errorf("invalid OJ limit")
		}
		out = append(out, v)
	}
	return out, nil
}
```

</details>

## Opcode 7f0b

**`0x7F0B` — GXChestCloakFits**

Declarations: [proto.GXChestCloakFits](../../internal/proto/cloak_fits.go#L10).

Source notes: GXChestCloakFits supplies the native chest-dependent cloak fit. It is an additive cosmetic catalog; existing packet layouts and game rules are unchanged.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendChestCloakFits (server/internal/world/cloak_fits.go)](../../server/internal/world/cloak_fits.go#L28)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L218)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendChestCloakFits — server/internal/world/cloak_fits.go</summary>

[Source definition](../../server/internal/world/cloak_fits.go#L28)

```go
func (s *Session) sendChestCloakFits() {
	rows := chestCloakFits(db.MustDefs())
	for len(rows) > 0 {
		n := min(len(rows), 256)
		payload := proto.ChestCloakFitsPacket(rows[:n])
		w := packet.New(packet.Opcode(proto.GXChestCloakFits))
		w.Bytes(payload[2:])
		s.send(w.MustFinish(), "GX chest cloak fits")
		rows = rows[n:]
	}
}
```

</details>

<details><summary>ChestCloakFitsPacket — internal/proto/cloak_fits.go</summary>

[Source definition](../../internal/proto/cloak_fits.go#L14)

```go
func ChestCloakFitsPacket(rows []ChestCloakFit) []byte {
	w := NewWriter(GXChestCloakFits)
	w.U16(uint16(len(rows)))
	for _, v := range rows {
		w.U8(v.Type)
		w.U8(v.TypeID)
		w.U8(v.Fit)
	}
	return w.Bytes()
}
```

```go
type ChestCloakFit struct{ Type, TypeID, Fit uint8 }
```

</details>

<details><summary>DecodeChestCloakFits — internal/proto/cloak_fits.go</summary>

[Source definition](../../internal/proto/cloak_fits.go#L24)

```go
func DecodeChestCloakFits(p []byte) ([]ChestCloakFit, error) {
	if len(p) < 4 || Op(p) != GXChestCloakFits || len(p) != 4+3*int(binary.LittleEndian.Uint16(p[2:4])) {
		return nil, fmt.Errorf("invalid chest cloak fits")
	}
	rows := make([]ChestCloakFit, 0, (len(p)-4)/3)
	for i := 4; i < len(p); i += 3 {
		v := ChestCloakFit{p[i], p[i+1], p[i+2]}
		if v.Type == 0 || v.TypeID == 0 || v.Fit > 7 {
			return nil, fmt.Errorf("invalid chest cloak fit")
		}
		rows = append(rows, v)
	}
	return rows, nil
}
```

</details>

## Opcode 7f0c

**`0x7F0C` — GXFallDamage**

Declarations: [proto.GXFallDamage](../../internal/proto/fall_damage.go#L7).

Source notes: GXFallDamage is an additive server notification. Health still uses 0x0521. No client-authored fall height or damage report is accepted by the server.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [applyFallDamage (server/internal/world/fall_damage.go)](../../server/internal/world/fall_damage.go#L18)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L816)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>applyFallDamage — server/internal/world/fall_damage.go</summary>

[Source definition](../../server/internal/world/fall_damage.go#L18)

```go
func (s *Session) applyFallDamage(drop float32) {
	if !config.FallDamageEnabled() || s.char == nil || s.dead || s.invulnerable() {
		return
	}
	st := s.stats()
	damage := min(st.CurrentHP, fallDamage(drop, st.MaxHP))
	if damage == 0 {
		return
	}
	s.cancelTravelCast()
	s.commitMovement()
	s.cancelMove()
	s.breakStealth()
	s.char.CurrentHP = satSub(s.char.CurrentHP, damage)
	s.persist()
	w := packet.New(packet.Opcode(proto.GXFallDamage))
	w.U32(s.char.ID)
	w.U32(damage)
	frame := w.MustFinish()
	s.send(frame, "GX fall damage")
	s.sendToViewers(frame, "GX fall damage (view)")
	current := s.stats()
	s.send(buildCurrentHitpoints(&current), "current hp after fall")
	if current.CurrentHP > 0 {
		s.wake()
		return
	}
	s.damageLedger = nil
	state, death := buildCharacterDeathState(s.char.ID), buildCharacterDeath(s.char.ID, deathTypeMob, 0)
	s.send(state, "fall death state")
	s.send(death, "fall death")
	s.sendToViewers(state, "fall death state (view)")
	s.sendToViewers(death, "fall death (view)")
	if s.player != nil {
		s.player.SetAlive(false)
		duelLeave(s.player, 3, true)
		guildDuelLeave(s.player, nil, 3)
	}
	s.dead, s.autoAttack, s.cast = true, false, nil
	s.clearBuffsOnDeath()
	// Environmental deaths grant no PvP credit, item/XP loss or Ultimate deletion.
	s.deathLoss = -1
}
```

</details>

<details><summary>DecodeFallDamage — internal/proto/fall_damage.go</summary>

[Source definition](../../internal/proto/fall_damage.go#L11)

```go
func DecodeFallDamage(p []byte) (v FallDamage, err error) {
	if len(p) != 10 {
		return v, fmt.Errorf("fall damage: expected 10 bytes, got %d", len(p))
	}
	r := NewReader(p)
	v.Target, v.Amount = r.U32(), r.U32()
	return v, r.Err()
}
```

```go
type FallDamage struct{ Target, Amount uint32 }
```

</details>

## Opcode 7f0d

**`0x7F0D` — GXLadder**

Declarations: [proto.GXLadder](../../internal/proto/ladder.go#L7).

Source notes: GXLadder is additive. Requests contain guide index + mode; notifications also identify the actor. Native movement positions and flags retain their layouts.

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L30)

**S→C serializers:** [buildLadderState (server/internal/world/ladders.go)](../../server/internal/world/ladders.go#L11)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L806)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildLadderState — server/internal/world/ladders.go</summary>

[Source definition](../../server/internal/world/ladders.go#L11)

```go
func buildLadderState(id uint32, index uint16, mode uint8) []byte {
	w := packet.New(packet.Opcode(proto.GXLadder))
	w.U32(id)
	w.U16(index)
	w.U8(mode)
	return w.MustFinish()
}
```

</details>

<details><summary>LadderPacket — internal/proto/ladder.go</summary>

[Source definition](../../internal/proto/ladder.go#L15)

```go
func LadderPacket(index uint16, mode uint8) []byte {
	w := NewWriter(GXLadder)
	w.U16(index)
	w.U8(mode)
	return w.Bytes()
}
```

</details>

<details><summary>DecodeLadderState — internal/proto/ladder.go</summary>

[Source definition](../../internal/proto/ladder.go#L21)

```go
func DecodeLadderState(p []byte) (v LadderState, err error) {
	if len(p) != 9 {
		return v, fmt.Errorf("ladder state: expected 9 bytes, got %d", len(p))
	}
	r := NewReader(p)
	v.ID = r.U32()
	v.Index = r.U16()
	v.Mode = r.U8()
	if v.Mode > 3 {
		return v, fmt.Errorf("invalid ladder mode")
	}
	return v, r.Err()
}
```

```go
type LadderState struct {
	ID    uint32
	Index uint16
	Mode  uint8
}
```

</details>

## Opcode 7f0e

**`0x7F0E` — GXSkillFailure**

Declarations: [proto.GXSkillFailure](../../internal/proto/skill_failure.go#L6).

Source notes: GXSkillFailure supplies a native SysMsg ID without inventing native result codes.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [sendSkillMessage (server/internal/world/skill_range.go)](../../server/internal/world/skill_range.go#L50)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L800)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendSkillMessage — server/internal/world/skill_range.go</summary>

[Source definition](../../server/internal/world/skill_range.go#L50)

```go
func (s *Session) sendSkillMessage(skillID, message uint16) {
	w := packet.New(packet.Opcode(proto.GXSkillFailure))
	w.U16(skillID)
	w.U16(message)
	s.send(w.MustFinish(), "localized skill failure")
}
```

</details>

<details><summary>DecodeSkillFailure — internal/proto/skill_failure.go</summary>

[Source definition](../../internal/proto/skill_failure.go#L10)

```go
func DecodeSkillFailure(p []byte) (v SkillFailure, err error) {
	if len(p) != 6 {
		return v, fmt.Errorf("invalid GX skill failure length")
	}
	r := NewReader(p)
	v.Skill, v.Message = r.U16(), r.U16()
	return v, r.Err()
}
```

```go
type SkillFailure struct{ Skill, Message uint16 }
```

</details>

## Opcode 7f0f

**`0x7F0F` — GXTargetEffectsQuery**

Declarations: [proto.GXTargetEffectsQuery](../../internal/proto/target_effects.go#L11).

Source notes: GX requests a selected target's native snapshot; no native request is claimed.

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L28)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>TargetEffectsQuery — internal/proto/target_effects.go</summary>

[Source definition](../../internal/proto/target_effects.go#L30)

```go
func TargetEffectsQuery(kind uint8, id uint32) []byte {
	w := NewWriter(GXTargetEffectsQuery)
	w.U8(kind)
	w.U32(id)
	return w.Bytes()
}
```

</details>

## Opcode 7f10

**`0x7F10` — GXVehicleState**

Declarations: [proto.GXVehicleState](../../internal/proto/mount_state.go#L12).

Source notes: Native requests and invitation messages. 0x021C is also the server's passenger attachment notification; direction determines its layout.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildVehicleState (server/internal/world/vehicle.go)](../../server/internal/world/vehicle.go#L359)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L377)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildVehicleState — server/internal/world/vehicle.go</summary>

[Source definition](../../server/internal/world/vehicle.go#L359)

```go
func buildVehicleState(v proto.VehicleState) []byte {
	raw := proto.VehicleStatePacket(v)
	w := packet.New(packet.Opcode(proto.GXVehicleState))
	w.Bytes(raw[2:])
	return w.MustFinish()
}
```

</details>

<details><summary>VehicleStatePacket — internal/proto/mount_state.go</summary>

[Source definition](../../internal/proto/mount_state.go#L31)

```go
func VehicleStatePacket(v VehicleState) []byte {
	w := NewWriter(GXVehicleState)
	w.U32(v.ID)
	w.U8(v.State)
	w.U8(v.Race)
	w.U8(v.Model)
	w.U8(v.Speed)
	w.U8(v.Seats)
	w.U32(v.Driver)
	w.U32(v.Passenger)
	w.U32(v.CastMS)
	w.U32(v.CooldownMS)
	return w.Bytes()
}
```

```go
type VehicleState struct {
	ID                               uint32
	State, Race, Model, Speed, Seats uint8
	Driver, Passenger                uint32
	CastMS, CooldownMS               uint32
}
```

</details>

<details><summary>DecodeVehicleState — internal/proto/mount_state.go</summary>

[Source definition](../../internal/proto/mount_state.go#L45)

```go
func DecodeVehicleState(p []byte) (VehicleState, error) {
	if len(p) != 27 || Op(p) != GXVehicleState {
		return VehicleState{}, fmt.Errorf("invalid vehicle state size")
	}
	r := NewReader(p)
	v := VehicleState{ID: r.U32(), State: r.U8(), Race: r.U8(), Model: r.U8(), Speed: r.U8(), Seats: r.U8(), Driver: r.U32(), Passenger: r.U32(), CastMS: r.U32(), CooldownMS: r.U32()}
	if v.State > VehiclePassenger || v.Race > 3 || (v.State >= VehicleDriver && (v.Speed < 4 || v.Speed > 5 || v.Seats < 1 || v.Seats > 2 || v.Driver == 0)) {
		return v, fmt.Errorf("invalid vehicle state")
	}
	return v, r.Err()
}
```

</details>

## Opcode 7f11

**`0x7F11` — GXSkillDash**

Declarations: [proto.GXSkillDash](../../internal/proto/skill_dash.go#L11).

Source notes: GXSkillDash is a server-authored Stinger trajectory. Native skill and move packets keep their layouts. Clients never submit dash destinations or speed.

**C→S receive adapters:** no direct mapping found; inspect source before assuming support or direction.

**S→C serializers:** [buildSkillDash (server/internal/world/skill_dash.go)](../../server/internal/world/skill_dash.go#L83)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L1188)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>buildSkillDash — server/internal/world/skill_dash.go</summary>

[Source definition](../../server/internal/world/skill_dash.go#L83)

```go
func buildSkillDash(d proto.SkillDash) []byte {
	raw := proto.SkillDashPacket(d)
	w := packet.New(packet.Opcode(proto.GXSkillDash))
	w.Bytes(raw[2:])
	return w.MustFinish()
}
```

</details>

<details><summary>SkillDashPacket — internal/proto/skill_dash.go</summary>

[Source definition](../../internal/proto/skill_dash.go#L24)

```go
func SkillDashPacket(d SkillDash) []byte {
	w := NewWriter(GXSkillDash)
	w.U32(d.Source)
	for _, p := range [][3]float32{d.From, d.To} {
		for _, v := range p {
			w.F32(v)
		}
	}
	bits := math.Float64bits(d.Started)
	w.U32(uint32(bits))
	w.U32(uint32(bits >> 32))
	w.F32(d.Contact)
	if d.Stopped {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
```

```go
type SkillDash struct {
	Stopped  bool
	Source   uint32
	From, To [3]float32
	Started  float64 // synchronized map physics seconds
	Contact  float32 // native ALT contact frame
}
```

</details>

<details><summary>DecodeSkillDash — internal/proto/skill_dash.go</summary>

[Source definition](../../internal/proto/skill_dash.go#L43)

```go
func DecodeSkillDash(p []byte) (d SkillDash, err error) {
	if len(p) != 43 || p[42] > 1 || Op(p) != GXSkillDash {
		return d, fmt.Errorf("invalid skill dash size")
	}
	r := NewReader(p)
	d.Source = r.U32()
	for i := range d.From {
		d.From[i] = r.F32()
	}
	for i := range d.To {
		d.To[i] = r.F32()
	}
	d.Started = math.Float64frombits(r.U64())
	d.Contact = r.F32()
	d.Stopped = r.Bool()
	if d.Source == 0 || math.IsNaN(d.Started) || math.IsInf(d.Started, 0) || d.Started < 0 || d.Contact < 10 || d.Contact > 30 {
		return SkillDash{}, fmt.Errorf("invalid skill dash metadata")
	}
	for _, v := range append(d.From[:], d.To[:]...) {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return SkillDash{}, fmt.Errorf("non-finite skill dash")
		}
	}
	if math.IsNaN(float64(d.Contact)) || d.Distance() > MaxSkillDashDistance {
		return SkillDash{}, fmt.Errorf("invalid skill dash range")
	}
	return d, r.Err()
}
```

</details>

## Opcode 7f12

**`0x7F12` — GXEnchantRules, Opcode**

Declarations: [proto.GXEnchantRules](../../internal/proto/enchant_rules.go#L9), [authwire.Opcode](../../internal/authwire/credentials.go#L12).

Source notes: GXEnchantRules publishes server INI bonuses for authoritative tooltips.

**C→S receive adapters:** [frame (server/cmd/gx_login/main.go)](../../server/cmd/gx_login/main.go#L224)

**S→C serializers:** [sendEnchantRules (server/internal/world/enchant_rules.go)](../../server/internal/world/enchant_rules.go#L97)

**Client receive reducers:** [Apply (internal/game/state.go)](../../internal/game/state.go#L229)

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendEnchantRules — server/internal/world/enchant_rules.go</summary>

[Source definition](../../server/internal/world/enchant_rules.go#L97)

```go
func (s *Session) sendEnchantRules() {
	payload := proto.EnchantRulesPacket(proto.EnchantRules{MaxLevel: enchantConfig.MaxLevel, WeaponBonus: weaponEnchantAttackBonus, ArmorBonus: armorEnchantAbsorption})
	w := packet.New(packet.Opcode(proto.GXEnchantRules))
	w.Bytes(payload[2:])
	s.send(w.MustFinish(), "GX enchant rules")
}
```

</details>

<details><summary>EnchantRulesPacket — internal/proto/enchant_rules.go</summary>

[Source definition](../../internal/proto/enchant_rules.go#L16)

```go
func EnchantRulesPacket(r EnchantRules) []byte {
	w := NewWriter(GXEnchantRules)
	w.U8(r.MaxLevel)
	for _, table := range [2][50]uint32{r.WeaponBonus, r.ArmorBonus} {
		for _, v := range table {
			w.U32(v)
		}
	}
	return w.Bytes()
}
```

```go
type EnchantRules struct {
	MaxLevel                uint8
	WeaponBonus, ArmorBonus [50]uint32
}
```

</details>

<details><summary>DecodeEnchantRules — internal/proto/enchant_rules.go</summary>

[Source definition](../../internal/proto/enchant_rules.go#L27)

```go
func DecodeEnchantRules(p []byte) (EnchantRules, error) {
	var r EnchantRules
	if len(p) != 403 || Op(p) != GXEnchantRules || p[2] > 49 {
		return r, fmt.Errorf("invalid enchant rules")
	}
	r.MaxLevel = p[2]
	off := 3
	for _, table := range []*[50]uint32{&r.WeaponBonus, &r.ArmorBonus} {
		for i := range table {
			v := binary.LittleEndian.Uint32(p[off:])
			off += 4
			if v > 2147483647 {
				return EnchantRules{}, fmt.Errorf("invalid enchant bonus")
			}
			table[i] = v
		}
	}
	return r, nil
}
```

</details>

<details><summary>Validate — internal/authwire/credentials.go</summary>

[Source definition](../../internal/authwire/credentials.go#L16)

```go
func Validate(user, password string) error {
	if len(user) < 1 || len(user) > MaxUser || len(password) < 1 || len(password) > MaxPassword || !utf8.ValidString(user) || !utf8.ValidString(password) || strings.ContainsRune(user, 0) || strings.ContainsRune(password, 0) {
		return errors.New("credentials exceed UTF-8 byte limits (18/128) or contain invalid text")
	}
	return nil
}
```

</details>

<details><summary>Encode — internal/authwire/credentials.go</summary>

[Source definition](../../internal/authwire/credentials.go#L22)

```go
func Encode(user, password string) ([]byte, error) {
	if err := Validate(user, password); err != nil {
		return nil, err
	}
	out := binary.LittleEndian.AppendUint16(nil, Opcode)
	out = append(out, byte(len(user)), byte(len(password)))
	out = append(out, user...)
	out = append(out, password...)
	return out, nil
}
```

</details>

<details><summary>Decode — internal/authwire/credentials.go</summary>

[Source definition](../../internal/authwire/credentials.go#L32)

```go
func Decode(payload []byte) (string, string, error) {
	if len(payload) < 4 || binary.LittleEndian.Uint16(payload) != Opcode {
		return "", "", errors.New("invalid credential extension")
	}
	u, p := int(payload[2]), int(payload[3])
	if len(payload) != 4+u+p {
		return "", "", errors.New("invalid credential lengths")
	}
	user, password := string(payload[4:4+u]), string(payload[4+u:])
	return user, password, Validate(user, password)
}
```

</details>

## Opcode 7f13

**`0x7F13` — GXTeleportStone**

Declarations: [proto.GXTeleportStone](../../internal/proto/teleport_stone.go#L4).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L77)

**S→C serializers:** [sendTeleportStone (server/internal/world/teleport_stone.go)](../../server/internal/world/teleport_stone.go#L49)

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>sendTeleportStone — server/internal/world/teleport_stone.go</summary>

[Source definition](../../server/internal/world/teleport_stone.go#L49)

```go
func (s *Session) sendTeleportStone(action uint8, ok bool, message string, points [proto.TeleportPointCount]db.TeleportPoint) {
	w := packet.New(packet.Opcode(proto.GXTeleportStone))
	w.U8(action)
	w.Bool(ok)
	writeChatText(w, message)
	for _, p := range points {
		w.Bool(p.Saved)
		w.U16(p.Map)
		w.F32(p.X)
		w.F32(p.Y)
		w.F32(p.Z)
	}
	if action == proto.TeleportCasting {
		w.U16(uint16(teleportStoneCastDuration / time.Millisecond))
	}
	s.send(w.MustFinish(), "recorded points")
}
```

</details>

<details><summary>TeleportStonePacket — internal/proto/teleport_stone.go</summary>

[Source definition](../../internal/proto/teleport_stone.go#L27)

```go
func TeleportStonePacket(action, slot uint8) []byte {
	w := NewWriter(GXTeleportStone)
	w.U8(action)
	w.U8(slot)
	return w.Bytes()
}
```

</details>

## Opcode 7f14

**`0x7F14` — GXSummonRune**

Declarations: [proto.GXSummonRune](../../internal/proto/summon_rune.go#L4).

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L75)

**S→C serializers:** [summonResult (server/internal/world/summon_rune.go)](../../server/internal/world/summon_rune.go#L58), [publishSummonCast (server/internal/world/summon_rune.go)](../../server/internal/world/summon_rune.go#L65), [tickPartySummon (server/internal/world/summon_rune.go)](../../server/internal/world/summon_rune.go#L134)

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>summonResult — server/internal/world/summon_rune.go</summary>

[Source definition](../../server/internal/world/summon_rune.go#L58)

```go
func (s *Session) summonResult(message string) {
	w := packet.New(packet.Opcode(proto.GXSummonRune))
	w.U8(proto.SummonResult)
	writeChatText(w, message)
	s.send(w.MustFinish(), "party summon result")
}
```

</details>

<details><summary>publishSummonCast — server/internal/world/summon_rune.go</summary>

[Source definition](../../server/internal/world/summon_rune.go#L65)

```go
func (s *Session) publishSummonCast(active bool) {
	w := packet.New(packet.Opcode(proto.GXSummonRune))
	if active {
		w.U8(proto.SummonCast)
	} else {
		w.U8(proto.SummonCastEnd)
	}
	w.U32(s.char.ID)
	ms := uint16(0)
	if active {
		ms = uint16(summonCastTime / time.Millisecond)
	}
	w.U16(ms)
	s.broadcastNear(w.MustFinish(), "party summon cast", true)
}
```

</details>

<details><summary>tickPartySummon — server/internal/world/summon_rune.go</summary>

[Source definition](../../server/internal/world/summon_rune.go#L134)

```go
func (s *Session) tickPartySummon(now time.Time) {
	if s.summonCastEnd.IsZero() {
		return
	}
	if s.char == nil || s.player == nil || s.dead || s.char.CurrentHP == 0 || s.actionBlocked() || s.inCombat() {
		s.cancelPartySummon()
		return
	}
	if now.Before(s.summonCastEnd) {
		return
	}
	s.summonCastEnd = time.Time{}
	s.publishSummonCast(false)
	group, targets := partySummonTargets(s.player)
	if len(targets) == 0 || !teleportStoneField(s.char.MapID) {
		s.summonResult("There are no party members to summon.")
		return
	}

	offer := partySummonOffer{summonOfferSerial.Add(1), s.player, group, now.Add(summonOfferTime)}
	w := packet.New(packet.Opcode(proto.GXSummonRune))
	w.U8(proto.SummonOffer)
	w.U32(offer.id)
	w.U32(s.char.ID)
	w.U16(realMap(s.char.MapID))
	w.U16(uint16(summonOfferTime / time.Millisecond))
	w.FixedString(s.char.Name, 21)
	frame := w.MustFinish()
	sent := false
	for _, p := range targets {
		if !p.IsAlive() || p.Faction() != s.faction {
			continue
		}
		p.mailMu.Lock()
		p.summonOffer = &offer
		p.mailMu.Unlock()
		_ = p.Send(frame, "party summon offer")
		sent = true
	}
	if !sent {
		s.summonResult("There are no party members to summon.")
	}
}
```

</details>

<details><summary>SummonAnswerPacket — internal/proto/summon_rune.go</summary>

[Source definition](../../internal/proto/summon_rune.go#L35)

```go
func SummonAnswerPacket(id uint32, accept bool) []byte {
	w := NewWriter(GXSummonRune)
	w.U8(SummonAnswer)
	w.U32(id)
	w.Bool(accept)
	return w.Bytes()
}
```

</details>

<details><summary>SummonCancelPacket — internal/proto/summon_rune.go</summary>

[Source definition](../../internal/proto/summon_rune.go#L43)

```go
func SummonCancelPacket() []byte {
	w := NewWriter(GXSummonRune)
	w.U8(SummonCancel)
	return w.Bytes()
}
```

</details>

## Opcode 7f15

**`0x7F15` — GXPartyInviteName**

Declarations: [packet.GXPartyInviteName](../../server/internal/packet/packet.go#L328), [proto.GXPartyInviteName](../../internal/proto/party.go#L107).

Source notes: GXPartyInviteName is the global /party <name> invitation request. GXPartyInviteName is the GX /party <name> request: the party invitation of 0x0B09 without the proximity or same-map requirement. The target still receives the native 0x0B09 prompt.

**C→S receive adapters:** [dispatch (server/internal/world/dispatch.go)](../../server/internal/world/dispatch.go#L118)

**S→C serializers:** no direct mapping found; inspect source before assuming support or direction.

**Client receive reducers:** no direct mapping found; inspect source before assuming support or direction.

### Wire layouts and validation

These are the exact source serializers/decoders. Parameter names and assignments identify fields; conditions identify optional variants. Referenced helpers and domain validation remain authoritative at the linked source.

<details><summary>PartyGlobalInvitePacket — internal/proto/party.go</summary>

[Source definition](../../internal/proto/party.go#L109)

```go
func PartyGlobalInvitePacket(name string) []byte {
	w := NewWriter(GXPartyInviteName)
	w.Fixed(name, 21, true)
	return w.Bytes()
}
```

</details>
