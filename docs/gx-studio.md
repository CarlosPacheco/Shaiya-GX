# GX Studio

GX Studio is a standalone desktop editor for the loose data files and SQLite
database of Shaiya GX. It opens both sources from the same window and displays
the exact SData field and SQLite column under each readable field name.

| Catalog | Client source | SQLite source |
| --- | --- | --- |
| Item Mall | `Item/Cash.SData` | `PS_GameDefs_ProductList` |
| Items | `Item/Item.SData` | `PS_GameDefs_Items` |
| Player skills | `Character/Skill.SData` | `PS_GameDefs_Skills`, matching level |
| NPC skills | `Monster/NpcSkill.SData` | `PS_GameDefs_NpcSkills`, matching SData rank |
| Enchant |  | `PS_GameDefs_EnchantAttack` / `PS_GameDefs_EnchantDefense` |
| Mob AI messages | | `PS_GameDefs_MobAIMessages` |
| Goddess Blessing | | `PS_GameDefs_GoddessBlessing` |
| World Drops | | `PS_GameDefs_WorldDrops` |
| Item Boxes | | `PS_GameDefs_ItemBoxes` / `PS_GameDefs_ItemBoxRewards` |
| NPCs and Quests | `Npc/NpcQuest.SData` | NPC and quest tables in `PS_GameDefs` |
| Map Rules | | `PS_GameDefs_MapProperties` / `PS_GameDefs_CharacterStartPositions` |
| Teleport Runes | | `PS_GameDefs_TeleportRunes` |

## Build

