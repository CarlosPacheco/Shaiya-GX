# Server modernization

The modernization preserves original game behavior and packet layouts while
replacing infrastructure and unsafe implementation patterns. Go and SQLite
remain the implementation and production storage. Shared test accounts and the
published development seed are intentional development fixtures.

## Delivered foundation

- Login/world share admission and lifecycle management: 200/500 total sockets,
  32 per source IP, bounded TLS handshake, SIGINT/SIGTERM cancellation, socket
  closure and handler drain before SQLite closes. Linux controllers request
  graceful termination, with a 30-second force-stop fallback. Detached Windows
  controllers still force-stop: do not claim graceful Windows shutdown.
- Login has a 30-second total protocol deadline, 5-second write deadlines and
  bounded token buckets: 30 attempts/IP/minute and 10/account/minute. The buckets
  refill continuously, cap memory and reclaim idle entries. Limits currently
  live in source; shared NATs may need tuning after measurement.
- Passwords upgrade from the legacy plaintext field to salted Argon2id on a
  successful login. Parameters are versioned (19 MiB, two iterations, one lane),
  with at most two simultaneous hashes per process. Hashing precedes the write
  transaction; the transaction verifies that the stored credential has not
  changed. The seed is not rewritten. Dormant accounts remain legacy until
  login. Account creation tools still create legacy development accounts that
  upgrade at first login. Do not downgrade to a server that only compares Pw.
- Handoffs use GX_SessionHandoff in the configured mutable runtime database.
  Validation and consumption share an IMMEDIATE transaction, so a token can
  succeed once across concurrent processes. Invalid attempts cannot consume it.
  Old runtime/sessions/*.bin files are no longer trusted. Upgrading requires a
  fresh login; mixed old/new login and world binaries are unsupported.
- Logout is session-scoped and idempotent: an old disconnect cannot overwrite
  a new login, and duplicate logout cannot credit online points twice.
- World input has per-connection budgets of 512 frames and 1 MiB per second
  (token buckets). Oversized traffic disconnects before dispatching more work.
- World output is ordered and asynchronous, with 256 queued packets / 1 MiB per
  connection including in-flight output. A slow receiver is disconnected on
  overflow; authoritative packets are never silently dropped. Writes and final
  queue drain have deadlines. This bounds network output, not gameplay mailboxes.
- Runtime initialization uses OS process locks shared with the Python SQLite
  tool; process exit releases the lock without leaving a stale lock directory.
- Character creation, starter inventory and slot/name guards share one write
  transaction. Item insertion failure rolls back the character and earlier items.
  Restoring a deleted character cannot overwrite an occupied slot.
- Login queries use a runtime UserID index; EXPLAIN QUERY PLAN verifies an indexed
  search rather than the previous full account-table scan.
- SQL transaction helpers roll back on panic. WAL, synchronous=FULL, foreign
  keys and IMMEDIATE transactions are retained. One connection per process is
  deliberate; increasing writers blindly would add contention. Graceful close
  runs PRAGMA optimize before closing the connection.
- Per-packet opcode logs require RAPTURE_DEBUG=1.

## TLS configuration

Both services can wrap existing game packets in TLS 1.3, using Go's standard
TLS implementation. The legacy inner encryption remains for packet compatibility.
Configure both login and world with:

```
RAPTURE_TLS_CERT=/private/gx-server-cert.pem
RAPTURE_TLS_KEY=/private/gx-server-key.pem
```

The certificate must cover the identity clients validate. On the client use
GX_TLS=1 for system trust, or GX_TLS_CA=/private/gx-ca.pem for a private CA.
GX_TLS_SERVER_NAME specifies the certificate DNS identity when the world address
is an IP or differs from that identity. These settings apply to both connections.
There is no verification bypass or fallback after TLS failure. Never distribute
the server private key. Plain TCP is still the default for local development;
TLS is implemented but not silently enabled on existing installations.

## Remaining architecture work

This foundation is not a completed rewrite or full security certification.

1. Make world/map instances explicit owners of simulation state. Registry maps,
   session state, actor AI and cross-session trade locks currently coexist.
   Inventory the ownership and lock order before moving each subsystem.
2. Separate protocol decoding from application actions, domain validation and
   storage. Start with character lifecycle and inventory, then commerce/quests,
   combat and social systems. Preserve existing real-wire regression tests.
3. Audit every economy operation for atomicity, optimistic preconditions and
   publish-after-commit. Trade and quest rewards already have these safeguards;
   do not replace their tested behavior with independent writes.
4. Bound gameplay mailboxes with explicit overload semantics. Reward/damage
   queues cannot simply drop entries or disconnect after silently losing rewards.
5. Replace global background sweeps with cancellable instance runtimes and
   deterministic clocks. Add tick budgets and latency measurements before
   changing simulation scheduling.
6. Add structured logs with rotation, metrics and loopback-only diagnostics;
   measure tick delay, writer backlog, database wait/transaction latency, WAL
   growth, admission failures and save failures. Avoid account/IP metric labels.
7. Version schema migrations explicitly, add backup/restore drills and failure
   injection for disk-full, interrupted startup and shutdown. Consider separate
   bounded read connections only when measured contention justifies them.
8. Finish platform lifecycle support, certificate provisioning and client-facing
   configuration before making secure transport the standard remote mode.
9. Add authentication timing normalization and password management tooling that
   creates hashes directly; replace the legacy password-length constraint only
   with coordinated client changes.

For each subsystem, require behavior tests, concurrent/failure tests, measured
resource use and removal of its obsolete path. Avoid a parallel unused framework.
No blanket maximum-player or latency claims follow from bounded fixture tests.

## Validation

Run tools/check.sh on affected packages; integration tests use private runtime
copies and isolated ports. Race tests cover credentials, admission, output and
session consumption. Fuzz Frames with Go fuzzing. Run govulncheck against headless
server packages and client transport. Existing engine crash-recovery and backup
checks remain in the integration suite (see persistence-audit.md).

References: [Argon2 implementation](https://pkg.go.dev/golang.org/x/crypto/argon2),
[SQLite planner maintenance](https://sqlite.org/pragma.html#pragma_optimize),
[Go vulnerability checking](https://go.dev/doc/security/vuln/).

## Initial Linux validation (2026-11-16)

Affected vet/unit checks and the complete server integration suite pass. The
headless world protocol suite completed in 71.4 seconds and world integration
in 22.2 seconds. Race checks cover the infrastructure and database, plus real
concurrent purchases, trades and quest rewards (137 seconds for that workload).
Frame fuzzing exercised approximately 196,000 inputs without failure. Four
Python storage-tool tests pass. Headless/transport govulncheck reported no
reachable vulnerabilities; one advisory belongs to a required module without
an affected imported package in this scan. This is not a client-wide scan.

Native Linux binaries started with isolated ports and a disposable database,
terminated cleanly on SIGTERM, retained database integrity and left the seed
unchanged. The login query plan uses the new index. Client/login/world/console/GUI
Linux binaries were built; existing Desktop executables were updated with eight
protected data/configuration hashes unchanged. TLS verification and hostname
rejection are tested with disposable certificates; TLS was not activated in the
user's local configuration. No Windows execution or maximum-player benchmark
was performed. The other terminal's profiling files are excluded from the change.
