# Obelisks and linked bosses

GX loads `server/data/game/Obelisk.ini` after map populations and before world
sweeps start. The supplied file defines nine obelisks (seven on map 0, one on
18, one on 30), 25 linked boss definitions and no active relic-to-portal links.
The live configuration is documented inline in English: every altar variant
and boss/companion phase names its mob IDs from the client catalog. Obsolete
text and unused examples have been removed; all 293 active statements retain
their original values and order.

See [Goddess blessing](goddess-blessing.md) for its complete tables, benefits and UI.

## Capture and goddess blessing

A relic has neutral, Light and Fury monster variants. Initial ownership follows
the configured variant; existing database ownership takes precedence on restart.
Friendly relics cannot be attacked. Destruction removes the guards immediately,
changes ownership and schedules the owning faction's variant using the destroyed
monster's respawn delay plus corpse time. Guards have their own replenishment
schedule while the parent stands; a zero variant means no population.
Mobile companions use ordinary grounded navigation and idle wandering around
their spawn, including normal chase and home behavior. Random spawn candidates
are retried when the original floor is blocked. Stationary principals/pillar
guards retain their authored placement and do not patrol. Parent removal and
managed generation/respawn rules still apply to the mobile companions.

The player dealing the most actual HP damage to a managed principal receives
credit; the first contributor wins a tie. Basic attacks, skills, additional HP
damage, reflection and damage over time participate. Ordinary monster credit is
unchanged. If that player is unavailable, the native fallback changes an owned
relic to the opposing country; neutral requests use the configured base country.
A capture does not fabricate personal PvP kills or rank rewards.

Each capture adds `final country KC threshold / eligible relic count + 1` to
the conquering country's blessing counter and subtracts the corresponding
amount from the previous owner, floored at zero. There is no separate 50% altar
cap. Native debit tier hysteresis is retained. Active maximum blessing freezes
country scoring. Reaching maximum resets relics to their configured base owners,
removes principals and guards and schedules their return after ten minutes.
This reset awards no capture score and produces no capture notice. Neutral base
relics retain the last conquering country for portal dependencies.

## Persistent ownership

The existing `PS_GameData_Keeps` table stores relic configuration keys
`(map << 16) | entry`, separately from sequential network runtime IDs. Missing
rows receive INI defaults; unrelated rows and existing values are preserved.
`OwnCountry` packs native owner in the low nibble (Light 0, Fury 1, neutral 2)
and last conqueror plus one in the high nibble. `ResetTime` is the original
packed calendar field, preserved as such, not a Unix timestamp or a normal
monster respawn deadline. Restart restores saved ownership and creates that
variant immediately; ordinary in-memory corpse/respawn deadlines are not saved.

Ownership writes complete before publication and scoring. Failed writes retry
without repeating capture credit. Country KC continues to use the existing
world persistence cadence. This is not an atomic transaction across ownership
and the separate world KC record; a process crash between those operations can
lose a score update. No runtime database or distributed seed is replaced.

## Portals and boss chains

Configured relic portal dependencies use the last conquering country. Complete
ownership opens that country's side, no matching dependencies closes it and
partial ownership can preserve the previous state. Paired entries exclude each
other. State persists under `0x10000000 | entry` and controls the actual portal
country access, not just its icon. The supplied INI has no live dependency
entries; synthetic fixtures exercise this path without inventing map links.

`bothmob` records create independent boss phases, companions and portal state
per map instance. Principal death removes the current companions; intermediate
phases use the defeated monster's respawn delay. The final phase opens the
configured portal and creates the final companion group. The next cycle closes
the portal and returns to phase zero. Final delays include the native random
minus 60 through plus 59 minute adjustment. `RandomRespawn` records use their
own configured minute interval/range and one of three positions, including a
delayed first spawn. These populations run without nearby players and are not
saved as relic Keeps. Party/guild instances get independent boss state.

## Client presentation and protocol

The server owns gameplay. The client consumes original packets:

| Opcode | Payload |
| --- | --- |
| `2101` | U8 count, then runtime ID U32, owner/state U8, X/Z F32 per record |
| `2102` | Runtime ID U32, owner/state U8 |
| `2103` | Defeated mob U16, previous/new owner U8 |
| `2104` | Portal U16, previous/new state U8 |

A snapshot replaces the current list. Selection, logout and map changes clear
old markers. Invalid lengths, owners, duplicate IDs and nonfinite coordinates
are rejected atomically. Current lists fit within the native 100-record limit;
multiple packets are separate replacement snapshots, not an atomic large list.
Relic owners on the wire are neutral 0, Light 1 and Fury 2. Portal states are
100 closed, 101 Light, 102 Fury and 103 both.

Original `obelisk_0.png`, `obelisk_1.png`, `obelisk_2.png` and `portal.png` draw
at 32 pixels in mini/full maps, before party and quest markers. Open portal
icons are visible to both factions; server checks still decide access. Original
map exclusions apply (68/69 hide these markers, 105/108 hide relic markers,
100–120 hide portal markers). These assets must be supplied externally.

Capture text uses SysMsg 463 for the conqueror and 462 for the losing country,
with the original monster name. Maps 0/18/30/70 receive centered style 8 notices;
other maps use history style 7. News uses the original separate five-entry
screen queue: regular 15px native chat font, ten-second lifetime, centered
17-pixel rows starting at
`screenHeight/2 - 150`, outlined text and the original `TipBox.TGA` frame (`tipbox.png` externally).
The frame uses 4px atlas borders around the native interior dimensions, with
ARGB CD1D1F24 fill; it is not a flat rectangle. It remains in
chat history and plays `Sound/ui_news001.ogg`. Styles 8/24/27/28/29/30/32/33
share this presentation; 30/32 use the native pink text. Portal IDs 1060/1061 switching to 101/102 use
SysMsg 5514/5515. No client-side periodic request or ownership calculation runs.

## Reference and validation

The server reference is `ps_game.exe`, SHA256
`0fb7c9f13b5bcf186118f8bfe73ab214226b93e8aa502b8e6fded4a8312bc547`;
the client is EP5 `Game.exe`, SHA256
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`.
Server loader/destruction/reset/list routines are 00427300, 0042a410,
0042aa40 and 0042b040; client notice handlers are 00562240/00562350.
Read-only reverse engineering established packet, persistence and lifecycle
rules. Phoenix supplied useful content/catalog research; its custom neutral
startup and 50% cap are not used as original behavior.

Checks cover parsing, persistent round trips, failed-save retry, capture/reset,
credit, stale generations, portal hysteresis, boss phases and random timers;
a full original-world boot verifies the nine relic populations. OpenGL pixel
checks cover 32-pixel mini/full map icons. An encrypted client/server test uses
a disposable DB and isolated ports to capture a relic, remove guards, respawn
the new owner, reject friendly attacks, teleport and restore ownership after a
world restart. These checks do not replace a full manual playthrough of every
configured boss chain or a comparison against a running original server.


Successful consumable `050A` acknowledgments play `Sound/ch_potion01.ogg`
within ten units of the local actor, with native linear distance attenuation.
This covers potions and other consumables, including remote uses; local
bag-zero failures play `ui_error001.ogg`. Native callback: `0056ee70`.
These sounds are external assets, converted privately from the original WAVs.
