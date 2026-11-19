# Teleport runes

Fixed-destination runes use item `Special` 47. Set the native `ReqVg`
(Variant) in Item.SData and the matching server item definition to a row's
`Index` in `PS_GameDefs_TeleportRunes`. Indices range from 1 to 32767.
Each row has `LightMapID`, `LightX`, `LightY`, `LightZ`, and the equivalent
`FuryMapID`, `FuryX`, `FuryY`, `FuryZ`. The world service reads the table
on startup. Several items can reference the same row.

| Index | Rune | Light map | Fury map |
| --- | --- | ---: | ---: |
| 1 | Capital Recall Rune | 35 | 36 |
| 2 | Bootleggery Recall Rune | 42 | 42 |
| 3 | Arena Recall Rune | 40 | 40 |
| 4 | Guild House Teleport Stone | 51 | 52 |
| 5 | Legacy transport rune (item 100169) | 200 | 200 |

The seeded coordinates come from the EP4.5 server. Map 200 is absent in the
standard GX installation, so the fifth rune refuses travel until its
destinations are installed or reconfigured.

Travel takes five seconds and consumes the rune only after the destination
is validated and the transfer is saved. The server checks the current
character's faction, map permissions and ground at the configured point.
A guild house destination opens or reuses the member's private guild instance
only for a ranked guild with its house purchased and enabled.

Migration 22 updates the five original server item definitions. To synchronize
an existing client's native Item.SData, apply this CSV with `cmd/itempatch`
using distinct input and output paths, then install the patched output.

```csv
type,type_id,req_vg,special
100,131,1,47
100,132,2,47
100,133,3,47
100,138,4,47
100,169,5,47
```

Use [GX Studio's Teleport Runes catalog](../gx-studio.md#teleport-runes) to edit
destinations or add custom rows with a verified SQLite backup.
