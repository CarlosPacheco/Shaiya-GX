# GX data layout and media formats

The playable package is:

```text
Shaiya GX/
  Shaiya GX          # Linux; Shaiya GX.exe on Windows
  data/
    interface/       # PNG only, including icons, portraits and minimaps
    Sound/           # Ogg Vorbis, including Music/
    Character/
    World/
    ...
```

The executable discovers `data/` beside itself regardless of the current working directory. `-data` / `SHAIYA_DATA` can explicitly select a different data directory for development. No SAH/SAF index, launcher, Python or external media codec is needed to run the game. Fonts are external resources under `data/fonts/`; no system font installation is required. Platform graphics/audio libraries and drivers are still required.

GX indexes loose file paths once at startup and preserves the old case-insensitive resource lookup on Linux and Windows. Restart after editing assets to refresh that index and the CPU/GPU caches. Symlinks and ambiguous case/hash names are rejected, so the folder remains portable. A legacy invalid-byte filename is escaped as `%XX`; lookup applies the same mapping to binary references without changing model data.

## Supported formats by role

| Role | Formats | Current contract |
|---|---|---|
| Playable resource layout | Loose files under `data/` | Required by game startup; case-insensitive lookup on Linux/Windows |
| Legacy import | SAH + SAF | Parsec-compatible importer/tools only; not a playable GX package |
| Interface, icons, portraits and minimaps | PNG | PNG required under `interface/`; `_dds.png` distinguishes legacy DDS atlases; optional `gxMP` preserves authored mipmaps |
| Native cursor images | PNG | `Cursor/01.png`–`07.png`, original dimensions, hotspot `(0,0)` |
| Runtime audio | Ogg Vorbis (`.ogg`) | Mono/stereo decoded to 44100 Hz stereo PCM for mixing; WAV decoder is an import/verification utility |
| World/model/effect textures | DDS, TGA, BMP, JPEG and PNG | Existing content decoder; DDS supports DXT1/DXT3/DXT5 and supported uncompressed layouts; not arbitrary texture encodings |
| Character/creature resources | 3DC, ANI, MLT, ALT, MON, ITM | Skeletal meshes, animation clips and model/action tables in the supported original layouts |
| Cloaks | PC, CTL, 3DC | Original cloth resources; complete physics parity remains in progress |
| World resources | WLD, DG, SMOD, 3DO, MANI, VANI, WTR | Field/dungeon layouts, geometry, placed/animated objects and water definitions |
| Effect resources | EFT, 3DE, SEFF | Supported effect libraries, animated meshes and weapon-enchantment recipes |
| Gameplay tables | SData | Supported table-specific schemas; active `Character/Skill.SData` must be EP5 |
| Text and map configuration | `sysmsg-uni.txt`, existing map metadata | Read by their dedicated loaders; not a general document/config importer |
| Server maps | SVMAP | GX Server data, separate from client WLD/Field assets |

### Helmet surfaces

The original character pass sets D3DCULL_NONE (`Game.exe` 0x41520c).
GX draws helmet shells from both sides, including their textured interior.
The part retains its MLT alpha-test mode and the original DDS.

Format support means the layouts implemented by the corresponding [image](../internal/data/image), [model](../internal/data/model), [world](../internal/data/world), [effect](../internal/data/effect) and [SData](../internal/data/sdata) parsers. It does not imply every episode/version or every effect branch is implemented.

```mermaid
flowchart LR
    A[Original SAH and SAF] --> B[Import and conversion]
    B --> C[Loose data folder]
    C --> D[PNG interface and cursors]
    C --> E[Ogg Vorbis audio]
    C --> F[Original model and world formats]
    D --> G[Shaiya GX on Linux or Windows]
    E --> G
    F --> G
```

## Import original data

The original SAH is a directory/index and SAF contains file payloads. The importer follows the Parsec-compatible reader in `internal/data/archive`; it does not modify the source archives.

With FFmpeg/libvorbis, optipng, the client build dependencies and a working OpenGL context, run from the GX checkout:

```sh
go run ./cmd/datamigrate \
  -source /path/to/original-archives \
  -output /path/to/staging/data \
  -report /path/to/private/conversion.json
```

The output must be a new directory. The JSON report stays outside the playable data and records source/output paths, hashes, sizes, conversions and reference updates. `-resume` reuses verified successful results only when both original archive hashes and quality match the report. Keep source archives outside the playable folder for reference/re-import. Never commit those archives, the loose game data or the private conversion report.