Use the repository Go toolchain and [Fyne desktop build prerequisites](https://docs.fyne.io/started/quick/).
Studio uses CGO for the GUI. The GX server commands build independently of it.

Linux, from the repository root:

```sh
source tools/go-env.sh
go build -trimpath -ldflags '-s -w' -o 'build/GX Studio/GX Studio' ./cmd/gx_studio
'build/GX Studio/GX Studio'
```

Windows, from the repository root in PowerShell with a compatible GCC on PATH:

```powershell
$env:CGO_ENABLED = '1'
go build -trimpath -ldflags '-H windowsgui -s -w' -o 'build/GX Studio/GX Studio.exe' ./cmd/gx_studio
& 'build/GX Studio/GX Studio.exe'
```

Keep the executable in a writable folder for the local backups and recovery journal.

## Open and edit

1. Choose Light, Dark or Dark blue on the opening screen. Studio remembers the
   theme in `gx-studio.ini` even before an installation is opened. Light is the default.
   Choose the client's loose `data/` directory. Studio checks known portable
   runtime locations and the `RAPTURE_DB_PATH` environment override. The database
   picker also accepts an explicit existing SQLite file. Verify the displayed path
   before opening. Studio remembers the last successfully opened paths in
   `gx-studio.ini` beside the executable. A distributed seed can have a separate manifest
   that becomes stale after editing.
2. Choose a catalog. Item Mall shows Popular, Essentials, Buffs and Packages in the sidebar.
   Select a product to edit its fields in the right pane. Click the icon to open
   its picker of populated client atlas tiles. The Slots tab lists all 24 slots
   in order for direct ID and quantity edits and includes a searchable item picker.
   Apply edits to the draft before saving both catalogs. The differences view shows how the draft
   compares with SQLite and can import a selected SQLite product's fields. A
   product code edit removes the old SQLite key and adds the new one.
3. Items opens by family. The compact list shows type, type ID, name, level, model and
   icon. Select a row to edit it in the right pane and click the preview to pick
   an inventory icon from that item's client atlas. Skill catalogs are grouped
   by `TypeEffect`.
   Items and skills edit existing definitions. IDs, item families and skill
   ranks are fixed. Shared fields are enabled only for records with a unique
   SQLite counterpart. Appearance and description can be edited on the client
   side independently. Each definition save updates only the explicitly edited
   mapped columns. It preserves other SQLite values, reserved slots and
   untouched SData bytes. Normal EP5 SEED envelopes in the definition files
   are preserved on save. Encrypted Cash.SData opens in read-only mode.
4. Stop the affected GX client and server and keep them stopped through the
   save. Review the proposed changes and choose **Save both**. Restart those
   programs after Studio reports that both destinations were verified.

## Enchant

Choose Enchant with the server SQLite database. The Attack and Defense
tabs let you edit the chances and bonuses directly. The last level in each
tab sets that equipment type’s cap. Add the next level or remove the highest
level to change it, up to +49. An empty table disables enchanting for that equipment type. Removing levels also removes their
bonuses for existing gear. **Save database** takes a full SQLite backup and
applies both tables in one transaction. Restart GX World to load the new rules.

## Mob AI messages

Open the server SQLite database in **Mob AI messages**. Edit **Message**
(`Text`) beside its fixed **Message ID** (`Index`), which the AI scripts use.
**Save database** creates a verified SQLite backup before saving the text.
Restart GX World to load the changes. To display Korean with no client
folder selected, place the GX CJK font and its license in `fonts/` beside
the Studio executable.

## Goddess Blessing

Open **Goddess Blessing** with the server SQLite database. Select a tier
to edit the kill threshold and bonuses used by both factions. Each bonus
shows its unit, SQLite column, original type ID and effective total after
inheritance from earlier tiers. **Save database** takes a verified SQLite
backup and updates the shared table in one transaction. Restart GX World
to apply the changes.

## World Drops

Open **World Drops** with the server SQLite database. Add, edit or remove
rules by grade and inclusive mob level range. Enter **Drop chance (%)**
with up to eight decimal places. `0` disables the rule. SQLite stores
`DropChance` in units of 0.00000001 percentage points, so `0.0001%`
is stored as `10000`. Saving creates a verified SQLite backup. Restart
GX World to apply the changes.

## Item Boxes

Open **Item Boxes** with the server SQLite database. A bag with
`Special = 42` uses its `ReqVg` as the box `Index`. Choose **Grade** or
**Item ID** and add the linked rewards.

Normal boxes award exactly one reward using relative weights from 0 to 1000.
Studio shows the normalized probability next to each weight. Independent
boxes roll against each reward's literal percentage and can award any subset.
SQLite stores independent chances in units of 0.00000001 percentage points.
A zero chance disables that row in either mode. Saving backs up SQLite and
updates the box and reward tables together. Restart GX World to apply edits.

## NPCs and Quests

Select a matching client data folder and server SQLite database. The NPC
browser groups rows by service and shows shop products, gate destinations
and ordered quest links. The quest browser groups the original fields into
requirements, starting items, objectives and reward choices. Item quest
links can be edited by item type and ID.

The same save updates gameplay fields in SQLite and the native client file
and preserves the client's encryption. Dialogues, descriptions, journal text
and destination labels are stored only in the client file. NPC and quest
lookup rows remain fixed. A save takes both backups before replacing either
catalog. Stop GX World and the client while editing and restart both after saving.

## Map Rules

Open **Map Rules** with the server SQLite database. **Map properties**
contains the level range, instance rules, faction rebirth destinations,
schedules, weather and three map curse slots. **Character starts** edits
the four race birth locations, also used when a character has no eligible
rebirth point or must leave a level-restricted map. Blank rebirth and
schedule blocks are stored as NULL. Map modifiers are reserved.

A zero minimum or maximum removes that side of the level restriction.
Weather uses 0 for clear, 1 for rain and 2 for snow. Weather events last ten
minutes. A zero interval starts an event every 60 minutes of world uptime.
A nonzero curse references the skill's level-1 persistent buff or debuff.
The effect is removed on leaving the map. Save creates a verified SQLite
backup. Restart GX World to apply the changes.

## Backups and recovery

Every applied save first creates `backups/<timestamp-id>/` beside the GX Studio
executable. It holds a byte-identical copy of the original SData, the proposed
SData and an integrity-checked SQLite snapshot including committed WAL changes.
The database snapshot contains all data in the selected database. Keep the
backup directory private.

Studio acquires an SQLite write transaction, checks for changes since the
catalog was opened, publishes a recovery journal and atomically replaces the
SData before committing SQLite. If a save is interrupted, reopen the same GX
Studio folder with the affected programs still stopped. **Recover last save**
checks the on-disk SQL state and restores or completes the corresponding SData
replacement. Unexpected external changes leave the journal and backups
available for review. Keep the executable folder and its backups together
until recovery is complete.

## Teleport Runes

Open **Teleport Runes** with the server SQLite database. Select a rune to edit
the Light and Fury map IDs and XYZ coordinates independently, or add a row
with an unused index from 1 to 32767. Items with Special 47 reference that index
through their Variant (`ReqVg`). Removing the row leaves those items without a
destination until their Variant is reassigned or the row is restored.

Saving creates and verifies a SQLite backup in the editor's backup directory.
Restart GX World to apply the changes. See [Teleport runes](teleport-runes.md)
for the original destinations and item setup.
