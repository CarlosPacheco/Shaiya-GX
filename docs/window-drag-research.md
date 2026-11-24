# Native window dragging and Status (T) sheet

Evidence from `Game.exe` EP5.4 (read-only Ghidra project; helpers
private Ghidra scripts).

## Drag rule

Every native frame embeds the same drag helper, `0x526f30` (clamp `0x526820`).
A frame object stores `x +4`, `y +8`, `width +0xC`, `height +0x10`, a dragging
flag `+0x14` and the grab offset `+0x18/+0x1C`. A left press inside
`x..x+width, y..y+height` with no other grab active starts the drag; the frame
follows the pointer and is clamped to the full screen (`0x70859c/0x7085a0`).
The drag rectangle is therefore the **whole frame**, not a title strip.

Each window's own handler tests its controls first and calls the helper last:
inventory `0x4f54e0` (item cells, bag tabs, close, preview), action bars
`0x4df520`/`0x4dd760` (slots, rotate/plus buttons), Status `0x5037f0`, skills
`0x500350`, party `0x50a930`. Pressing a plain button still starts a drag in the
original; GX keeps button presses on their control instead.

Frame sizes (width x height) recovered from constructors:

| Frame | Size | Address |
| --- | --- | --- |
| Inventory (`inven.tga`) | 256x486 | `0x4f4210` |
| Status, collapsed / expanded | 277x465 / 409x465 (`x-0x84`) | `0x502ab0` |
| Skills (`skill.tga`) | 330x488 | `0x500460` |
| Sub actions (`Sub.tga`) | 275x466 | `0x5077b0` |
| Player portrait (`main_stats_*`) | 213x114 | `0x509f10` |
| Target (`enemy_bar_*`) | 180x45 | `0x50a2e0` |
| Action bar (`main_slot_1.tga`) | 446x53, swapped when rotated | `0x4deac0` |
| Potion bar (`QuickPotion.tga`) | 89x53 | `0x4de160` |
| Revolver | 58x58 / 58x158 | `0x4e7be0`, `0x4e65c0` |
| Waypoint 255x360, gatekeeper 507, auction 740x600, party 160x52, raid 98x35/111x37/125x45, quest 252x510, mailbox 364x346, My Shop 452x445, guild windows 505x363 | | various |

GX now drags every personal and service frame from any empty part of it:
inventory, Status, Skills, Sub actions, portrait, warehouse, market, personal
market, exchange/duel, blacksmith, emblem, quest log, guild, guild directory,
guild house, NPC talk/quest, gatekeeper, PvP rewards, mall and recorded points.
Controls are detected through the window's own `Button` fields
(`ui.ButtonHit`); item cells, lists and scroll areas are excluded explicitly.
Still on title strips: the gift box, the raid configuration window, the full map
(whose body pans) and the auction house (its frame has no free body).

## Status (T) sheet

`0x5037f0` (input) and `0x503ee0` (draw), class `SHEET_STATUS`:

- The side tab `stats_button.tga` (27x102 hit) only **opens** the sheet. Once
  open it is neither drawn nor clickable.
- The open sheet is closed only by **Apply** (sends the six-value packet only if
  something was allocated, then folds) or **Cancel** (drops the draft), or by
  the close button, which also resets everything.
- Opening resets the draft and takes the free points from the character.
- The 12 +/- buttons have 22x22 hit boxes on a 20px pitch; the loop keeps the
  last button under the pointer.
- Each row of the expanded sheet shows **base + draft** (`0x503ee0`), white until
  points are placed and yellow after, not just the draft count. The +/- buttons
  are native repeat buttons (mode 5): they act on the press itself and again
  every 100 ms while held, for both directions, with no initial delay.
- A press inside the expanded graph panel (`x..x+0x9b`, `y+0x4e..y+0x12a`)
  neither drags nor clicks.
- Collapsed frame 277x465 at `(screen width-0x129, 0)`; expanded it grows 132
  pixels to the left, keeping the main panel where it was.

## Ladders

The WLD Objects loader `0x537200` derives each guide from the model header:
point A and B share the model centre X and minimum Z and sit at the lowest and
highest Y; both are transformed by the instance matrix. Entry from the bottom
places the root at A plus 0.9 along A to B (from the top, 0.9 down from B);
attach distance is 0.6 (`0x536fa0`). `0x43aea0` then moves along that line with
no sideways offset and no collision. The ivy models are zero-thickness planes,
so the character is drawn on the plane. GX keeps its 0.55 collision clearance
for simulation and pulls only the drawn model onto the guide, fading out over
0.55 units above the top so the landing step stays natural.