Stage and verify the new directory and executable together before replacing an installation. Original clients cannot use this GX-specific layout without their own loader/media changes.

## Interface PNG

Every image under `interface/` is converted from its actual decoded content, including files whose old extension did not match their header. TGA, DDS, JPEG and BMP become real PNG files; this is not an extension-only rename. Conversion is lossless relative to the existing decoded pixels, including dimensions, alpha and RGB under fully transparent pixels. PNG uses best-compression deflate and lossless optipng optimization; no palette quantization or resizing is applied.

Some original TGA and DDS atlases share a stem while containing different artwork or layouts. DDS-origin files therefore use `_dds.png`, e.g. `icon/01_dds.png`, while the TGA counterpart becomes `icon/01.png`. Runtime references use these explicit PNG names.

For compressed DDS, the importer reads back the same GPU-decoded pixels as the existing renderer. Authored mip levels are preserved as additional lossless PNG images in an optional private ancillary `gxMP` chunk. Standard PNG viewers display the base image normally; GX restores all authored levels and their original sampling behavior. Editing the base image requires regenerating or dropping its old mip chunk to avoid stale levels. Textures for 3D characters, world geometry and effects retain their original formats and GPU compression.

## Ogg Vorbis audio

All WAV resources become `.ogg` using libvorbis VBR quality 6, 44100 Hz, retaining mono or stereo where applicable. Vorbis is lossy; this quality setting targets a substantial disk reduction while preserving high audio quality. Original sound durations are checked against the converted decoded PCM, and every result is decoded by GX's actual Go decoder before acceptance.

Audio references in structured MON/WLD and similar resources are retargeted from `.wav` to `.ogg` using equal-length substitutions, preserving record lengths and offsets. Hardcoded recipes, UI sounds, login music and tests also request OGG. Game audio decodes only Ogg Vorbis, without FFmpeg or codec DLLs at runtime; the bounded decoder normalizes supported mono/stereo streams to the existing 44100 Hz stereo PCM mixer. The old WAV decoder remains an import/verification utility.

This changes resource storage and loading, not server rules, packet formats, combat timing or database contents.


## Cursors and skill data

`Cursor/01.png` through `07.png` contain the native cursor artwork. Conversion preserves every decoded RGBA pixel and each cursor's dimensions; all seven original hotspots are `(0,0)`, used directly by the client. CUR is retained only as an import/test decoder.

The active player catalog is `Character/Skill.SData` in EP5 layout, with
nine record slots per group. `Monster/NpcSkill.SData` holds NPC skills, and
`Npc/NpcQuest.SData` holds client NPC and quest presentation. GX Studio
reads these and displays the mapped server fields alongside client fields.
Use a matching loose data set when testing current source.


Map audio retains WLD point emitters and terrain walk-sound references. Music
fades with distance to each zone box; ambience uses spherical falloff, the
strongest emitter per asset and up to 24 audible ambient tracks. These mix rules
and gait-dependent footstep intervals follow Phoenix `scripts/map_audio.gd` and
`player_controller.gd`. OGG names resolve across `Sound/`, `Sound/back/` and
`Sound/Music/`, including legacy WAV references. Maps without authored music
zones retain their authored ambience. Music and Environment Sound settings control their
respective loops independently from combat/UI effects. Map decoding runs outside the render thread with at most two
jobs, while player creation and volume fades stay on the owner thread.


Outdoor ambience also retains the faction field bed explicitly listed in the
map's sound catalog (`bg_League of light_*` or `bg_Union of fury_*`). It plays
continuously on Environment Sound at the ambient base gain; local emitters add
their strongest in-range detail. This is a presentation adaptation: gating the
bed by its point placements left silent gaps, including the checked map 19
position. No music is invented for maps without authored music zones. Actual
ambient output has been captured from an isolated test sink with music and
effects muted, in addition to decoder/player-state checks.


WLD named areas retain their box, radius, both legacy strings, mode and trailing
word. Their position in the complete original record list is also the text ID:
mode-zero regions resolve it through the UTF-16 `world/<map>.txt` table. IDs are
sparse: mode-one/two records still occupy indices, and bitmap regions can occur
before later text labels. Do not compact text rows or derive IDs from the legacy
Korean caption. Mode-two captions are asset names resolved to PNG under
`interface/worldname/`.


