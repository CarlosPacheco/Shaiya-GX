# Server map connectivity audit

Snapshot: 2026-11-15. The server loader accepts 77 numeric `data/maps/*.svmap`
files: maps 0–67 and 100–108. This historical audit used the server's SVMAP parser and
`NpcQuest.SData` definitions, including only gatekeepers actually placed in those
maps. It found 265 portal rows and 98 nonempty placed-gatekeeper destinations;
105 of those routes stay on the same map. The current server reads NPC destinations
and map policies from SQLite. See [map rules](wiki/map-rules.md) for current access checks.

An orphan below has **no incoming route from another loaded map**, including
conditionally closed portals. GM commands, login/respawn positions, guild/event
instance creation and consumable teleports are not NPC/portal entrances. NPC
definitions without a placed NPC do not count. Player-specific faction, level,
gold, party and schedule checks can restrict the remaining routes further.

| Map | Current client name | NPCs | Initial regular mobs | Outgoing routes |
| --- | --- | ---: | ---: | --- |
| 39 | Arena | 0 | 0 | None |
| 41 | Prison | 0 | 0 | None |
| 60 | Elimental Cave 2 | 0 | 481 | Portal to 19, Light |
| 105 | Battle Zone | 0 | 104 | None |
| 106 | Battle Zone | 0 | 1 | None |
| 107 | Battle Zone | 0 | 1 | None |
| 108 | Battle Zone | 0 | 104 | None |

Counts are parsed initial regular spawns, before dynamic events or navigation
rejections. Names in this snapshot come from the legacy client `sysmsg-uni.txt`/native map-ID lookup,
including its spelling; they do not prove that local geometry matches the name.
Maps 100–108 have server data but no matching numeric WLD in the installed
0–67 client resource set.

Maps 100–104 have incoming links inside the battle-map set, so they are not
zero-incoming orphans. However, following all declared NPC/portal routes from
starting maps 1 and 2, even allowing conditional gates, never reaches any map
100–108. Maps 100–104 are therefore an isolated route group as well.

## Conditional and dangling links

Maps 48 and 49 (Cryptic One's Throne) have incoming portals from 19 and 20,
respectively. Their external SVMAP gates use IDs 1001 and 1002, rather than an
initial open faction value. These are configured event/boss gates, not missing
links; actual availability depends on runtime event state.

Six portal rows from maps 100, 101 and 104 target map 200, which is absent from
the loaded map-file set. Those are dangling destinations, not an extra playable
map. The audit reports them without altering server content.

## Map 60 resource mismatch

The installed `world/60.wld` is byte-identical to `world/25.wld`, a 1024-unit
field. Core's `world/60.wld` is instead a dungeon using `EV_worldcup2010.dg`, and
its numbered minimap depicts that stadium. Its `Dungeon_MiniMap.cfg` scale is
therefore not a calibration for the currently installed field. The importer
skips that incompatible image. No world geometry, server spawns, map names or
routes are changed by the minimap work.
