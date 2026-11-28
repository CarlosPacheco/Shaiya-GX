# Upgrade to Shaiya GX gx-v0.1.23

1. Close the client and run `shutdown server` in GX Server's console. Back up
   your runtime SQLite database and local configuration.
2. Extract the new client and server packages into a fresh writable directory.
   Reuse compatible loose client data and preserve the supplied fonts and
   licenses. Keep your runtime database separate from the development seed.
3. Refresh the GX interface rows in your client data:
   `python3 tools/patch_sysmsg.py <client data>`. To get the VSync and
   Anisotropic labels and remove the NPC Sound option from the option art,
   run `tools/patch_option_graphic.py` and `tools/patch_option_sound.py` on
   the same folder, or use the complete portable, which already includes them.
4. Update `data/game/Obelisk.ini` from the server package. `[RandomRespawn]`
   now takes an optional leading percentage followed by up to 16
   coordinates; the old minimum/maximum time pair is ignored.
5. Language overlays are no longer used. Remove any language setting from
   your launcher; the client reads the base data text.
6. Start GX Server, then open the updated client.

See the [server guide](../server.md) and the
[portable guide](../client-portable.md) for the data layout.
