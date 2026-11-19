# Shaiya GX gx-v0.1.13

Changes since **gx-v0.1.12**.

- **Auction House.** Use a Notice Board to browse the realm-wide market, filter by item family, list stacks, bid, buy out, track favorites, and claim items and gold. SQLite transactions protect escrow and refunds through disconnects and concurrent changes.
- **GX Studio.** The standalone desktop editor ships for Linux and Windows. It edits loose SData and SQLite catalogs with visible native field names, verified backups, and recovery. Catalogs cover items, player/NPC skills, NPCs and quests, the mall, enchanting, AI messages, Goddess Blessing, world drops, item boxes, map rules, and teleport runes.
- **World and progression.** Add faction guard patrols and stationary summoned guards, synchronized rain and storms, authored dungeon skies and fog, map requirements and curses, shared experience thresholds and level cap, configurable world loot, and transactional item boxes. Global and scheduled notices appear in chat.
- **Travel and administration.** Add recorded points, interruptible party and raid summons, faction-specific teleport runes, and account-targeted GM travel and summons. The GM cure command restores resources and resets item and mount cooldowns.
- **Combat and items.** Update Fire Thorn, Fire Bush, traps, Detection, Health Assistant, Taunt, Sacrifice, Potential, Degeneration, cooldown prerequisites and modifiers, and NPC lapis effects. Correct skill tooltips and item absorption, remove experience and inventory/gold loss from PvP deaths, and fix a Lua monster AI memory leak.
- **Client presentation.** Restore chat filters, help, scrollbars, whisper entry and item links. Add the third shortcut bar and Revolver, improve minimap zoom and status labels, blend animation changes, and keep an inventory icon stationary until dragging starts. Update outdoor lens flare, sky and water resources.
- **Server and packaging.** Ship the refreshed development seed, synchronized mall stack limits, SQLite catalogs and migrations, 68 maps of server geometry, Lua AI, native client, GX Studio, and terminal server packages. Refresh the Wiki and support LF and CRLF migration histories on both platforms, and fix cross-platform fixtures.

[Upgrade guide](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/releases/gx-v0.1.13-upgrade.md) · [Wiki](https://github.com/Spelunkern/Shaiya-GX/wiki)

GitHub packages include matching binaries, fonts, and server resources. The complete offline folder prepared by the maintainer also carries matching loose client game data. The included development seed retains test characters and inventory with the previously published test credentials. The existing v0.1.12 Drive link remains the earlier package.
