# Updating to gx-v0.1.10

Close the client and stop GX Server before updating. Preserve the existing
`Server/runtime/`, `.local/` configuration and client settings. Replace matching
Linux/Windows binaries and merge packaged server data/resources, client fonts
and `Client/data/Vehicle/bones.ini`. Protocol 0x11122209 rejects older clients.
The distributed seed already has the new mount definitions; it is not a
replacement for your personal runtime database.

## Existing runtime databases

The three mount-definition fields changed meaning: Type 42 `Range` is the
racial MON index, `AttackTime` is the seat count and `Speed` is the absolute
mounted tier. `ReqIg`, accounts, characters and existing item instances are
preserved. With Python 3, from the portable root, run this once offline:

```sh
python Server/tools/update_mount_content.py --data Client/data \
  --database Server/runtime/rapture.sqlite --backup-dir mount-upgrade-backup
```

Use a new backup directory. The tool accepts original or already-migrated
plaintext GX Item.SData, validates both catalogs before writing, and installs
the shared bones.ini. If an old installation has no runtime yet, starting with
the new packaged seed requires no runtime migration. See [mounts](../mounts.md).

## Game data

The newly prepared complete portable includes refreshed client data: calibrated
original dungeon minimaps, normalized rejected creature mesh headers and five
native mount OGG cues. GitHub binary archives include fonts and authored bone
bindings, but do not upload original client artwork/audio/maps. The [complete Drive package](https://drive.google.com/file/d/1YoUifTj5D-nn378cyFmyT7H4Dw4Vyegp/view?usp=sharing)
now provides gx-v0.1.12, including the refreshed data introduced in gx-v0.1.10.
Maintainer import/repair procedures are documented in [data formats](../data-formats.md)
and [mounts](../mounts.md). Never overwrite an existing personal runtime with
the seed or copy logs, credentials or mutable databases into a distribution.

Fall damage is now enabled by default with 50% of the previous uncapped curve.
`RAPTURE_FALL_DAMAGE=false` explicitly disables it. Ordinary jumps remain safe.
