# Updating to gx-v0.1.11

Update the matching client and server together. This release retains protocol
revision `0x11122209`, but adds coordinated authentication support and database
migrations; an unchanged version number is not a promise that mixed builds work.

## Existing installations

1. Close the client. In the server console enter `shutdown server` and confirm
   login/world are OFF before replacing files or taking an offline backup.
2. Back up `Server/runtime/rapture.sqlite` consistently, including committed WAL
   content. See [SQLite operations](../../server/docs/sqlite.md). Preserve all of
   `runtime/`, `.local/`, client settings and any private TLS files separately.
3. Replace matching executables and merge the packaged server data/resources,
   fonts, documentation, tools and the complete new `Server/ai/` directory.
   Remove obsolete `Server/build/gx_server[.exe]` and its dedicated icon. The
   controller is now `Server/build/gx_console[.exe]`; replace old GUI launchers
   with `Start GX Server.sh` / `.bat` or run that console directly.
4. Keep your existing mutable runtime database. The distributed
   `Server/data/rapture.sqlite` is an intentional development seed, not a runtime
   replacement. Startup applies the versioned additive migrations to your runtime.
5. Start the console and enter `start server`. Review login/world startup logs,
   then verify login and a save/relogin cycle with the matching client.

Do not downgrade to an older server after migration/password upgrades without a
verified rollback backup. Legacy password records upgrade to Argon2id on successful
login; account/password administration writes hashes directly. Password changes
revoke existing handoffs and sessions. Use the new console's administration
commands for account/password changes rather than old tools.

## Remote connections

Loopback development installations retain their normal defaults. Remote servers
use TLS 1.3 by default. Run `gx_console tls-init` as documented in
[modernization](../../server/docs/modernization.md#accounts-and-tls), configure both
server processes and distribute only the public client trust configuration/CA.
Keep CA/server private keys private. Existing users of insecure remote transport
must configure TLS before connecting; the development escape hatch is explicit.

## Assets, AI and older upgrades

The complete local folder includes `Client/data/`. GitHub binary updates omit
original client game resources; keep your compatible loose data from the complete
package. Current GitHub packages include fonts, authored vehicle bindings, server
physics/maps and the modern AI catalog. Never distribute an active runtime,
`.local/`, logs, packet dumps, credentials or private TLS keys.

The modern AI runtime uses `Server/ai/` or its matching embedded default catalog.
Remove obsolete `Server/data/game/AI/`, `Server/data/game/Lua/` and
`Server/data/game/mob_ai.ini` when updating an older installation. Do not merge an
old policy catalog over the new one. A malformed selected catalog fails startup.

If upgrading from before v0.1.10, also follow the
[v0.1.10 mount-definition migration](gx-v0.1.10-upgrade.md). This release does not
resnapshot your personal database or run the mount-data migration automatically.
