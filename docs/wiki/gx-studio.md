# GX Studio

GX Studio is a standalone Linux and Windows editor for loose Shaiya GX data
and the server's SQLite runtime. The [source guide](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/gx-studio.md) lists build
commands, paths, field mappings and recovery behavior.

Choose the client data folder and verify the SQLite path. Studio remembers
both paths and the Light, Dark or Dark blue theme in `gx-studio.ini` beside
the executable. Each edit shows the readable name and the exact native field
or SQLite column it changes.

| Catalogs | Sources |
| --- | --- |
| Item Mall, Items, Player skills, NPC skills, NPCs and Quests | Client SData and mapped SQLite definitions |
| Enchant, Mob AI messages, Goddess Blessing, World Drops, Item Boxes, Map Rules | SQLite |

Studio backs up the affected sources before writing. Use the active runtime
path for server changes. Reload loose client data by restarting the client and
reload server catalogs by restarting GX World. See [map rules](map-rules.md),
[item data](item-data.md) and [SQLite backups](server-operations.md#back-up-and-restore).
