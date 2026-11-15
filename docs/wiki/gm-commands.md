# GX GM role and commands

GM authority comes exclusively from `PS_UserData_Users_Master.Admin`: `0` is
normal and `1` is GM. It is loaded on world login; changing it requires a new
session. `AdminLevel` does not grant this role. The distributed seed enables GM
for its ten test accounts. Existing runtime databases are preserved; the narrow
SQL in `server/tools/enable_test_gms.sql` can update those named accounts in an
intentionally selected offline database.

GM characters display a light-blue `(GM)` prefix (`#66CCFF`) beside their white
world name when visible to the viewer. GMs start visible and vulnerable;
skill-based stealth follows the ordinary faction visibility rules. Stored character names are
unchanged. Turbo starts disabled for GM sessions; `/turbo` enables it and multiplies movement speed by
12, including backward movement, mounts and swimming; server immobilization and
death still prevent movement.

The `/char` and `/attack` controls, including both on/off forms, are temporarily
unavailable. Skill-based stealth remains available.

| Command | Result |
|---|---|
| `/set money amount` | Sets your own gold to 0–4,000,000,000; larger values clamp to the cap. Example: `/set money 10000000`. |
| `/set str value`, `/set dex value`, `/set rec value`, `/set int value`, `/set wis value`, `/set luc value` | Sets that base attribute on your own character to 0–32,000, clamping larger values. Example: `/set str 1000`. Refreshes stats and resource limits immediately, without spending/refunding stat points or changing PvP claims. Equipment and blessing bonuses still apply on top. |
| `/set kill count` | Sets your own kills to 0–1,000,000; larger values clamp to one million. Updates your rank/title and nearby nameplates immediately. Already claimed PvP rewards remain claimed. |
| `/iclear` | Permanently deletes the issuing character's carried items (bags 1–5). Equipped items, mounts, other characters and account storage are preserved. |
| `/turbo` | Toggles the issuing GM's turbo off/on for this session. |
| `/getitem type id quantity` | Grants the complete quantity, respecting definition stack limits and free bag slots. `/getitem 25 1 200` grants 200 apples in one stack with the current 255 limit. |
| `/mmake mobid quantity` | Spawns 1–100 live mobs around the GM, with their normal stats, AI and drops, without automatic respawn. |
| `/nmake type id` | Spawns exactly one static NPC around the GM, retaining its definition and services. `/nmake 1 1` selects merchant 1; `/nmake 2 1` selects gatekeeper 1. Type and row ID are separate arguments. |
| `/cmove mapid` | Teleports to the nearest safe arrival relative to the current map position. Uses rebirth boxes, or authored portal/gatekeeper arrivals when no box is available. GM travel can inspect base instance maps without party/guild entry requirements. |

Item quantities are positive integers, up to 119,880; grants that do not fit are
rejected entirely. Inventory changes are transactional. Spawns use expanding
rings, reject blocked/occupied positions, and enter nearby clients' visibility
immediately. Spawned NPCs follow terrain height on clients. Spawned entities live only in memory and disappear when the world server stops or restarts.
Maps without a known safe arrival are rejected. Commands are case-insensitive,
validated on the server, and never broadcast as ordinary chat when handled.
Finish an active exchange before issuing GM commands. `/set` accepts only a variant and a non-negative decimal integer; it always targets the issuing GM and persists before confirming. The configured base-attribute cap is 32,000, below the signed 16-bit DB ceiling of 32,767; effective stats with gear/buffs may be higher. Ordinary allocation also refuses increases beyond this persistent base-stat cap without spending points. It sets the total rather than adding to it, and does not award quest or country kill credit.

The client sends commands through the normal chat request independently of the
selected UI channel. GX server response `0x7F01` contains self character ID `u32`,
GM `u8`, turbo `u8`, UTF-8 message byte length `u16`, and message bytes, all numeric
fields little-endian. GM/turbo bytes must be 0 or 1; turbo requires GM. Ordinary
`0x0201` marker 2 supplies GM role metadata for remote nameplates. A forged client
response cannot grant server privileges. This extension requires the updated GX
client and world server; the original client does not implement turbo state.

