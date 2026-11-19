# Operating GX Server

GX Server is part of GX under `server/`. In a source checkout, server paths in this guide are relative to `server/`; in a portable archive they are relative to the extracted top-level folder. The former standalone Rapture repository is retired after migration.

## Terminal console

Run `build/gx_console[.exe]` in a terminal, or use `Start GX Server.sh` / `.bat`. Enter `start server` or `shutdown server`. Closing the console leaves login/world detached. Use one console per installation at a time.

Alternatively run `build/gx_login[.exe]` and `build/gx_world[.exe]` directly and manage those process lifetimes yourself. Each binary finds `data/` relative to its executable when no data/database overrides are provided, so launching from another working directory works. Do not mix binaries from different versions.

The controller checks executable paths, including on Windows, so stopping one installation does not stop a separate copy merely because its process names match. Status checks may take several seconds, especially Windows process enumeration.

## Database and persistence

- Distributed seed: `data/rapture.sqlite`; accompanying hash/table counts: `data/rapture.manifest.json`.
- Mutable state: `runtime/rapture.sqlite`. Created from the seed only when absent.
- Login/world handoff keys: the SQLite `GX_SessionHandoff` table.
- Local overrides: `.local/runtime.env` (KEY=VALUE, not committed).
- Logs: `logs/gx_login.log`, `logs/gx_world.log`.

The distributed database intentionally retains the current development accounts, characters and inventory requested by the author. Treat it as a shared test fixture. **SQLite is the primary production database, chosen for its embedded operation and straightforward deployment.** The gameplay-facing database API remains isolated in `server/internal/db`. See the [persistence audit](../server/docs/persistence-audit.md) for transaction-boundary fixes, regression coverage and the scope of concurrency/recovery checks.

Existing `RAPTURE_*` configuration names and SQLite filenames are retained for compatibility. They do not imply an external Rapture installation is required.

## LAN connections

Defaults listen on `127.0.0.1:30800` (login) and `127.0.0.1:30810` (world). To accept another computer, set the following before starting the controller.

Windows PowerShell (replace the example IP with your server's reachable IPv4 address):

```powershell
$env:RAPTURE_LOGIN_LISTEN = '0.0.0.0:30800'
$env:RAPTURE_WORLD_LISTEN = '0.0.0.0:30810'
$env:RAPTURE_PUBLIC_IP = '192.168.1.20'
.\build\gx_console.exe
```

Linux:

```sh
RAPTURE_LOGIN_LISTEN=0.0.0.0:30800 RAPTURE_WORLD_LISTEN=0.0.0.0:30810 \
RAPTURE_PUBLIC_IP=192.168.1.20 ./build/gx_console
```

Allow inbound TCP 30800/30810 on the server firewall. Connect the client with `-login 192.168.1.20:30800`. If using another world port, also pass `-world-port PORT` to the GX client. `RAPTURE_PUBLIC_IP` is the address advertised to the client for world, so it must be reachable from the client's computer; `0.0.0.0` is a listen wildcard, not an advertised address.

For persistent overrides create `.local/runtime.env` in the server directory. Explicit environment values take priority. `RAPTURE_DATA_DIR`, `RAPTURE_DB_PATH` and `RAPTURE_DB_SEED` select data/runtime/seed locations independently. Relative paths in overrides are resolved against the server folder when using the controllers. Prefer absolute paths for direct headless launches with explicit overrides.

## Mob AI

All mob behavior selects the [modern Lua catalog](../server/ai/README.md) by
`Mobs.AI`. Portable installations include an editable `ai/` beside `data/`;
`RAPTURE_AI_DIR` selects another complete catalog. The same Lua definitions are
embedded for installations without that directory. Restart world after edits.
Invalid selected scripts stop startup; there is no legacy Go AI fallback.

## Back up and restore

For a consistent live backup, use the SQLite backup API through the optional Python tool:

```sh
python3 tools/snapshot_sqlite.py --source runtime/rapture.sqlite --output /private/backups/gx.sqlite
```

On Windows replace `python3` with `py -3` and use your destination path. This preserves committed WAL data, validates integrity and emits a manifest. Backups can contain account credentials and player state; keep personal backups outside Git.

To restore, stop both services, verify they are stopped, replace the runtime database with the validated backup and remove any stale `-wal` / `-shm` companions from the stopped runtime before restarting. Never overwrite an open SQLite database or restore only its main file while its old WAL is still present. Keep the previous stopped runtime until restoration is verified.

## Update an installation

1. Stop login/world using the controller.
2. Back up `runtime/rapture.sqlite` consistently and retain `.local/runtime.env`.
3. Extract the new release to a new folder and copy your stopped runtime into its `runtime/` directory. Preserve your override settings. Do not replace the new seed with your runtime unless deliberately preparing a shared test fixture.
4. Start the new version and verify login, character selection and expected progress. Keep the previous folder until verified.

Starting a newly extracted package without copying a runtime creates a new world from the included test snapshot. Replacing the seed alone never resets an existing runtime. Future database-engine migrations will need explicit migration instructions; no automatic cross-engine conversion exists today.

## Renaming and migration

Rapture's source is now `server/` in the GX module. Commands and binaries use `gx_*`, and the advertised server name is GX Server. The legacy `Rapture_LoginLog` table and `rapture.sqlite` filenames intentionally remain, preserving compatibility. Existing clients still use the same login/world endpoints.


## Equipment condition

GX equipment has no durability: attacks never wear items, legacy stored quality
never reduces equipment statistics, and repair requests are ignored without
charging gold. Enhancement failure does not create a broken/zero-quality item.
Legacy database and wire quality fields remain compatible; authored quality
metadata still has unrelated uses such as quest-item drop probabilities. No
personal database migration is required. Source changes take effect after the
world server is rebuilt and restarted.


The static Item Mall catalog loads from SQL `PS_GameDefs_ProductList`. The
client presents native `Cash.SData`; keep both catalogs aligned. See
[cash catalog](cash-catalog.md). Balances, transactions and gifts use SQL.


Blacksmith previews use the read-only GX forge quote (`0x7F03`), with request
serial, live inventory slot references and material count. Replies supply the
success rate per million, gold cost and compatible enchantment hints; no gold or
items change during preview. Actual attempts still validate inventory and use
the same rate/cost functions. The client discards stale quotes and requests a
fresh quote before opening a link/extract confirmation. Enchant attempts execute
immediately; their success percentage is displayed above the action button.

Forge costs use the native ReqIg table without Vg/level discounts or the old
free-cost environment override. Enchant rules are stored in SQLite `PS_GameDefs_EnchantAttack` and
`PS_GameDefs_EnchantDefense`. Each table has its own cap, set by its highest
consecutive level starting at +1, with a maximum of +49. Shipped caps are
+20 for both. Edit the Enchant catalog in GX Studio, then restart gx_world.
The client receives the server bonus tables for its tooltips. See the [feature wiki](wiki/README.md) for exact rules,
item content fields, extraction, recreation and drop persistence.

Operational architecture, resource limits, secure remote transport and local
account management are described in [server modernization](../server/docs/modernization.md).

## Experience curve

GX World loads `PS_GameDefs_ExpDefs` from SQLite at startup. Each consecutive
`Level` has one cumulative `Exp` threshold for reaching the following level.
The highest configured level is the cap. Experience stops at the threshold
used to reach that level. Restart GX World after editing the table.
