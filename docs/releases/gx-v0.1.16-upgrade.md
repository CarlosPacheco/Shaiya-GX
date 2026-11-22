# Updating to gx-v0.1.16

1. Close the client and GX Studio, then enter `shutdown server` in the GX Server Console.
2. Back up your SQLite runtime, local configuration and edited catalogs outside the installation.
3. Use the gx-v0.1.16 client and server together. Extract the [complete portable from Google Drive](https://drive.google.com/file/d/1-yK2bZAsWv3nKK-RBryVa-tstWfU3pRz/view?usp=sharing) into a writable directory. It includes the matching loose game resources, fonts and mount bindings in `Client/data/`. You can also use the [GitHub binary packages](https://github.com/Spelunkern/Shaiya-GX/releases/tag/gx-v0.1.16) with matching loose game resources in that folder. GX Studio is in `Client/Studio/`.
4. For an existing server, preserve its runtime database and reapply your local configuration and catalog edits. An installation already on SQLite schema 35 can keep its runtime. A fresh installation initializes from the included schema-35 development seed.
5. Open `Server/build/gx_console` or `gx_console.exe`, enter `start server`, then open the client for your OS.

[Release notes](gx-v0.1.16.md) · [Portable layout and requirements](../client-portable.md)