The external font pack contains `fonts/ui.ttf`, `fonts/fonts.json` and unmodified
fallback faces/licenses under `fonts/unicode/`. The JSON coverage ranges allow
script faces to load only when a character requires them. The UI fonts share
parsed faces and rune resolution; shaped layouts and raster textures remain
bounded. A local `fonts/arial.ttf` takes precedence over the bundled Wine Tahoma.
Pair it with `fonts/arialbd.ttf` for 12 px emphasized tooltip headings.
`SHAIYA_FONT` and `SHAIYA_FONT_BOLD` can explicitly select the two local files.
The Unicode and color emoji fallbacks remain available with each face.
`tools/prepare_fonts.py` indexes an existing primary/fallback source; fontTools is
an offline import dependency, never a game runtime dependency. Keep the primary
font's license beside `ui.ttf`. No font data is embedded in the game executable.
Emoji ZWJ, flag, skin-tone and keycap clusters use the colour-emoji face, with a
separate colour raster layer that preserves its pixels under chat tinting.

The World Map uses `WorldMap.INI` (one section per region: `Country`, `Map`, `Click`, `Image`; converted from the original `WorldMap.cfg`), `interface/wm_map.png`, `wm_letter.png`,
`wm_warning.png`, `wm_button_m.png`, `wm_button_w.png`, region overlays
`wm_e<MapNum>.png`, titles `wm_f<MapNum>.png`, and the existing full-map frame,
arrow and `MiniMap/<MapNum>.png` previews. CFG coordinates refer to its original
512×512 content area. Country 0/1/2 means both/Light/Fury. The legacy
`worldmap-light.jpg`/`worldmap-dark.jpg` images are not used by this EP5 path.

Outdoor FLD geometry uses the original finest four-triangle fan per eight-unit
square, preserving corner/centre samples and reconstructing unused intermediate
heights. Mesh, height queries and shadow receivers share this surface. See
[terrain reverse engineering](terrain.md) for original addresses and validation.


### Standalone dungeon environments

DUN keeps the black background when no valid sky layers are defined. Its native
fog RGB, start and end fields are used when they contain a valid authored range.
The stock grey RGB 128/128/128 with range 150–200 is treated as an unset
placeholder. Changing the fog does not change the black backdrop.

The native DUN format lacks FLD's three sky filenames. GX reads an optional
`GXSK` footer immediately after the DUN fog fields. The four ASCII tag bytes
are followed by three NUL-padded 256-byte basenames in Sky, Cloud A, Cloud B
order. Any name can be empty. Names resolve under `data/Sky` and prefer their
converted PNG equivalents. Old files continue to work, and adding the footer
does not alter native bytes.

PS0107's dungeon branch at `0x451fd6` assigns `Sky_A1.bmp` to map 39 and
`sky_B1.bmp` to map 40, with two `clouds01_A1.tga` layers in both cases.
EP5.4's `0x452196` also checks ID 60, whose shipped and installed WLD is an
FLD with its own sky. The GX data uses the two DUN assignments, without
map checks in the renderer. The EP5.4 wedding DUN files 59 and 61 already
contain fog 200–300 and need no patch.

To prepare a loose data pack, build `cmd/assets` and run the following with a
data directory and two new output paths. Install the generated files as
`world/39.wld` and `world/40.wld` after backing up the originals.

```sh
assets -data "$SHAIYA_DATA" dungeon-sky world/39.wld 39.wld Sky_A1.bmp clouds01_A1.tga clouds01_A1.tga
assets -data "$SHAIYA_DATA" dungeon-sky world/40.wld 40.wld sky_B1.bmp clouds01_A1.tga clouds01_A1.tga
```

### Placed dungeon assets and monster skill metadata

FLD `Dungeons` placements reference full `world/dungeon/*.dg` assets. The client
applies each WLD basis and position to DG vertices, normals and collider bounds,
loads the authored lightmap pages, and preserves those textures during field
streaming. Placed DG parts use normal frustum/terrain visibility; they do not
switch an outdoor field to dungeon fog or dungeon-only occlusion.

`Monster/NpcSkill.SData` shares the nine-slot Skill.SData record format but owns
an independent one-based skill ID namespace. In the 116-byte serialized payload,
byte 3 is animation style, word 4 is the NPC visual selector and word 22 is skill
detail. Wire level 100 selects rank one of this table, never player rank one.
NPC icon numbers address the existing native skill icon atlases.

MON bone/effect pairs address individual emitters in `Effect/Monster.EFT`, not
sequence IDs. They use animated bone times creature-world transforms directly;
world/skill EFT basis conversion does not apply to bone attachments. MON attack
and death filenames address their own EFT libraries and remain separate from
NpcSkill recipient effects. Invalid placeholder names such as LOAD are ignored.


