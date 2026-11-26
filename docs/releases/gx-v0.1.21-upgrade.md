# Upgrade to Shaiya GX gx-v0.1.21

Update the client and server together to receive the new Guild House quotes
and compatible movement behavior. The combined GitHub portable contains both
native platforms. Use your existing compatible loose client data with it.

1. Close the client and enter `shutdown server` in GX Server's console.
2. Back up your runtime SQLite database and local configuration. Keep the
   published development seed separate from your runtime.
3. Extract the new packages into a fresh writable directory and merge the
   supplied fonts and mount bindings into the existing client data.
4. Confirm that `Server/data/geometry/mob-dungeons/catalog.json` is present,
   together with the main catalog in `Server/data/geometry/`. For custom maps,
   rebuild the appropriate geometry before installing it. See
   [mob navigation](../mob-navigation.md).
5. Keep your existing `RAPTURE_*` settings and runtime paths. Start the server
   and open the client executable for your platform.

`Npc/GuildHouse.SData` is optional. World reads `GuildHouse.ini` for purchase
and upgrade pricing. GX Studio is installed from its
[independent releases](https://github.com/Spelunkern/GX-Studio/releases).
