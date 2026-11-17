# Server package responsibilities and contracts

This is the package map for the integrated GX Server, including all runtime
packages, commands and the Lua runtime. Detailed file-by-file declarations are
in [infrastructure](server-reference-infrastructure.md),
[storage](server-reference-storage.md) and [world](server-reference-world.md).
Those inventories include private functions, direct GX imports, test files and
both OS alternatives. The [architecture guide](server-architecture.md) explains
how the contracts fit together.

## Commands

| Package | Entrypoint and dependencies | Operational contract |
|---|---|---|
| `server/cmd/gx_login` | `run`, `handle`, `client.frame`; config, crypto, password/db, session, netx, controlplane | 30 s login lifetime, 5 s writes, single-use ticket after successful credentials; no world simulation |
| `server/cmd/gx_world` | `run`; db, world, config, netx, controlplane | Acquire one-world lease, load required state before accepting, drain sessions before storage close; startup failures return nonzero |
| `server/cmd/gx_console` | Interactive console or `admin.Run(args)` | Same control path as GUI; account create/password and `tls-init` are local operator actions |
| `server/cmd/gx_server` | Fyne app using `control` | UI owns controls/status; detached core lifetime is independent of the window |
| `server/cmd/gx_mapbuild` | Offline generator using archive/world/mapnav/mapgeometry | Reads local client resources, writes server navigation/collision products; `-maps`, `-data`, `-output`, `-server-maps`, `-physics-only` select work |

## Infrastructure packages

### admin

