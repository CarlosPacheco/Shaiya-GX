# Protocol and server authority

Shaiya GX targets the EP5.4 client protocol implemented by GX Server. Packets are serialized field by field with explicit widths and ordering; Go struct layout is never used as a wire format.

`internal/proto` provides one reader/writer implementation per supported packet family, with defensive bounds checks and codec tests. `internal/netx` implements framing and cryptography; `internal/session` owns the login-to-world transition. Passwords are supplied through the graphical login or an environment variable for diagnostics, never stored in the repository.

The server owns authentication, character rules, combat, skills, inventory, equipment, purchases, quests, trade, guild state and persistence. Client interfaces wait for confirmation and handle timeout/rejection without inventing successful state changes.

Examples of implemented packet families include character entry/movement, target selection, normal and player attacks, skill activation/cooldown, resource updates, bags and equipment, pickup, chat, party/raid, NPC services, storage, quests, trade, duels, guild and Item Mall operations.

Automatic action approach is client movement prediction. It selects a known nearby target, follows its position and sends an attack or pickup request in reach. Item removal, damage and rewards still require server packets. Manual movement, cancellation or target disappearance cancels approach.

Native addresses and counterpart behavior are cited in source comments. Unknown fields remain explicitly unknown; protocol parity is not inferred from a visually working screen. Integration tests run against isolated GX Server instances with disposable accounts and a temporary database.

GX extension `0x7F02` communicates authorized character concealment (`u32` ID,
`u8` boolean). Unauthorized viewers receive native `0x0202` leave and no entity
updates. See [Stealth and GM protection](stealth-and-gm-protection.md).

GX `0x7F04` sends an atomically assembled, chunked map quest-NPC catalog. It
contains definition keys and X/Z anchors without interactive entity IDs. See
[Quests and map markers](quests-and-map.md) for layout and cache invalidation.

## Map-click GM teleport

Revision `0x11122208` adds client request `0x7F09`: map ID (`uint16`),
normalized image U and V (`float32` each). Values must be finite and within
0–1. X = U × map size; Z = (1 − V) × map size. The server verifies its account
GM role, rejects unknown/instance IDs, blocked destinations and active trades,
and resolves an exact supported X/Z destination and height with its navigation
and original collision geometry. Success uses the normal map-position and
movement responses; diagnostics use `GXGMState`. There is no `/goto` command.

`0x7F0A` publishes equipment OJ metadata at entry in chunks: row count
(`uint16`), then type, type ID, maximum distinct stats and per-stat value
(`uint8` each). Zero count/value disables recreation for that definition.
The client uses server limits in item tooltips instead of stale catalog fields.

`0x7F0B` publishes chest-dependent cloak fits at character entry: row count
(`uint16`), then item type, type ID and zero-based fit (`uint8` each, fit 0–7).
Rows are sent in chunks of up to 256 using the standard server length framing.
This additive cosmetic catalog corrects stale client Item.SData without changing
item requirements, database contents, existing packet layouts or movement rules.
The client falls back to its local catalog when connected to an older server.
Update and restart both client and world to apply the synchronized fit.

`0x7F0C` is a server-only fall-damage notification (enabled by default;
opt out with `RAPTURE_FALL_DAMAGE=false`): target ID and actual damage
(`uint32` each; 10 bytes including opcode, 12 including the server length).
Native `0x0521` still supplies health. The server never accepts client-reported
fall distances/damage. SysMsg 5 reports the confirmed amount. See
[GX fall rules](../fall-damage.md) for thresholds, exemptions and respawn policy.

`0x7F0D` carries functional ladder intent and authoritative actor state. Requests
contain guide index (uint16) and mode (uint8); replies prepend actor ID (uint32).
Modes are detach/hold/ascend/descend (0–3). Existing native movement framing and
flags are unchanged. See [functional ladders](../ladders.md) for controls,
collision validation, catalog generation and synchronization.

Native selected-target effects use `0x030B` snapshots and `0x030C/0x030D`
application/removal layouts. GX requests the selected visible actor through
`0x7F0F` (kind uint8 + target uint32). Server-only `0x7F0E` carries skill ID and
native SysMsg ID (uint16 each) for localized reach/obstacle refusals. See
[skill targeting and effects](../skill-targeting-and-effects.md) for exact fields.

Revision `0x11122209` is required by gx-v0.1.10, including the authoritative
mount state. GX `0x7F10` separates MON index zero, race, speed, seat count,
cast/cooldown timers and driver/passenger links; native mount requests use
0x0216/0x0217 and 0x021C–0x021E. See [mounts](mounts.md).

## Complete implementation reference

The [server wire guide](server-protocol.md) explains framing, field widths,
transport transitions and phase admission. The [complete opcode catalog](server-opcodes.md)
covers 275 current source opcodes in fifteen families with exact codecs, server
adapters and client reducer links. The [codec appendix](server-codecs.md) includes
all shared helpers and row definitions. `0x7F12` adds bounded UTF-8 credentials for
passwords up to 128 bytes; replies keep the legacy `0xA102` layout. Remote TLS
wraps existing game frames; see [security](server-security.md).
