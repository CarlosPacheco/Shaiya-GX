# Chat audit against EP5.4

This audit distinguishes recovered native behavior from GX's chosen flat chat
presentation. Reference: original `Game.exe`, SHA-256
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`.
Addresses below refer to that executable. Original binaries, extracted assets,
disassembly and private captures are not distributed in this repository.

## Geometry and text

`0x470120` initializes both configured row counts to five. `0x4707d0`
recomputes the layout every draw. With viewport height H, upper row setting U
and lower row setting L, its inner frames are:

- Upper height: `(U + 2) * 16`.
- Lower height: `(L + 4) * 16`.
- Upper Y: `H - upperHeight - lowerHeight - 92`.
- Lower Y: `upperY + upperHeight + 8`.
- History text X: 25; line pitch: 16.

At 768px height and settings 5/5, these yield upper Y420/height112 and lower
Y540/height144. The old GX geometry gave those same frame sizes but deducted
inner margins before calculating row capacity, losing an entire line in each
pane. A partially filled pane then anchored against that reduced row count,
leaving the newest line unnecessarily high.

GX now displays seven/nine complete cells and anchors the newest text at
`frameBottom - fontHeight - 1`. At the current 13px font height this moves the
newest line down 14px, independently of how many lines exist. The oldest visible
cell, descenders and one-pixel shadow stay clipped inside the flat rectangle.
Native text origins are `upperBottom - 11` and `lowerBottom - 10`; its four-pixel
ornamental frame extends beyond the inner rectangle. GX deliberately reserves
the whole font cell inside its borderless plate instead of drawing past it.

| Entry detail | Previous GX | Native measurement / corrected GX |
| --- | --- | --- |
| Entry top | `H - 82` | `H - 76` |
| Selector | 64 x 22 | 64 x 22 |
| Visible input strip | 286 x 22, reconstructed patches | 292 x 22, unscaled authored strip |
| Combined visible width | 350 | 356 |
| Input text origin | X70, entry Y+2 | X69, entry Y+6 |
| Selector text origin | X5, entry Y+3, scaled font | X7, entry Y+6, regular chat font |
| Right-hand atlas icon | Cropped out by reconstruction | Preserved |
| Caret | Always-visible appended underscore | Blinking vertical caret, independent of text width |

The native draw submits a 512 x 32 atlas quad starting at X64; only 292 x 22
pixels of that strip are visible. Treating the full submitted quad as a visible
input width would be another error. Pointer bounds now follow the actual 356px
combined row; the histories remain 350px wide. The gap between the lower history
and the input is eight pixels.

The current external primary font is Tahoma, loaded at a 13px cell height, with
configured Unicode/emoji fallbacks. Old documentation claiming embedded Open
Sans 12px was obsolete. Font shape across GDI/D3DX and FreeType still needs a
controlled native raster comparison; matching a cell size does not prove
pixel-identical glyphs.

## Palette and formatting

`0x41cfe0` confirms the existing channel RGB values: Normal D5D4D3, Party
A8F3A0, Whisper F3A0A0, Guild EEA4ED, Trade F3DEA0 and Area FF3000. These did not
need replacement. Routing is separate: `0x41f120` sends gameplay styles to the
upper history and conversation styles to the lower one.

`0x41d4a0` expands native message placeholders. For `<hp>`, it calls `0x578230`,
which retrieves resource names from SysMsg 416/417/418 (HP/SP/MP). Therefore
omitting HP from damage is an explicitly requested GX presentation change,
not evidence that the original executable lacks the label. GX suppresses only
that HP damage suffix; healing and MP/SP labels remain localized. Numeric
combat amounts use a yellow run and reset to the surrounding notice color.
This emphasis is also a GX choice, rather than an asserted native numeric tint.
Player conversation remains literal and cannot activate trusted notice markup.

## Gameplay notice coverage

Decoding an opcode and showing floating damage never proved that its chat
notification existed. The audit traced packet readers through the native
session vtable at `0x6bbd68`, including the target-side handlers.

| Event | Native evidence / SysMsg IDs | GX result |
| --- | --- | --- |
| Own/incoming basic damage, critical and miss | 1000–1005 | Already connected; combat numbers now have separate yellow emphasis |
| Skill use / area announcement | `0x57db20`, `0x579390`; 1100/1300 | Missing use notices connected; result 4 is an announcement, not a failure |
| Targeted damage and healing | 1101–1104, 1301–1303 | Existing paths retained; NPC definitions resolved in the correct level-100 namespace |
| Zero-damage skill effect / failure | 1106/1108, 1305/1306 | Existing paths retained |
| Extra damage `0x0514` / `0x051A` | `0x578d70` / `0x5798a0`; 1105, 1304, 1310 | Previously floating numbers only; relevant notices now connected |
| User periodic `0x050F` | `0x577b90`; 1103, 1206, 1211, 1310 | Previously no chat; self and party pulses now connected |
| Mob periodic `0x0607` | `0x577f30` | Native handler emits floating numbers without a chat call; no invented notice added |
| Reflection `0x0515` / `0x051B` | `0x5790f0` / `0x579aa0` | These handlers also emit floating numbers without chat; no invented notice added |
| Buff add / ordinary removal | `0x580880` / `0x580610` | No blanket gained/expired messages invented for every snapshot or removal |
| Special continuous resurrection rune expiry | `0x580610`, detail 46; 837 | Still requires the native conditional branch; generic removal must not emit it |
| XP, gold/items, learned skills, level, party joins/invites | 50/52/53/59/111/204/205/207/217/218 etc. | Existing confirmed-event paths retained |

GX currently orders non-HP healing tick values differently from debuff tick
values. Periodic chat selects their meaning from the skill effect; it does not
blindly call both positions MP/SP. No gameplay resource or packet layout was
changed by these presentation fixes. Unknown skill definitions and unrelated
players do not generate guessed periodic messages.

## Remaining differences and limits

These are recorded differences, not a declaration of full native parity:

- The flat plates and optional ten-second auto-hide remain chosen GX behavior.
  Native TipBox borders, Alpha controls and filter/help panels are not restored
  by this correction. The current native background color and eight-pixel gap
  are preserved.
- Native histories are distinct lists with a count capped at 100 each
  (`0x41f120`). GX retains its chosen shared 300-message buffer. A heavy combat
  stream can therefore evict conversation history, even with independent pane
  scroll positions. This is a storage-policy difference, not missing packets.
- Native `0x41cf40` filters different gameplay/chat categories independently.
  GX's retired filter preferences do not currently provide that behavior.
- The current six-channel picker is a GX arrangement. Native picker rows use
  15px pitch and native channel numbering includes conditional channels; GX's
  dropdown still uses 22px rows. Native whisper has a separate recipient row;
  GX uses its existing one-row recipient syntax. Raid remains outside the picker.
- Input editing still appends/deletes at the end, with history navigation.
  Cursor-position editing, selection, clipboard and IME composition need their
  own implementation/validation; drawing the authored IME/language icon does
  not implement those platform services.
- Full third-party party/raid combat narration, every skill refusal code and
  specialized service/GM notice are not covered merely by connecting the
  self/incoming paths above. Follow-up work should trace each callback and its
  target relevance rather than displaying every nearby event indiscriminately.
- Native resource names, channel prefixes and wording remain data-dependent.
  Tests with fabricated strings do not establish native localization coverage.

## Verification

Focused unit checks cover resource-specific notices, yellow-run reset, result-4
announcements, extra/periodic effects, localized labels, party relevance and
literal player text. OpenGL tests compare empty and populated framebuffers,
require the first line in each pane's lowest cell, check descenders and clipping,
and capture normal/whisper/picker states. The isolated N2 suite verifies live
chat echo, whisper/error delivery, self-cast timing and gameplay regressions
without using the personal database. Visual captures are GX regression evidence,
not a replacement for a matched native/live screenshot comparison.

## Font measurement follow-up

Renderer initialization `0x544620` selects font creation by locale, through
helper `0x54bdf0`. English (locale3) creates the regular
chat font at **15px, weight400, Arial**, and the optional large font belongs to
other locale branches. GX was reusing its 13px tooltip font. Chat now owns a
dedicated 15px font loaded from the existing external font pack; tooltips and
other panels retain their sizes. The external primary face remains Tahoma, so
matching the measured cell height does not claim identical Arial glyph raster.
The right-hand language icon has subsequently been removed by user request,
while preserving the measured entry width/position. This supersedes the 13px
and icon-preserved statements in the historical audit above.

Item-acquisition notice 52 includes the acquired item's atlas icon immediately
before its name. The inline square follows chat font height, participates in the
316px line wrap and history fade, and uses the same icon and elemental badge
renderer as inventory items. Only confirmed item-add packets attach structured
item metadata; player text cannot inject icons or trusted color markup.