[Source/API](server-reference-infrastructure.md#serverinternaladmin).
`Run` dispatches `account create NAME [--admin]`, `account password NAME` and
`tls-init`. Password input is hidden on a terminal, with bounded stdin/environment
alternatives. No secret belongs in CLI arguments. Storage calls use the same
Argon2id implementation as login; duplicate accounts are refused. Password changes
increment the credential version and revoke tickets/active authorization.

TLS provisioning generates an Ed25519 CA and server certificate in a fresh private
directory. Server private keys stay outside the generated `client/` bundle. The
CA lasts five years, leaf certificates 90 days; replacement/rotation is an operator
responsibility. Existing output directories are refused rather than overwritten.

### config

[Source/API](server-reference-infrastructure.md#serverinternalconfig).
`LoadRuntimeEnv`/`LoadRuntimeEnvFrom` parse `.local/runtime.env` before applying
values; malformed input does not partially update the environment. Explicit
process environment takes precedence. `resolvePaths` makes data/database paths
installation-relative. `Network` returns immutable validated admission/input/
output policy; zero, malformed and excessive values fail startup.

`DSN` builds a file URI with WAL, FULL synchronization, foreign keys, 10 s busy
timeout and IMMEDIATE transactions. Windows drive letters remain URI path data.
`LoginListen`, `WorldListen`, `PublicIP`, `DatabasePath`, `DatabaseSeed`, `DataDir`
and `FallDamageEnabled` preserve compatibility environment names.

### control

[Source/API](server-reference-infrastructure.md#serverinternalcontrol).
Shared start/status/stop and safe runtime initialization for both frontends.
Services launch detached and are tracked by exact executable/PID identity. Stop
tries authenticated HTTP control first, waits for graceful exit, then invokes
platform fallback. Windows has its own launch/process logic; Unix uses signals.
Do not replace this with `killall` or filename-only termination.

`database.go` validates/initializes the missing runtime with a cross-process lock;
existing mutable state is never replaced by seed packaging. Python storage tools
use the corresponding lock convention.

### controlplane

[Source/API](server-reference-infrastructure.md#serverinternalcontrolplane).
`Start` binds a numeric loopback address, defaults to an ephemeral port and writes
an owner-only `runtime/control/<service>.json` descriptor. `GET /healthz` and
`GET /metrics` are local read endpoints. `POST /shutdown` and optional pprof need
a random bearer token. `RequestStop` verifies PID/executable/loopback and disables
proxy routing. Cleanup removes only its own descriptor token, preserving a newer
process descriptor. HTTP time/header limits bound administrative connections.

### crypto

[Source/API](server-reference-infrastructure.md#serverinternalcrypto).
Compatibility implementation of login RSA exchange, HMAC-SHA256 AES-material
derivation, little-endian AES-128-CTR and expanded world XOR. Separate send/receive
state matters. These legacy layers preserve packet compatibility; they do not
replace authenticated TLS for remote traffic. Never log derived material or
handoff identities. [Wire guide](server-protocol.md) defines the transition.

### filelock

[Source/API](server-reference-infrastructure.md#serverinternalfilelock).
`Acquire(ctx,path)` obtains an OS-backed exclusive lock with cancellation. Unix
and Windows provide platform implementations. Process exit releases ownership;
a leftover lock file is not a stale lock directory to delete blindly. Runtime
initialization and world ownership use distinct lock paths. Do not unlink a live
lock file to bypass exclusion.

### logx

[Source/API](server-reference-infrastructure.md#serverinternallogx).
`New`, `Printf`, `Errorf`, `Event` and tagged loggers emit JSON records with UTC
time, level, component, optional session and message/attributes. Writes and
rotation are serialized; tag replacement is atomic. `Debugf` is opt-in.
Default retention is 16 MiB plus three backups per logger. Private log files and
rotation failures are explicit; logs are operational evidence, not durable game
state or a substitute for transaction results.

### netx

[Source/API](server-reference-infrastructure.md#serverinternalnetx).
`Serve` owns listener cancellation, handler drain, total/per-IP admission,
establishment rate limiting and bounded TLS handshakes. `Tune` enables keepalive
and disables Nagle. `Frames.Push` reconstructs TCP frames from fragments and
coalesced reads; invalid lengths fail. `RateLimiter` is a bounded token bucket
with idle-entry reclamation. `Writer` preserves ordered output with packet/byte
caps and deadlines. `Abort` interrupts pending TLS/TCP IO before closing on
cancellation/error; normal close allows a bounded TLS close notification.

This package is distinct from client `internal/netx`; direction-specific framing
and encryption wrappers must not be confused.

### packet

[Source/API](server-reference-infrastructure.md#serverinternalpacket).
Server opcode aliases and explicit frame assembly. `New(op)` reserves the length
and writes the opcode. Width methods append little-endian fields; `Count` writes
one byte and rejects quantities above 255. `FixedString` rejects overflow instead
of silently accepting an altered value. `Finish` reports errors and enforces
u16 frame size; `MustFinish` is for bounded known layouts. Reader helpers share
`internal/proto` bounds/text behavior. [Opcode catalog](server-opcodes.md).

### password

[Source/API](server-reference-infrastructure.md#serverinternalpassword).
`HashContext`/`VerifyContext` implement versioned Argon2id with random salts,
strict encoded-parameter validation and bounded concurrent work. Defaults are
19 MiB, two iterations and one lane, with two workers/process. Failed or unknown
credentials perform dummy work to reduce enumeration timing. Legacy plaintext
is accepted only through the explicit upgrade path on successful login; new
operator accounts are hashed immediately. Hashing precedes the guarded write
transaction, avoiding a long KDF while holding the SQLite writer.

### scheduler

[Source/API](server-reference-infrastructure.md#serverinternalscheduler).
`New(start,jobs)` validates unique job names, positive intervals and callbacks.
`Step` runs due jobs serially with supplied time, records lateness/duration and
advances past missed intervals. `Run(ctx)` owns the ticker. Cancellation is checked
between callbacks; a running callback must return before shutdown completes.
This is a world maintenance timeline, not a general unbounded worker pool.

### session

[Source/API](server-reference-infrastructure.md#serverinternalsession).
This is the **server** handoff package, not the client's session connection API.
`Handoff` contains identity, AES material, user/session ID and credential version.
`Save` replaces a ticket only if its credential version is still current.
`Consume` validates identity/version/expiry and deletes in one write transaction;
invalid identities cannot burn another user's ticket. Random identities are 16
bytes; maximum ticket age is ten minutes. Old `runtime/sessions/*.bin` files are
not authentication authority.

### telemetry

[Source/API](server-reference-infrastructure.md#serverinternaltelemetry).
Process-wide fixed-name registry, capped at 128 metrics. `Number` supports atomic
counters/gauges; `Histogram` records cumulative duration buckets and sums.
`WritePrometheus` sorts output deterministically. Names are validated; no user,
account, item or source-IP labels create unbounded cardinality. Producers own
metric meaning; scraping must not perform checkpoint/truncation work on live WAL.

## ai

[Source/API](server-reference-infrastructure.md#serverai).
`server/ai` embeds catalog/core/definition Lua source and can load an explicitly
selected editable catalog. `Program` validates modules/catalog entries and caches
compiled units; VMs/actors carry execution state. The host interface exposes
named world primitives rather than arbitrary OS/network/database access. World
integration holds the appropriate shard ownership and queues bounded structural
commands. Script failures are visible and fail the actor; required unavailable
primitives are errors, not a silent encounter bypass.

See [Lua host API](../../server/ai/HOST.md) and
[catalog maintenance](../../server/ai/README.md) for every operation and event.
Gameplay AI IDs retain their content meaning; Go does not silently substitute a
legacy decision loop after a selected catalog fails.

## db

[Complete API/file inventory](server-reference-storage.md),
[transaction/storage guide](server-storage.md).

| Files | Responsibility |
|---|---|
| `db.go`, `errors.go` | Runtime initialization, one shared connection/process, contexts, metrics, transaction failure classification and clean close |
| `migrations.go` | Additive checksummed ledger independent of legacy user_version |
| `ownership.go` | World OS lease, durable exact-session account ownership, bounded failed-release retries |
| `account.go`, `accounts_admin.go`, `faction.go` | Authentication/logout, role/status policy, password-version mutation and account faction |
| `defs.go` | Read-only game definitions and cached typed lookup catalog |
| `chars.go` | Character lifecycle, guarded creation/starter inventory, selection/saves and deletion/restore/name constraints |
| `items.go`, `gameplay.go` | Item state and composable before/after guarded gameplay transaction |
| `warehouse.go`, `guildbank.go` | Shared/personal storage exchanges including fees and both inventories |
| `mall.go` | Points, purchases, gifts, gift claiming and audit logs |
| `quests.go`, `skills.go` | Active/finished quests, atomic reward state, skills, buff/bar/cooldown snapshots and guarded learning cost |
| `guilds.go`, `guild_creation.go` | Guild membership/permissions/mirrors, master fee, creation/disband and related atomic mutations |
| `social.go`, `keeps.go`, `gm.go` | Friends/blocks, persistent keep ownership and server-authorized GM persistence |

## world

[Complete API/file inventory](server-reference-world.md).
This package contains both application adapters and the authoritative game domain;
there is no parallel unused service framework. The following map identifies the
actual seams. The inventory lists every remaining file and symbol.

| Subsystem | Principal files | State / contract |
|---|---|---|
| Realm and connection | `realm.go`, `session.go`, `dispatch.go`, `players.go` | Registry lifetime, exact-account reservation, protocol phase, session serialization and published player snapshots |
| Timeline and instances | `sweeps.go`, `instance.go`, `partyzone.go`, `worldcal.go` | Owned periodic work, map instance lifecycle and original calendar |
| Maps, NPCs and views | `svmap.go`, `entities.go`, `view.go`, `visibility.go`, `mob_navigation.go` | Server content, spatial membership, visibility/concealment and authoritative navigation |
| Movement and physics | `movement.go`, `player_movement.go`, `attack_movement.go`, `fall_damage.go`, `ladders.go` | Validated movement/landing/ladder intent, speed and attack approach constraints |
| Character and resources | `character.go`, `rules.go`, `rebirth.go`, `pvpdeath.go`, `elementstate.go` | Character normalization, original progression/death policies, elements and live pools |
| Combat | `combat.go`, `combatformula.go`, `combat_cadence.go`, `pvp.go`, `pvpskill.go` | Hit/chance/damage/cadence, hostile-target policy and authority |
| Skills/effects | `skills.go`, `skillcast.go`, `skillattack.go`, `skillonce.go`, `skill_dash.go`, `skill_range.go`, `abilities.go`, `mobeffects.go`, `playereffects.go`, `target_effects.go` | Cast scheduling, resource cost, ranges, periodic effects and published target snapshots |
| Inventory/economy | `inventory_operation.go`, `inventory.go`, `mapitems.go`, `drops.go`, `trade.go`, `warehouse.go`, `guildbank.go`, `itemmall.go` | Staged mutations, exclusive loot claims and durable cross-inventory commits |
| Crafting/metadata | `crafting.go`, `enchant_rules.go`, `forge_quote.go`, `recreation_runes.go`, `itemoption.go`, `item_skills.go`, `oj_limits.go`, `cloak_fits.go` | Original cost/chance/target rules, deferred effects and metadata catalogs |
| Quests | `quests.go`, `npcquest.go`, `quest_map.go` | Eligibility, counters, choices, atomic rewards/completion and quest-NPC metadata |
| Social/groups | `chat.go`, `social.go`, `friendly.go`, `party.go`, `raid.go`, `duel.go` | Validated text, friends/blocks, social ownership and group state |
| Guilds/ranking | `guild.go`, `guildduel.go`, `grb.go` | Membership/permissions, guild creation consensus and ranking/duel rules |
| Relics and world PvP | `relics.go`, `relic_config.go`, `relic_portals.go`, `linked_bosses.go`, `killcount.go`, `pvpreward.go` | Persistent ownership, linked mechanics, blessing counters and PvP reward eligibility |
| AI | `mob_ai.go`, `mob_ai_speech.go` | Host primitives, actor decisions, bounded summons/commands and mob speech |
| Vehicle and GM | `vehicle.go`, `gm.go`, `gm_map.go` | Authoritative mounts/seats and role-checked administrative actions |
| Data/wire assembly | `gameini.go`, `protocol.go` | Validated content/rule loading and field-by-field response construction |

## Shared root packages used by the server

These are still in the same Go module. Their shared definitions avoid divergent
client/server math or wire layouts; server authority does not move to the client.

| Package | Shared role |
|---|---|
| `internal/attackspeed`, `internal/movespeed` | Original speed indices, duration/tier normalization |
| `internal/authwire` | Bounded UTF-8 long-password extension, exact length validation |
| `internal/proto` | Explicit packet readers/writers, row types and opcode names |
| `internal/transport` | TLS listener/client trust policy and portable network configuration |
| `internal/textwire` | UTF-8 and legacy ANSI-compatible text conversion |
| `internal/itemstack`, `internal/itemtype` | Byte stack limits/revision and item-kind classification |
| `internal/element`, `internal/forgecost` | Element matrix and original crafting cost rules |
| `internal/skillrange`, `internal/skilltiming`, `internal/pvprank` | Skill reach/timing and PvP thresholds/reward metadata |
| `internal/gm` | GX role/extension definitions |
| `internal/mapgeometry`, `internal/mapnav` | Collision/navigation products and authoritative queries |
| `internal/data/archive`, `internal/data/world`, `internal/data/sdata` | Offline import/content readers; no proprietary data is embedded by documenting them |

## Tools and verification

`server/tools/sqlite.py` initializes/queries or delegates account creation to
`gx_console`. `snapshot_sqlite.py` performs a consistent backup and manifest;
`runtime_lock.py` interoperates with initialization locks. Packaging preserves
existing runtimes and bundles operational documentation. Content migration scripts
are explicit operations, not startup resets.

Use [building/testing](building-and-testing.md) for exact commands. Unit tests,
disposable-DB protocol integration, race checks, fuzzing and native start/stop
cover different invariants. Generated API/protocol pages are validated by
`python3 tools/generate_server_reference.py --check`; a complete filename/symbol
inventory does not replace game-rule or operational validation.
