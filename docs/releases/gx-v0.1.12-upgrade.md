# Updating to gx-v0.1.12

This is the client-side map-transfer hotfix for v0.1.11. The server protocol and
SQLite schema do not change in this hotfix. Existing v0.1.11 game data remains
compatible; there is no new data or database migration to run.

1. Close the client. If replacing the bundled server binaries, enter
   `shutdown server` in its console and confirm the services are OFF first.
2. Preserve your complete `Server/runtime/`, `.local/`, client settings and
   private TLS material. Back up the runtime consistently before an upgrade.
3. Replace the client with the matching v0.1.12 binary for your OS. Matching
   terminal-only server binaries are also supplied. Keep the existing data,
   configuration and mutable runtime; never overwrite a personal runtime with
   the packaged development seed.
4. Restart the server if it was stopped, then reconnect the client. Repeated
   `/cmove` and other cross-map placements now complete the load handshake.

The complete maintainer-prepared v0.1.12 folder includes both operating systems,
client data, fonts/licenses, server seed DB, maps/physics, Lua AI, resources,
tools and docs. First startup creates an independent runtime from its seed.
GitHub binary packages omit original client game resources; retain those from
the v0.1.11 complete Drive download when using binary updates.

If upgrading from before v0.1.11, also follow the
[v0.1.11 upgrade guide](gx-v0.1.11-upgrade.md), including coordinated authentication,
TLS and migration guidance. Earlier mount-definition upgrades remain documented
in the [v0.1.10 guide](gx-v0.1.10-upgrade.md).