Native forge views also require `interface/rapisian.png` (enhancement) and
`interface/Remake3.png` (recreation); quest objectives use `radar_enemy.png`.
These are decoded from the corresponding original TGA files, with no new art.
They remain external game assets, not repository binaries. The ordinary full
asset import produces them alongside the other interface resources.


Item Mall runtime data uses the native binary `Cash.SData` table and SQL
ProductList on the server, with byte quantities up to 255. The asset importer
preserves Cash.SData. See [cash catalog](cash-catalog.md).

## Dungeon minimaps

`DungeonMiniMaps.INI` lists numbered image calibrations, one `[Map ID]` section
each with `ScaleX` and `ScaleZ` (pixels per world unit; the original
`Dungeon_MiniMap.cfg` stored them as `ImgPos_x/y` fixed-point values divided by
1000). The two scales are independent.
For an image of width W and height H, image U = world X × scaleX / W and
V = 1 − world Z × scaleY / H. They are not offsets or the dungeon's physical
size. Dungeon WLD files do not contain the field MapSize member.

GX applies these scales to the circular view, expanded map, player and
party/quest/obelisk markers, hover coordinates and GM map clicks. Field images
retain the existing WLD-size projection. Missing image/calibration is normal;
GX does not invent artwork or infer another map's image.

After extracting original files with Parsec, stage the loose data and run:

```sh
python3 tools/import_dungeon_minimaps.py \
  --source /path/to/core-extracted --data /path/to/staged-gx-data \
  --server-maps server/data/maps --report /path/to/import-report.json
```

The importer needs Pillow. It retains existing numbered PNGs byte for byte,
imports only missing playable-map artwork, verifies decoded RGBA pixels after
PNG conversion, and copies the native CFG intact. If source world files are
available, incompatible field/dungeon types or different dungeon layouts are
reported and skipped. The current import adds 43 images, preserves 16, leaves
50–54/62–64 without source images, and skips mismatched Core map 60. Resources
remain local and are not included in source history or a new public resource
release.

Generated `interface/MiniMap/map-sizes.json` maps numeric IDs to matching server
SVMAP world extents. It only converts dungeon image clicks into the existing
normalized GM request; server geometry, permissions and safe-destination checks
remain authoritative. It does not replace native CFG calibration. See the
[server connectivity audit](map-connectivity.md) for orphan maps and the map-60
content mismatch.

## Offline creature mesh normalization

GX keeps its current runtime `Parse3DC` layout. Some legacy creature resources,
including `Mob_Rend_01_A.3DC` and `Mob_Rend_01_B.3DC`, replace the leading zero
version word with a length-prefixed, zero-terminated texture filename. Normalize
those resource files during data preparation:

```sh
go run ./cmd/assets -data /path/to/loose-data \
  repair-creature-models /path/to/new-repair-output
```

This scans Monster/Npc 3DC files using the existing parser, leaves accepted
meshes alone, and stages only recognized rejected texture-prefix variants. It
validates the filename header, canonical mesh length and face indices, then
writes a zero version word followed by the original bone/vertex/face payload
without changing any payload byte. Unknown damage stops the operation before
any output is written. Output must be new and outside the source directory.

The output contains `data/` with canonical replacement files, `originals/` with
exact backups and `report.json` with counts and before/after/payload hashes.
Install the staged replacements with the client stopped; restart an active
client so its archive index and model cache reload. Original archive/reference
clients remain unchanged. The removed embedded texture name does not replace
MON material assignments; those continue selecting each creature's texture.

The current local audit covers 668 monster and 590 NPC meshes: exactly the two
Rend files require normalization. Their 35-bone payloads retain respectively
961 vertices/980 faces and 545 vertices/505 faces. They are reused by ten MON
visuals and 24 monster IDs, including Cloron Ogre 2045 and Cloron Guard 2046.
All 590 NPC meshes already parse; no NPC resources are rewritten. Repairs and
proprietary backups remain local, outside source history and release assets.

### Monster definition fields and visual scale

See [monster sizes](wiki/monster-sizes.md) for the scale table and original
combinations. See [mob properties research](mob-properties-research.md) for the complete
31-byte Monster.SData payload and the database/runtime authority split.
The byte traditionally called `Day` is the native client visual scale selector,
independent of combat `Size`. GX retains the full payload and applies the native
uniform model scale and scaled number anchor. AI and quest metadata in this file
do not replace authoritative server behavior.

