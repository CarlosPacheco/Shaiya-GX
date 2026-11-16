# Persistence and concurrency audit — 2026-11-15

SQLite remains GX's primary production database. This audit separates engine
recovery from the gameplay operations that must commit together. The tested
baseline is gx-v0.1.10 / source 95234a8, on Linux with local game data.
This change adds tests and records existing behavior; it does not fix the
transaction boundaries described below or certify the server for production.

## Results

| Probe | Result | Scope |
| --- | --- | --- |
| Server integration suite | Passed | Disposable SQLite fixture; package suite passed, but five legacy protocol tests were skipped (see coverage limits) |
| Two-client N2 | Passed | Real local assets/OpenGL, creation, native controls, loot, skills, guild, paid portals, duel, movement, attack cadence and ladders |
| Concurrent protocol clients | Passed | 16 encrypted world connections, including a two-player party; 80 acknowledged purchases plus eight in-flight purchase disconnects; charges/items agree after clean logout, abrupt socket close and relog |
| World concurrency race detector | Passed | The same 16-client party/purchase/disconnect/relog workload completed with `-race`, with no race reports |
| SQLite process contention | Passed | Four independent Go/modernc processes; 800 atomic two-row transfers, conserved balances and no busy failures |
| SQLite process termination | Passed | Killing a writer before commit rolls back both rows; killing it after commit retains both |
| Live backup and restoration | Passed | A private full-seed copy, 96 tables, concurrent WAL transfers, backup API, manifest/row counts and integrity check; restored balances conserved |
| Trade write failure | **Failed** | A rejected source deletion still grants the receiver an item |
| Trade durable gold | **Failed** | A successful exchange saves only one gold balance before the partner mailbox runs |
| Quest completion failure | **Failed** | Gold can commit while the active quest remains unfinished |

The private fixtures and approved distributed seed are distinct. These probes do
not change a personal runtime, account inventory, installation or seed snapshot.
The packaged development/test accounts remain fixtures, independently of SQLite's
role as the production database.

## Coverage limits found during the audit

Five legacy protocol tests skip because the current approved seed has no
`Interloper`, `Valiant` or `Arcanist`: `TestCombatAgainstSQLite`,
`TestItemMallAgainstSQLite`, `TestPartyAgainstSQLite`, `TestRaidAgainstSQLite`
and `TestViewAgainstSQLite`. Their package returning success does not establish
those scenarios. Migrate those tests to explicit disposable fixtures before
claiming their coverage. The new client-load probe creates its own 16 accounts,
characters and priced product and independently verifies party formation,
purchases and reconnect persistence. The distributed mall is intentionally free;
the priced fixture exists only in test memory and does not change its prices.

## Reproducible failures

Run the opt-in fault-injection suite from the repository root:

```sh
. tools/go-env.sh
go test -tags integration,persistenceaudit ./server/internal/world -run '^TestAudit' -count=1 -v
```

**This command currently fails three tests.** The additional tag explicitly
separates the known failing diagnostic probes from the regular regression suite.
Their assertions require the correct invariants; they do not pass by expecting
corruption. Promote each probe to the ordinary integration suite when its defect
is fixed. All SQL triggers below exist only in disposable fixtures.

### Trade: item conservation

`TestAuditTradeWriteFailureConservesItems` injects an SQLite `RAISE(ABORT)` trigger
on the offered item's source deletion. The source keeps the original, but
[`transferOffer`](../internal/world/trade.go) ignores that failure and continues
the receiver grant: the combined quantity changes from **1 to 2**.

Required correction: plan both inventory changes without mutating live state,
persist all debits/credits in one transaction with guarded ownership/quantity
checks, and publish the inventory/packets only after commit. A receiver insert
failure must also roll back the source debit. Include partial stacks and full
inventories in regression coverage.

### Trade: acknowledged gold must already be durable

`TestAuditTradeMoneyDurableBeforePartnerMailbox` offers 100 gold between two
characters holding 1,000 each. Immediately after `finalizeTrade` reports success,
the stored balances are **900 and 1,000**, rather than 900 and 1,100. The other
balance is deferred through `QueueTradeRefresh` / `applyTradeRefresh`.

The test inspects this deterministic crash boundary; it does not claim to kill
an actual world process at that exact instruction. An exit there has no durable
recipient credit to recover. The separate process-termination test verifies that
SQLite preserves committed transactions and rolls back uncommitted ones.

Required correction: include **both** money balances with the inventory transfer
transaction. Coordinate session ownership, disconnects, duplicate confirmation
and pending refreshes so a later session save cannot overwrite committed money.
A delayed mailbox may refresh presentation, but must not be responsible for the
first durable credit.

### Quests: reward and completion must be inseparable

`TestAuditQuestFinishFailureCannotRepeatReward` uses a synthetic 100-gold reward
on a disposable quest-1 row and rejects its completion update with a trigger.
[`awardQuest`](../internal/world/quests.go) leaves **1,100 gold with Finish=0**:
the reward is durable while the quest remains active in storage. This is a
replay risk on reload, rather than a demonstrated full network replay exploit.

Required correction: persist required-item consumption, selected reward,
experience/level/stat changes and quest completion together. On any failure,
retain the original live state and do not send completion success. Test duplicate
hand-in, reward selection, full bags, disconnect and failed writes as well as
ordinary completion.

## Repeatable passing checks

```sh
. tools/go-env.sh
go test -tags integration ./server/... -count=1
go test -tags integration ./server/internal/db -run '^TestSQLiteConcurrentProcessesAndCrashRecovery$' -count=1 -v
go test -tags integration ./server/cmd/gx_world -run '^TestConcurrentClientsPurchaseLogoutAndRelog$' -count=1 -timeout=120s -v
go test -race -tags integration ./server/cmd/gx_world -run '^TestConcurrentClientsPurchaseLogoutAndRelog$' -count=1 -timeout=300s -v
python3 -m unittest discover -s server/tools -p 'test_*.py'
python3 tools/n2_smoke.py
python3 tools/n2_mounts.py
```

The process/transaction and client-load probes print measured timings. They are
bounded smoke workloads, not maximum player capacity, a long soak, disk-full
recovery or a representative combat/world simulation. The 16-client workload
uses real protocol handlers with an in-process world and test map; it does not
measure 16 rendered clients, real network latency or multi-map combat. Host and
filesystem effects matter; do not turn these measurements into a player limit.
Ordinary N2 coverage is not a complete level-1-to-60 progression playthrough.

For actual operations and consistent backups, see [SQLite](sqlite.md) and
[server operations](../../docs/server.md). Fix the transaction-boundary failures
before treating normal-path test results as crash-safe gameplay persistence.
