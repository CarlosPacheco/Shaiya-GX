# GX Studio

GX Studio is a standalone desktop editor for the loose data files and SQLite
database of Shaiya GX. It opens both sources from the same window and displays
the exact SData field and SQLite column under each readable field name.

| Catalog | Client source | SQLite source |
| --- | --- | --- |
| Item Mall | `Cash.SData` | `PS_GameDefs_ProductList` |
| Items | `Item/Item.SData` | `PS_GameDefs_Items` |
| Player skills | `Character/Skill.SData` | `PS_GameDefs_Skills`, matching level |
| NPC skills | `Monster/NpcSkill.SData` | `PS_GameDefs_Skills`, level 100 for SData rank 1 |

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

1. Choose the client's loose `data/` directory. Studio checks known portable
   runtime locations and the `RAPTURE_DB_PATH` environment override. The database
   picker also accepts an explicit existing SQLite file. Verify the displayed path
   before opening. Studio remembers the last successfully opened paths in
   `gx-studio.ini` beside the executable. A distributed seed can have a separate manifest
   that becomes stale after editing.
2. Choose a catalog. Item Mall shows Popular, Essentials, Buffs, Packages and
   Hidden, with product icon previews, a store icon picker and a searchable item
   picker for all 24 bundle slots. The differences view shows how the draft
   compares with SQLite and can import a selected SQLite product's fields. A
   product code edit removes the old SQLite key and adds the new one.
3. The left pane groups Items by the native `Type` and skills by `TypeEffect`.
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
