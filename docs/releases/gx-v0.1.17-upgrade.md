# Updating to gx-v0.1.17

1. Close the client and GX Studio, then enter `shutdown server` in GX Server Console.
2. Back up the existing SQLite runtime, local configuration and edited catalogs outside the installation.
3. Extract the [gx-v0.1.17 packages](https://github.com/Spelunkern/Shaiya-GX/releases/tag/gx-v0.1.17) to a writable folder. Update the client and server together. Keep the packaged fonts and mount bindings and merge compatible loose game resources into `Client/data/`.
4. When updating older client data, use the release's matching interface art and Item.SData. Run `python3 tools/gx_ini.py /path/to/Client/data` from the source checkout to convert legacy map and filter configuration and `python3 tools/patch_sysmsg.py /path/to/Client/data` to add missing GX text. Preserve any custom loading messages in `Tip.INI`.
5. Keep your existing `Server/runtime/rapture.sqlite` and reapply local configuration. The first start upgrades the runtime to schema 36. A fresh installation starts with the included schema-36 development seed.
6. Open `Server/build/gx_console` or `gx_console.exe`, enter `start server`, then open the client for your OS.

For graphics reports, include the client's `logs/client.log` and describe where the problem occurred.

[Release notes](gx-v0.1.17.md) · [Portable layout and requirements](../client-portable.md)
