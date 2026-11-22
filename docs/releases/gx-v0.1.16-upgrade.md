# Updating to gx-v0.1.16

1. Close the client and GX Studio, then enter `shutdown server` in the GX Server Console.
2. Back up your SQLite runtime, local configuration and edited catalogs outside the installation.
3. Use the gx-v0.1.16 client and server together. Install the [combined Linux and Windows binary portable](https://github.com/Spelunkern/Shaiya-GX/releases/tag/gx-v0.1.16) in a writable directory. Keep matching loose game resources in `Client/data/` alongside the bundled fonts and mount bindings. GX Studio is in `Client/Studio/`.
4. For an existing server, preserve its runtime database and reapply your local configuration and catalog edits. An installation already on SQLite schema 35 can keep its runtime. A fresh installation initializes from the included schema-35 development seed.
5. Open `Server/build/gx_console` or `gx_console.exe`, enter `start server`, then open the client for your OS.

[Release notes](gx-v0.1.16.md) · [Portable layout and requirements](../client-portable.md)