All command responses, usage hints and validation messages are in English and
use green system-chat style 4 (`#33FF33`), including refusals.

## Practical examples

Use `/turbo` once to enable fast movement and again to restore ordinary speed.
It resets on the next session; the account remains a GM. Movement restrictions
from death or immobilizing effects still apply.

`/getitem 25 1 255` requests 255 units of Type 25, ID 1. Arguments are separate
integers, not the packed item ID 25001. The request uses the selected definition's
stack limit; a nominally large grant does not force non-stackable items into one
stack. If the entire quantity cannot fit, the operation is rejected rather than
partially granting it. To discover IDs, inspect the intended SQL Items table or
matching client definitions instead of assuming names uniquely identify items.

`/mmake 1 3` requests three mobs using definition ID 1. They behave like live
mobs, so select a suitable test location. This command does not author a permanent
spawn row. `/nmake 1 1` uses NPC service type 1 and definition row 1; it is not
interchangeable with `/nmake 2 1`.

`/cmove 1` asks for a safe arrival in map 1. It does not take X/Y/Z coordinates,
and it does not promise to preserve the exact current location. A map that has no
known safe arrival is refused.

Use `/iclear` only when intentionally discarding the issuing GM's carried items.
It is a permanent inventory mutation, unlike closing a panel or removing a
blacksmith selection. It does not delete equipped gear or clear another account.

## Role changes and diagnosis

If commands are refused, verify Admin in the **runtime database actually used by
world**, end the session and log in again. Editing the seed does not modify an
existing runtime copy; editing AdminLevel does not grant GM authority. The client
badge and turbo state reflect server replies and cannot elevate privileges.

A missing mob/item/NPC definition, invalid number, full inventory, occupied spawn
area or unavailable safe map arrival can reject an otherwise correctly typed
command. Follow the English usage response. Restarting world removes temporary
spawns but does not reverse already persisted item grants or deletions.

See [Server operations](server-operations.md) before changing account roles or
selecting a database, and [Item data](item-data.md) for definition identifiers.

See [Stealth and GM protection](stealth-and-gm-protection.md) for visibility,
combat disengagement, ordinary stealth reveal rules, retired GM controls
and the recovered visual.

## Collision diagnostics (local development)

`/cview` toggles the in-game collision overlay; `/cview on` and `/cview off`
select an explicit state. The client handles this presentation command locally
after checking its server-confirmed GM role. It consumes the command privately,
and revoking GM status disables the view. Movement continues to collide.

Gold shows blocking faces, cyan model support surfaces, blue the terrain and
magenta the player's ellipsoid. Grass, VANI/MANI animations and models without
collision remain traversable according to the recovered original rules.
See [Original world collisions](../collisions.md) for geometry selection,
native evidence and validation. The server uses versioned geometry and
authoritative movement checks; see [ground navigation](../mob-navigation.md).

## Teleport from a map

GM accounts can click the circular minimap or the full local map to teleport
to that X/Z point when it is valid. A blocked or floorless click resolves to
a nearby safe point within 32 units instead. Dragging the full map pans it
without teleporting.
In World Map, select a region and click its map preview to teleport there;
GM previews allow both factions. No `/goto` chat command is added.
The server validates the account role and map, rejects trading, and resolves
the destination floor and full player-body clearance from its own resources.
It prefers the exact click, then the nearest navigable point, with a bounded
expanding search when that point is obstructed. Blocked mask cells, map bounds
and missing floors remain invalid; absence of any safe nearby point is reported.
The client never supplies a height. On layered maps the nearest navigable
floor to the current height is used locally; cross-map arrivals use the map's
terrain height as the floor hint.
