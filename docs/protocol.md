# Protocol and server authority

Shaiya GX targets the EP5.4 client protocol implemented by GX Server. Packets are serialized field by field with explicit widths and ordering; Go struct layout is never used as a wire format.

`internal/proto` provides one reader/writer implementation per supported packet family, with defensive bounds checks and codec tests. `internal/netx` implements framing and cryptography; `internal/session` owns the login-to-world transition. Passwords are supplied through the graphical login or an environment variable for diagnostics, never stored in the repository.

The server owns authentication, character rules, combat, skills, inventory, equipment, purchases, quests, trade, guild state and persistence. Client interfaces wait for confirmation and handle timeout/rejection without inventing successful state changes.

Examples of implemented packet families include character entry/movement, target selection, normal and player attacks, skill activation/cooldown, resource updates, bags and equipment, pickup, chat, party/raid, NPC services, storage, quests, trade, duels, guild and Item Mall operations.

Automatic action approach is client movement prediction. It selects a known nearby target, follows its position and sends an attack or pickup request in reach. Item removal, damage and rewards still require server packets. Manual movement, cancellation or target disappearance cancels approach.

Native addresses and counterpart behavior are cited in source comments. Unknown fields remain explicitly unknown; protocol parity is not inferred from a visually working screen. Integration tests run against isolated GX Server instances with disposable accounts and a temporary database.

Mount requests retain the native 0x0216/0x0217 and 0x021C–0x021E family.
GX mount state (0x7F10) carries explicit model, speed, seats, cast/cooldown
timing and conductor/passenger IDs; see [mounts](mounts.md). Both endpoints
must use matching GX builds.

Stinger uses server-only `GXSkillDash` (`0x7F11`): source uint32, start XYZ and
destination XYZ (six float32), start physics time float64, contact frame float32,
and stopped uint8 (0/1). The body excluding opcode is 41 bytes. The synchronized
map physics clock drives prediction; stopped updates terminate both self and
remote presentation at the server position. Destinations, speed and movement
credit are never accepted from a client. Native skill/move layouts are unchanged.

## Monster movement and contact

World uses the same movement segment for its position queries and the duration
in `GXMobMove` (`0x7F06`). When a moving mob dies, a zero-duration update can
place the corpse at its last server position before `0x0606`. The client
uses its rendered position to approach and aim, and presents a lethal melee
or ranged weapon hit at its animation or projectile contact. HP and targeting
still follow the server.

Monster skill packets `0x060B` and `0x060C` carry the attack animation slot.
Additional `0x060D` results apply their effects without replaying the cast.

## Map-load acknowledgements

Initial character entry sends `0x0201` after the initial `0x010B` skill-bar
snapshot. Every subsequent self `0x020B` placement that changes the map also
requires one `0x0201`, whether the source is a GM command, portal or another
teleport. The server admits normal gameplay again after this acknowledgement.
Same-map teleports, other actors and the server's same-map placement echo must
not generate another acknowledgement. The shared reducer reports this boundary
through `Event.MapLoadAck`; the graphical client, CLI and protocol fixtures use
that event without changing gameplay packet layouts.


GX `0x7F28` carries party assist. A request contains a mode byte (0 to
publish a selection, 1 to ask for an ally's selection) followed by a
little-endian uint32. The reply contains the ally ID and target ID, both
uint32. World checks group membership, shared map and visibility before
replying. A target ID of zero represents an empty selection.
