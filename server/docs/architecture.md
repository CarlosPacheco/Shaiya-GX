# Architecture

GX Server is a modular Go application in the GX root module. SQLite is the
primary production database. There is no separate database service or server
repository. The GUI (`gx_server`) and console control `gx_login` and `gx_world`.

## Ownership and lifecycle

One world process owns one realm and holds an OS-backed lease on its database.
The realm owns map shards, registries and instance identifiers. Each shard owns
its actors, Lua execution and ground items; a session owns its character and
serial command processing. Social operations retain explicit domain locks.
Cross-session trade uses ordered locking. SQLite work does not run while the
account registry lock or ground-item claim lock is held.

A cancellable world runtime owns one deterministic scheduler for periodic
simulation and maintenance. Jobs run serially, skip overdue catch-up bursts and
record duration and delay. Socket handlers remain concurrent. This keeps the
existing game timings without adding a goroutine per timer or an actor framework.
Shutdown closes admission, interrupts sockets, drains handlers and persistence,
stops the runtime, then optimizes and closes SQLite.

## Commands and persistence

Framing and phase validation precede command dispatch. Inventory application
operations stage private state, validate the expected persisted state, commit a
single transaction and then publish packets and live state. Trade, rewards,
shops, crafting, warehouses, guild banks, skills and guild creation use atomic
storage operations. Conflicting/replayed requests cannot reuse the same funds
or item. Ground loot has an exclusive claim with rollback on failed persistence.

Storage is confined to `internal/db`. WAL, FULL synchronization, IMMEDIATE write
transactions and one bounded connection per process deliberately serialize
writers. Checksummed additive migrations preserve the distributed seed and old
EP4.5 tables. Runtime copies, credentials, certificates and logs stay outside Git.
Unusable storage cancels the world rather than continuing unsaved simulation.

Network queues and gameplay mailboxes are bounded. Economic rewards accumulate
with acknowledgement after commit; bounded item delivery falls back to ground
loot. Unresponsive combat recipients disconnect on mailbox overflow. AI commands
have explicit capacity and actor limits.

## Operations

Authentication uses versioned Argon2id hashes, bounded hashing workers, atomic
single-use handoffs and credential revocation. TLS 1.3 is required for remote
listeners unless the operator explicitly opts into insecure development mode.
Loopback diagnostics provide health and bounded-cardinality Prometheus metrics;
shutdown and optional profiling require a private bearer token. Controllers use
this same authenticated shutdown on Linux and Windows before force-stop fallback.

See [modernization](modernization.md), [SQLite operations](sqlite.md) and
[server operations](../../docs/server.md) for configuration and recovery.
