# Shaiya GX gx-v0.1.11

Changes since **gx-v0.1.10**. Update the client and server together.

## Combat and original presentation

- Restore looping Wind Spin skills, early area-combo release and the authoritative Stinger dash, including original target-size reach allowances.
- Correct prepared-cast range handling and keep stationary casts active across idle-position synchronization. Restore Blizzard area effects.
- Correct authoritative HP/MP/SP pools, weapon and consumable presentation, and stack-count labels so non-stackable items no longer display quantities.
- Correct meteor rotation, placement and ground contact without adding render passes. Preserve fixed spawn effects and freeze corpses at their last rendered death position.
- Restrict the temporary combat-ready stance to basic attacks; spellcasts and buffs no longer renew it. Stabilize cloak lighting.
- Start ordinary group-respawn timers when the first corpse is removed, preserving the original five-second corpse interval. Restore authored monster scales and quest-drop indexing.
- Report skill recharge/cadence refusals and make idle monsters retaliate on a genuine missed attack without inventing damage.

## Monster AI

- Make modern embedded Lua the default authoritative monster-AI policy runtime, with 46 documented definitions, bounded execution and real server movement/combat primitives.
- Ship editable `Server/ai/` policies with embedded defaults. Remove the legacy AI/Lua catalogs and the separate external checker.
- Preserve original special-case behavior and lifecycle rules. The audit and regression catalog do not constitute complete encounter-by-encounter parity certification.

## Persistence and server operation

- Commit trades, quest rewards and economy/inventory operations atomically before publishing successful results. Add rollback, replay, concurrent-pickup, disconnect and crash-recovery coverage.
- Modernize realm/map ownership, cancellable scheduling, bounded mailboxes and protocol admission while keeping the original gameplay packet layouts.
- Keep SQLite as the primary embedded production database, with WAL/FULL durability, checksummed additive migrations, ownership leases, indexes and fail-stop handling of unusable storage.
- Add Argon2id password hashing, credential/session revocation and coordinated client/server support for passwords up to 128 UTF-8 bytes. Existing password records upgrade on successful login.
- Require TLS 1.3 for remote connections by default; add certificate provisioning and client trust configuration. Loopback development settings remain available.
- Add rotating structured logs, loopback health/Prometheus metrics and authenticated, executable-scoped shutdown on Linux and Windows.
- **GX Server is now terminal-only.** Remove the graphical controller, its icon and Fyne dependencies. Use `gx_console` with `start server` / `shutdown server`; closing the console leaves login/world running.

## Rendering and development

- Reuse actor animation matrices, cache immutable rest transforms and share palettes only when inverse-bind arrays are identical.
- Reject safely off-screen actors before preparing their exact poses. Preserve independent reflection/shadow passes and current name anchors; uncertain data and cloth keep exact culling.
- Share scene setup across consecutive map effects while preserving transparency order.
- Focused synthetic measurements showed 18–22% lower CPU submission time for pose reuse in 200/400-actor scenes, plus a separately measured 3.7–7.3% reduction for pre-culling in the 200-actor fixture. These are separate comparisons, not additive gains or presented-FPS promises; other scenes may show little change.
- Add the reproducible CPU/GPU measurement harness and expand the common Wiki with architecture, all server packages and the complete protocol/codec reference.
- Update Go to 1.27.4 and validate native Linux/Windows builds, server integration, startup/isolation, race checks and bounded fuzz probes in CI. Fix physics provisioning and an incorrect diagonal-distance assertion in CI.

## Downloads and upgrade

GitHub releases contain native client/server binaries, fonts, server maps/physics,
AI policies, documentation and the intentional development seed. They do not
include original client game artwork/audio/maps. The maintainer's separately
prepared complete folder includes the local client `data/` directory.

Preserve your existing `Server/runtime/`, `.local/`, client settings and private
TLS material. Do not replace a personal runtime with the distributed seed. The
server applies its additive migrations on startup; back up first and keep both
endpoints on this release. See the [upgrade guide](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/releases/gx-v0.1.11-upgrade.md).
