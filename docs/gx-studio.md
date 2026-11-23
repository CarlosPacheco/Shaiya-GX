# GX Studio

GX Studio is a standalone desktop editor for the loose data files and SQLite
database of Shaiya GX. It opens both sources from the same window and displays
the exact SData field and SQLite column under each readable field name.

| Catalog | Client source | Server source |
| --- | --- | --- |
| Item Mall | `Item/Cash.SData` | `PS_GameDefs_ProductList` |
| Items | `Item/Item.SData` | `PS_GameDefs_Items` |
| Player skills | `Character/Skill.SData` | `PS_GameDefs_Skills`, matching level |
| NPC skills | `Monster/NpcSkill.SData` | `PS_GameDefs_NpcSkills`, matching SData rank |
| Enchant |  | `ItemEnchant.ini` |
| Mob AI messages | | `MobAIMessages.ini` |
| Goddess Blessing | | `KillStatus.ini` |
| World Drops | | `WorldDrops.ini` |
| Item Boxes | | `ItemBoxes.ini` |
| NPCs and Quests | `Npc/NpcQuest.SData` | `NpcQuest.SData` |
| Map Rules | | `Map.ini` |
| Teleport Runes | | `TeleportRunes.ini` |
| Notices | | `Notices.ini` |

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
   mapped columns. Reserved empty item rows and level-zero skill slots are
   omitted from the editor and preserved byte-for-byte on save. Other SQLite
   values and SData fields retain their original values. Normal EP5 SEED envelopes in the definition files
   are preserved on save. Encrypted Cash.SData opens in read-only mode.
   In **Skills → Cast and target**, **8 · Entire raid (heal or buff)** writes
   `TargetType=8` to both catalogs. It uses `ApplyRange` at cast completion for
   player skills with `TypeEffect=2` or `4`, including healing-over-time buffs.
4. Stop the affected GX client and server and keep them stopped through the
   save. Review the proposed changes and choose **Save both**. Restart those
   programs after Studio reports that both destinations were verified.

## Enchant

Select the server's `data/game` folder. **Enchant** edits the
`ItemEnchant.ini` success rates and total bonuses. Edit the chance per million
or enter a percentage with up to four decimal places. The columns stay in
sync, so entering `12.5` in **Chance (%)** gives `125000` per million. Add or
remove levels to adjust the cap for each equipment type, up to +49. Zero
disables enchanting for that type.

## File-backed server catalogs

**Mob AI messages** edits the English UTF-8 text in `MobAIMessages.ini`.
The message IDs are fixed so scripts continue to reference the same line.
**Goddess Blessing** edits the shared tiers in `KillStatus.ini`. The two
factions keep independent counts and use the same thresholds and bonuses.

**World Drops** uses literal percentages with up to eight decimal places
and matches mobs by an inclusive level range. **Item Boxes** supports both
weighted single rewards and independent percentage rolls. Its targets can
be a grade or an exact ItemID. Select the optional SQLite database to
validate reward references against the server's items.

**Map Rules** edits `Map.ini` with the original map sections and character
starts. The editor shows the native keys for faction respawns, instance
schedules, level restrictions, weather and map curses. It retains other
settings in the existing INI. **Notices** edits `Notices.ini`, including
periodic, fixed weekly and manual announcements.

The file editors create a verified backup, refuse to overwrite a file
changed since opening, and replace the file atomically. Restart GX World
to apply a saved catalog.

## NPCs and Quests

Select a client data folder and a server data/game folder with matching
`NpcQuest.SData` definitions. The editor shows NPC services, shop products,
gate destinations, quest rules, rewards and links. Shared fields are saved
to both files. Client dialogue, descriptions and destination labels are
edited separately. The editor preserves the encoding of each file, backs
up both and leaves a journal to complete an interrupted two-file save.
Stop GX World and the client while editing and restart both after saving.

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

Open **Teleport Runes** with the server data/game folder. Select a rune to edit
the Light and Fury map IDs and XYZ coordinates independently, or add a row
with an unused index from 1 to 32767. Items with Special 47 reference that index
through their Variant (`ReqVg`). Removing the row leaves those items without a
destination until their Variant is reassigned or the row is restored.

Saving creates and verifies a backup of `TeleportRunes.ini` in the editor's backup directory.
Restart GX World to apply the changes. See [Teleport runes](teleport-runes.md)
for the original destinations and item setup.