### Lens flare

The PS107 client loads `Lensflare/flare0.jpg` through `flare6.jpg` at
`0x538cc0`. The outdoor sun pass at `0x539020` draws seven additive
layers using textures 4, 0, 1, and 5. Textures 2, 3, and 6 are loaded
by the original and have no draw call in that pass. The blacksmith
panels load texture 0 separately.

GX projects the original sun position and applies the original edge
attenuation and visibility fade. The occlusion ray checks map geometry
within the rendered world. Place opaque, losslessly converted PNGs in the loose
`data/Lensflare/` directory beside the client executable. GX also accepts
the original JPEGs when the corresponding PNGs are absent.

### Sky textures

Field skies prefer a PNG with the same basename as each texture named in the
WLD and fall back to the original filename. The compressed `login_sky.dds`
and `mt.dds` can stay in DDS to preserve GPU compression. The dome geometry
remains in `Sky/sky.3DO`.

Water surface height maps `WaterBumps1.tga` and `WaterBumps2.tga` belong in
`Entity/Water/`. GX derives the two surface normal maps from them and also
accepts the original root paths in older data packs.

### Skill catalog cleanup

`cmd/skillpatch` (`sdata.NormalizeSkillCatalog`) rewrites `Character/Skill.SData` without reserializing untouched bytes and server migration 7 applies the same decisions to the skill definitions. Dead levels (placeholders such as "Hell LordED" / "Not Available.") of skills that have working levels become empty slots; the chain mechanic is not used, so chain wording leaves the descriptions and the previous-skill word (record offset 52) of chain skills is cleared; Mana Shot and Stamina Shot lose their "enemy players only" sentence; Aggravation loses its time damage (int32 at offset 75); detail 25 becomes detail 21. The output is plain EP5, which the loader accepts. Back up the original catalog before installing the result.

The player catalog reserves the IDs of Interpretation (3), Backpack Upgrade (4), Double Warehouse Stone (234), and the equipped-item drop-protection effects (236, 253, 254) as empty groups. Run `skillpatch -retired-only -input Skill.SData -output Skill.updated.SData` to apply just this change to an existing catalog. The client also hides these IDs when it loads older data. Server migrations 28 and 31 remove their player definitions, learned rows, obsolete skill shortcuts and persisted player effects. Existing skill point balances and the separate NPC skill catalog are preserved.

Run `skillpatch -soul-burn-only -input Skill.SData -output Skill.updated.SData`
to enable Soul Burn in an existing player catalog. The patch enables rank 1
at ID 125 for the Priest and Oracle class slot, rewrites its description, and
clears the two obsolete higher records without shifting IDs. Server migration
29 applies the corresponding definitions. The two catalogs share `TypeEffect=6`,
`TypeDetail=122`, `SuccessType=1`, `SuccessValue=100`, and `KeepTime=0`.

Dead levels are recognised by their description ("Not Available.", "Skill not available currently", "Non-existant skill"), not by their fields: many keep copied effect data. The same tool now rewrites the descriptions listed in `internal/data/sdata/skill_text_fixes.csv` (id, level, current text, new text): typos and grammar, values that did not match the real definition, heals that add twice the caster's WIS, dispels that remove every harmful effect, sitting recovery every 5 seconds, and areas that use `ApplyRange` as the radius (the server applies it as is). A row applies when the slot still has its old text, is skipped when it already has the new one, and aborts on anything else. Server migrations 7 and 8 delete the same dead levels from the definitions.

`skillpatch -combat-debuff-text-only -input Skill.SData -output Skill.updated.SData` updates the Slump Shot and Confusion tooltips. Their text distinguishes penalties that affect both enemy players and monsters from STR and REC reductions that affect enemy players only. The patch reads the values, duration, and Confusion radius from each rank and changes only the descriptions. Full catalog normalization also applies this text update.

Use `skillpatch -npc-special-only -input NpcSkill.SData -output NpcSkill.updated.SData` on the EP5 monster catalog to update Porcine Party (100), Ice Stun (172), and Wide Death Sentence (344). The patch preserves the catalog's SEED encoding and all other records. Server migration 30 updates the corresponding entries in `PS_GameDefs_NpcSkills`.

`PrevSkillID` (offset 52) is a property any skill can have: the skill it names must be cooling down for this one to be usable, otherwise the server answers with SysMsg 315 ("A previous skill is required"). The tool restores the original values and appends "Can only be used while X is cooling down." to those descriptions; server migration 9 restores them in the definitions.


