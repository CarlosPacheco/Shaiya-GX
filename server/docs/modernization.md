# Server modernization

The server modernization replaces unsafe infrastructure and persistence paths
while preserving original game rules and binary gameplay packets. Go and SQLite
remain the production architecture. Shared development accounts and the shipped
seed are intentional fixtures. The implementation is a modular monolith, with
explicit state ownership rather than extra services or a parallel framework.

## Delivered changes

- Realm/map ownership, cancellable serial world scheduling, deterministic
  scheduler tests and measured job duration/delay replace detached sweep loops.
- Phase-aware input dispatch rejects commands outside character selection,
  loading and active play. Partial-frame traffic cannot repeatedly drive ticks.
- Atomic application operations cover inventory moves, grants/drops, shops,
  crafting, warehouse, guild bank, skill learning and guild creation. Trade and
  quest reward guards remain. State and packets publish after durable commit;
  stale snapshots, concurrent pickups and SQL failures have regression tests.
- Ground-item claims prevent double credit. Reward accumulators acknowledge
  after commit; item grants, combat mailboxes and AI commands have explicit bounds
  and overload behavior. Quest/quickbar/buff snapshots use atomic batches.
- SQLite has checksummed additive migrations, account and warehouse indexes,
  world ownership leases, crash-released startup locks, WAL/FULL synchronization,
  guarded transactions and fail-stop handling of unusable storage. A single
  connection per process bounds contention; no speculative writer pool is added.
- Passwords use salted, versioned Argon2id (19 MiB, two iterations, one lane),
  with two hashing workers per process and dummy work for failed authentication.
  Legacy passwords upgrade on successful login without rewriting the seed.
  New account/password commands write hashes directly. Credential changes revoke
  handoffs and active sessions. Role/status changes are rechecked during play.
- Single-use handoffs and active account ownership are transactional SQLite data.
  Logout is idempotent and scoped to the exact session.
- TLS 1.3 is the default remote transport. Certificate provisioning and portable
  client trust configuration are supplied. Local plaintext remains available for
  development; remote plaintext requires explicit RAPTURE_ALLOW_INSECURE=1.
- Structured JSON logs rotate with three backups (16 MiB by default;
  RAPTURE_LOG_MAX_MB changes this). Debug opcode logs require RAPTURE_DEBUG=1.
- Loopback-only health and Prometheus metrics expose scheduler latency, storage
  waits/transactions/WAL, admissions and queue pressure without user/IP labels.
  Authenticated shutdown supports both platforms; optional profiling requires
  RAPTURE_PPROF=1 and authentication. Runtime descriptors are private files in
  runtime/control, scoped to PID and executable. There is no public admin API.

## Limits

All policy values are validated at startup. Defaults are 200 login connections,
500 world connections, 32 concurrent connections/IP, 120 new connections/IP/minute,
30 login attempts/IP/minute and 10/account/minute. World input allows 512 frames
and 1 MiB/second; ordered output allows 256 packets and 1 MiB including in-flight
bytes. Economy commands allow 20/second. Slow clients disconnect on overflow.

Overrides: RAPTURE_LOGIN_MAX_CONNECTIONS, RAPTURE_WORLD_MAX_CONNECTIONS,
RAPTURE_CONNECTIONS_PER_IP, RAPTURE_CONNECTIONS_PER_IP_PER_MINUTE,
RAPTURE_LOGIN_ATTEMPTS_PER_IP, RAPTURE_LOGIN_ATTEMPTS_PER_ACCOUNT,
RAPTURE_INPUT_FRAMES_PER_SECOND, RAPTURE_INPUT_BYTES_PER_SECOND,
RAPTURE_OUTPUT_PACKETS, RAPTURE_OUTPUT_BYTES and
RAPTURE_ECONOMY_ACTIONS_PER_SECOND. Adjust shared-NAT limits using measurements.
These are resource limits, not a measured maximum player capacity.

## Accounts and TLS

From the server directory, use `build/gx_console account create NAME [--admin]`
or `build/gx_console account password NAME`. Passwords are read without terminal
echo; automation can supply RAPTURE_ACCOUNT_PASSWORD or stdin. Do not place them
in command arguments. Names accept 1–18 UTF-8 bytes, passwords 1–128. Existing
short-password login packets remain compatible; GX adds a bounded authentication
opcode for longer passwords, implemented in both client and server.

Generate certificates with:

```
build/gx_console tls-init -out /private/gx-tls -dns gx.example -ip 127.0.0.1
```

The destination must not exist. Keep CA/server private keys private. Configure
RAPTURE_TLS_CERT and RAPTURE_TLS_KEY using the generated server.env paths in both
services. Certificates expire after 90 days; rotate them before expiry. The CA
lasts five years. Copy only the generated client/network.json and gx-ca.pem to
client data/. The config supports tls, ca_file and server_name. Relative CA paths
resolve beside the configuration file. GX_NETWORK_CONFIG selects another file;
GX_TLS, GX_TLS_CA and GX_TLS_SERVER_NAME override its fields. System trust works
with GX_TLS=1. Invalid certificates never fall back to plaintext. GX_TLS=0 is an
explicit development override. Existing loopback installations need no changes.

## Verification and operating boundaries

Run `tools/check.sh`, the server integration suite, targeted race checks and
frame/credential fuzzing. Integration fixtures use disposable databases and
isolated ports. Tests cover concurrent transactions, replay, injected constraint
and disk-full failures, process crash recovery, backup/restore, TLS client login,
revocation, scheduler cancellation, queue limits and authenticated control.
See [persistence audit](persistence-audit.md) for storage drills.

Native Linux startup and shutdown must be checked against built binaries.
Windows source and CI builds preserve platform support; Linux validation does
not establish native Windows execution. A bounded multi-client probe does not
establish large-scale capacity or complete parity with every historical game
feature. Ground loot and pending in-memory simulation events remain ephemeral
across a crash; durable committed economy changes survive recovery. Backups,
certificate rotation, resource monitoring and game-rule regression testing remain
ongoing operations, not a reason to replace SQLite or add speculative services.

## Linux validation (2026-11-16)

Full vet/unit and server integration checks pass, with race checks on storage,
authentication, bounded networking, diagnostics, scheduling and inventory/loot
ownership. Credential fuzzing exercised over 650,000 inputs; frame fuzzing and
four Python storage-tool tests pass. A 32-client encrypted protocol probe completed
160 durable purchases (p50 4.5 ms, p95 16.3 ms in this local fixture), followed by
disconnect/relogin and durable-state verification. These numbers describe this
fixture and machine, not a production capacity guarantee.

The toolchain and CI use Go 1.27.4, which fixes the standard-library
[HTTP advisory GO-2026-6859](https://pkg.go.dev/vuln/GO-2026-6859).
The private local installation retains loopback development transport; secure
remote defaults and provisioned client/server TLS are covered by integration.

Native Linux binaries pass SIGTERM and authenticated control shutdown with
SQLite integrity and seed preservation. Client and all four server binaries
were installed in the existing Desktop folders; eight protected database/seed/
configuration hashes remain unchanged. The final headless/transport vulnerability
scan on Go 1.27.4 reports no vulnerabilities. Native Windows execution and a
maximum-player benchmark were not performed.
