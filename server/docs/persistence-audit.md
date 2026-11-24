# Persistence and concurrency audit

SQLite is GX's primary production database. The original Linux audit of
95234a8 / gx-v0.1.10 reproduced three gameplay persistence defects. All three
are now corrected in source, with ordinary integration regression tests.
The gx-v0.1.10 release archives predate these fixes; updating this document does
not replace an existing executable or publish a new release.

## Corrections

### Atomic player exchanges

Previously, a failed source-item deletion could still grant the recipient a
copy. Gold was also saved for only one participant before the recipient's
mailbox applied its credit. Both defects came from separate gameplay writes.

A trade now plans both inventories and balances on private copies and commits
them in one SQLite transaction through `server/internal/db`. Expected inventory
and money are checked under the transaction's write lock. Failed deletions,
recipient inserts or character updates roll back the complete exchange.
Unchanged inventory rows and their original metadata are preserved.

The two sessions' state locks protect planning, commit and publication. The
lower character ID coordinates the commit and retries on a later tick when the
partner is busy, without waiting for the partner lock while holding its own.
Both owners receive their committed state before success is sent. Logout or a
later position save therefore cannot overwrite the credit with stale money.
The active trade is claimed once, so captured duplicate confirmations cannot
replay even a money-only trade. Disconnect invalidates pending requests.
Overflow beyond the existing gold cap cancels the exchange rather than silently
losing currency.

### Atomic quest rewards

Previously, gold could commit while the quest completion update failed,
leaving the reward paid and the quest active in storage.

Required-item consumption, reward placement, gold, experience, level/stat/skill
points, HP/MP/SP and quest completion now share one transaction. Item checks and
reward-choice prompts do not consume hand-in items. Live state, reward packets
and completion success are published only after commit. A write failure retains
the original quest and items; a full bag permits a later retry without losing
the reward. Existing stacking, reward quantities and experience formulas apply.

A guarded completion update requires an active unfinished quest row. Repeatable
quests use a guarded deletion in that same transaction. A stale session object
cannot replay either kind of reward; a newly accepted repeatable quest can still
be completed normally.

## Regression checks

The original three diagnostic tests are now part of the normal `integration`
suite; the extra `persistenceaudit` build tag is no longer required.

```sh
. tools/go-env.sh
go test -tags integration ./server/internal/world -run '^(TestAudit|TestTrade|TestQuestRollback|TestQuestFullBag|TestRepeatableQuest)' -count=1 -v
go test -tags integration ./server/cmd/gx_world -run '^(TestTradeDurableBeforeAckAndDisconnectAgainstSQLite|TestQuestWriteFailureRetryAndDurableAckAgainstSQLite)$' -count=1 -timeout=120s -v
go test -tags integration ./server/... -count=1
```

Coverage includes:

- Source deletion, receiver insertion and second-character save failures with
  unchanged live and stored inventories and gold on both sides.
- Partial stacks, full inventories, gold cap overflow, duplicate confirmations,
  a busy partner and cancellation after disconnect.
- Quest completion, required-item deletion, reward insertion and character save
  failures, with retry, level-up, full bags, durable replay guards and repeatables.
- Real encrypted trade requests and simultaneous confirmations; both balances
  and item counts are durable on acknowledgement, before reading recipient
  updates. Immediate logout, relog and a pending-trade disconnect preserve them.
- Real quest hand-in with a rejected character save: no completion is sent, the
  original quest remains active, retry commits all rewards before acknowledgement,
  and duplicate hand-in/relog preserves the exact result.

## Previously skipped protocol coverage

Five old protocol tests depended on `Interloper`, `Valiant` or `Arcanist` being
present in the development seed. Combat, item mall, party, raid and visibility
now create their own disposable accounts, faction and characters. Combat uses a
real mob definition in a private test entity if no eligible seed spawn exists.
These five tests now execute instead of silently skipping when seed names change.

## Engine and bounded load checks

The audit also verified four independent Go/modernc processes completing 800
atomic two-row transfers with conserved balances and no busy failures. Killing
a writer before commit rolls both rows back; killing it after commit retains
both. A live backup of a disposable full-seed copy restored all 96 tables with
matching row counts, valid integrity and conserved balances during concurrent
WAL transfers. The live fixture remained writable afterward.

The 16-client protocol workload creates a two-player party, completes 80
acknowledged priced purchases, then cuts eight additional purchases in flight.
Charges, inventory quantities and point-log counts agree after clean logout,
abrupt socket close and encrypted relog. The priced product exists only in test
memory; the distributed mall remains unchanged.

```sh
. tools/go-env.sh
go test -tags integration ./server/internal/db -run '^TestSQLiteConcurrentProcessesAndCrashRecovery$' -count=1 -v
go test -race -tags integration ./server/cmd/gx_world -run '^(TestConcurrentClientsPurchaseLogoutAndRelog|TestTradeDurableBeforeAckAndDisconnectAgainstSQLite)$' -count=1 -timeout=300s -v
python3 -m unittest discover -s server/tools -p 'test_*.py'
```

All fixtures use disposable databases and isolated services. No personal runtime
or distributed seed is refreshed by these checks. They are bounded regression
workloads, not a maximum-player benchmark, long soak, disk-full recovery,
16 rendered clients or a level-1-to-60 playthrough. Passing SQLite recovery
checks does not certify every other gameplay subsystem as crash-safe.
For deployment and consistent backups, see [SQLite](sqlite.md) and
[server operations](../../docs/server.md).

For the opt-in earned early-game sequence with two new characters, a real world
restart and dungeon party resurrection, see the [progression journey](party-progression.md).
Its explicit position-only portal checkpoint does not validate the complete
overland route.

## Runtime write audit and 1,000-client workload

The runtime audit examined SQLite's writer path, imported identity triggers,
authentication tickets, character exit and the frequently written inventory and
quick-bar tables. A schema-only inspection and integrity/duplicate checks of the
approved seed found valid integrity and no duplicate live character or inventory
slots in that snapshot. Authoritative gameplay catalogs, archived tables and the
distributed seed were retained. Existing login and economic logs remain available
for operator-defined retention rather than being pruned during a schema upgrade.

A production-sized probe is described in the [SQLite guide](sqlite.md). The
original 1,000-connection probe hit database context deadlines. CPU profiling
identified the full-table `MAX(RowID)` in imported insert triggers as a primary
cost. Migrations add indexed access without changing legacy values or
regenerating existing data. Reader isolation, set-based quick-bar updates and
group commit remove secondary contention while preserving full WAL sync.

The probe is disk-backed when `TMPDIR` points to the target filesystem. It
measures encrypted World sessions with a mixed, closed-loop write workload.
Credential login rate, storage capacity on other hosts and multi-hour soak
testing require separate measurements.

On the Linux development host with the fixture on the local disk, the
1,000-session probe completed 96,000 mixed operations in 39.81 seconds, plus
1,000 clean logouts and sampled relogs. It recorded zero failed transactions
and zero outgoing queue overflows. SQLite integrity and foreign-key checks
passed, and all 1,000 accounts had the expected balances, sale logs, items and
quick-slot rows. The writer grouped 104,447 operations, including background
and exit work, into 3,302 durable commits.

| Wire operation under the sustained burst | Samples | p50 | p95 | p99 |
|---|---:|---:|---:|---:|
| Purchase | 32,000 | 383 ms | 701 ms | 2.42 s |
| Inventory move | 32,000 | 333 ms | 730 ms | 2.17 s |
| Quick-bar save followed by ping | 32,000 | 376 ms | 765 ms | 1.45 s |
| Character logout | 1,000 | 550 ms | 1.03 s | 1.05 s |
