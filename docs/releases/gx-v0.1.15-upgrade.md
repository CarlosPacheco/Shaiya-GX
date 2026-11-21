# Updating to gx-v0.1.15

1. Stop GX Server with `shutdown server` and close the client and GX Studio.
2. Back up your SQLite runtime, local configuration and edited catalogs. Keep the backups outside the new installation.
3. Install the gx-v0.1.15 client, GX Studio, server binaries and server resources together in a writable directory. Keep the matching loose client data next to the executables in `Client/data/`, including the updated item and skill catalogs and User Options artwork. The older Drive snapshot has gx-v0.1.14 binaries and data.
4. Carry over your existing runtime and reapply deliberate catalog changes to the new defaults. If you configured `GuildCreateMoney`, set its value as `CreateGuildCost` under `[House]` in `GuildHouse.ini`. Include `PortalBounds.json` with the updated server resources.
5. Start the terminal console and enter `start server`. Existing runtimes migrate automatically through schema 35. A fresh installation copies the included schema-35 development seed.

Schema migrations cover retired item and skill definitions, durable skill-reset cooldowns, explicit experience and combat chance fields, restored consumable links and unified character progression. Keep the pre-upgrade backup if you need to return to the previous version.

[Release notes](gx-v0.1.15.md) · [Portable layout and requirements](../client-portable.md)