## Outdoor resources

Field skies use a dome and scrolling cloud layers. The loader prefers the
converted PNG for the WLD sky or cloud basename and can resolve legacy
references. Weather uses `Sky/sky_001.png`, with the legacy BMP as a fallback,
and the native rain and normal snow textures in `Effect/DDS/`. Rain audio is
`Sound/Music/Rain.ogg`. Sun flare textures prefer PNG, and water bump textures
are stored under `Entity/Water/`.

## Map names in SysMsg

The minimap, world-map previews, character selection, recorded points and
summon prompts read map names from `sysmsg-uni.txt`. The message index is
`11001 + MapID` for map IDs 0–199. For example, map 0 uses `11001`, map 99
uses `11100`, and map 199 uses `11200`. Missing names display as `Map N`.

The initial block covers IDs 0–99. To add it to older loose client data,
run the migration tool from the source checkout with the client closed:

```sh
python3 tools/migrate_map_names.py '/path/to/data/sysmsg-uni.txt' \
  --backup '/path/to/backups/sysmsg-before-map-names.txt'
```

The tool copies known names for installed numeric WLD maps and adds `Map N`
for unused slots. It removes the old map-name rows and preserves other entries,
including previously configured names in the new range. To name a future map, edit
or add its message in the same UTF-16LE format, then restart the client.

## Risk Sensor music

The original EP5 client passes the nearby-enemy flag to its music selector
at `0x450067`. The selector at `0x53A2A0` picks one of `bgm_battle01`,
`bgm_battle02`, and `bgm_battle03`. GX resolves the loose OGG tracks through
the music channel. It holds a single track for the encounter, fades back to
the map music when the alert clears, and keeps environmental ambience mixed
in both cases.

## Character voices

The original `VOICE_CHAR` flag gates the attack grunt (`0x4102C0`),
damage reaction (`0x40FF40`), and death cry (`0x418BA4`). Sounds vary by
race and sex. The attack and damage paths always request a voice on critical
results and sample ordinary attacks and hits. GX Sound Option uses
**Character Sound** for local and nearby player voices and **Risk Sensor**
for battle music. **Sound Effect** still controls weapon and spell audio.
**NPC Sound** is kept as a display-only option.

A loose pack needs the original character cues in `Sound/` as Ogg Vorbis.
Use the same import command below with the stems
`ch_ohat{m,w}{001,002,003,004}` and
`ch_{hum,huw,elm,elw,vim,viw,dem,dew}_{dam,die}`.

## Player progression and map-entry effects

The EP5 client uses sequence 0 of `Start.EFT` on fresh map entry (effect
dispatch at `0x4F085B` in the unpacked client). Ordinary view snapshots
do not replay it. Level-up uses `LevelUp.EFT` sequence 0 for the Alliance of
Light and 1 for the Union of Fury, with `ch_levup001.wav` (`0x4ECD9C`,
`0x4ECDBF`, and the deferred path at `0x4C0F47`). A successful skill
learn or upgrade uses sequence 0 of `Up.EFT` and `ui_skillnew001.wav`
(`0x50048C` and `0x50058F`). All three libraries retain their authored
emission times.

GX uses native `0x0201`, `0x0508` and `0x0209` for the local player.
Observers receive the existing level-up `0x051E` and GX `0x7F25` for
fresh map entry and skill progression. The GX packet contains the player
ID (u32) and cue (u8, 1 for entry, 2 for skill, or 3 for an administrative level change). It is sent after view
registration or a committed skill change, and respects stealth and view
membership. The nearby skill confirmation is a GX presentation addition.

For loose packs missing the original cues, import them from your own EP5
archive with the GX asset tool and FFmpeg. Run from the repository root
with `EP5_ARCHIVE_DIR` and `GX_DATA_DIR` set to the source archive directory
and the destination loose `data` directory.

```sh
for cue in ch_levup001 ui_skillnew001; do
    build/assets -data "$EP5_ARCHIVE_DIR" cat "Sound/$cue.wav" |
        ffmpeg -n -i pipe:0 -ar 44100 -c:a libvorbis -q:a 6 "$GX_DATA_DIR/Sound/$cue.ogg"
done
```


## Loading pictures

The random loading pool is every `random_loadingNN.png` in `interface/`. `MapScreens.INI`, beside `sysmsg-uni.txt`, lists `MapID=file` lines naming an interface picture for specific maps; invalid entries are ignored. Original EP5.4 only special-cased map 64 (`loading64.jpg`).
