# GX Studio

GX Studio is a standalone Linux and Windows editor for loose Shaiya GX data
and the server's SQLite runtime. The [source guide](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/gx-studio.md) lists build
commands, paths, field mappings and recovery behavior.

Choose the client data folder, server `data/game` folder or SQLite file as needed. Studio remembers
both paths and the Light, Dark or Dark blue theme in `gx-studio.ini` beside
the executable. Each edit shows the readable name and the exact native field
or SQLite column it changes.

| Catalogs | Sources |
|---|---|
| Item Mall, Items, Player skills, NPC skills | Client SData and mapped SQLite definitions |
| NPCs and Quests | Client and server `NpcQuest.SData` |
| Enchant, Mob AI messages, Goddess Blessing, World Drops, Item Boxes, Map Rules, Teleport Runes, Notices | Files in the server's `data/game` folder |

Studio backs up the affected sources before writing. Use the active runtime
path for server changes. Reload loose client data by restarting the client and
reload server catalogs by restarting GX World. See [map rules](map-rules.md),
[item data](item-data.md) and [SQLite backups](server-operations.md#back-up-and-restore).


Server files include `ItemEnchant.ini`, `MobAIMessages.ini`, `KillStatus.ini`,
`WorldDrops.ini`, `ItemBoxes.ini`, `Map.ini`, `TeleportRunes.ini` and
`Notices.ini`. The mob message file uses `[Messages]` with one quoted English
message per numeric ID. Item boxes support both weighted single rewards and
independent rolls. Notices support intervals and weekly times.
