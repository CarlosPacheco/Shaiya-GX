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

## EP5.4 controls and restoration scope

The follow-up used the same executable hash as this audit and the matching
`data.sah` / `data.saf`. Both original UI panels were extracted and inspected
locally. The matching EP4 PS0107 archive also contains the named atlases.

| Control | Original evidence | GX before restoration |
| --- | --- | --- |
| Help / Chat Mode | `chat_config1.tga`, opened at `0x477d07`. The panel is anchored at X380, Y=H−296. It shows Common, Whisper, Party, Guild and Trade, plus a separate Alpha checkbox. Its printed help demonstrates `! name message`, `@ message` and `$ message`. | GX handles these prefixes and offers an Area channel through its own dropdown. It has no native help panel. |
| Filter Option | `chat_config2.tga`, opened at `0x477c7a`. Eight independent checkboxes, drawn by `0x47088d` onward. Message filters are Damage, Acquire, Death and Notice. Conversation filters are Common, Party, Guild and Trade. | GX has no category filter control. |
| Filters and scrolling | `0x470b0c` consults `0x41cf40` while drawing stored messages. A filtered line is skipped during drawing. The filter lookup always permits styles such as 11 (whisper), 14 (area) and 15/16 (admin-related). | Scroll positions are independent in GX. Category filtering should operate on the rendered view and leave stored messages available when re-enabled. |
| Alpha | The sixth checkbox at `0x47837e` sets the setting persisted as `CHAT_BOX_VIEW` via `0x46e630` / `0x46e720`. Enabled keeps the frame visible. Disabled lets the background and its frame appear under the pointer (`0x4779ef`–`0x477a60`). Text rendering continues separately. | GX's optional ten-second auto-hide fades the history itself, including text. |
| Histories | `0x41f120` routes entries to separate histories, each capped at 100. The resize path clamps the pane setting to 0–20 (`0x477bdf`–`0x477c15`), with the first configured value read from `CHAT_SIZE0` and the second from `CHAT_SIZE1`. | GX restores both row counts and supports resizing. Its shared 300-entry buffer allows combat to evict conversation. |
| Position and frame | The INI routines read/write `CHAT_POS_X/Y`. The examined desktop draw and pointer paths (`0x47080a` and `0x4779fe`) explicitly anchor X to 0 and recompute Y from the viewport and row counts. The translucent plate and frame are drawn separately. | GX uses flat plates at the fixed left edge. The INI entries alone do not establish free dragging in this build. |
| Whisper composition | A separate recipient input and message input. The key handler at `0x478cf7` uses left Shift and DirectInput key `0x2c` (Z) to enter whisper mode. In that mode, Tab walks the stored recipient list cyclically via `0x470750`. The recipient popup at `0x47138d` is 131px wide with 15px rows. | GX can prefill a recipient through player context or social panels, and supports `!` / `/w`. It does not expose the recipient as a separate input or provide the native recipient shortcut/history. |
| Direct channel shortcuts | `0x478bf6`–`0x478cf0` selects Party with left Ctrl+Enter, Guild with left Alt+Enter, Trade with left Shift+Enter and Area with right Shift+Enter. The labels come from the native switch at `0x46e270`. | GX supports typed aliases and Enter. Existing bindings should be checked before assigning both sides of Shift. |
| Composer editing | GX currently delegates to the append/backspace-only `TextField`. The recovered native recipient/history controls establish separate editing state. | Caret navigation, selection, clipboard and IME need individual verification before claiming native behavior for each. |

### Filter lookup for implementation

The four message groups and four conversation groups below follow the recovered
switch at `0x41cf40` and checkbox ordering at `0x4708ba` and `0x4782be`.
The appended GX notices should retain their assigned styles when filtering.

| Checkbox | Styles routed through it |
| --- | --- |
| Damage | 0, 1, 5 |
| Acquire | 3, 4, 6, 7, 17 |
| Death | 2 |
| Notice | 8, 24–29, 33 |
| Common | 9, 30 |
| Party | 10 |
| Guild | 12 |
| Trade | 13, 31 |

Styles 25 and 26 pass through the Notice filter but still use the conversation
pane under the original routing, so filter groups and pane selection must
remain independent.

### Restored in GX

GX uses the original help and filter atlases when they are installed. Older
data packs get live text-and-checkbox panels in the same positions. The two
panel buttons use separate 22×19px hit boxes and unscaled crops from `chat_icon.png` at the left edge. Both popups
share the native X380 and H−296 anchor, clamped on smaller viewports. The
borderless histories leave a 2px visible seam. The composer opens on demand,
shares both horizontal edges with the lower history and sits 2px below it.
When vertical space shrinks, visible rows contract without overwriting the
saved row settings.

The sidebar uses separate tracks with no opaque rail, a 4px left inset, and
the existing mouse wheel, arrow repeat and track paging. The `0x52c430`
constructor calls at `0x47045a` and `0x4705a5` use the original 128×64
`scrollbar.tga`: arrows use the full 16×32 cells with 13×19 hitboxes and
the chat thumb joins the atlas body to a single bottom bevel in a 13×13 footprint.
It rests at the bottom with no overflow. The arrows keep their normal artwork
when there is nothing to scroll. The lower scrollbar reserves room for both
buttons inside the lower history. The two auxiliary icons come from `chat_icon`
and the popup checkmarks from the separate 16×16 `chat_check`. The installed
PNG pixel data for all three matched both local EP4 PS0107 and EP5.4 archives.
Both buttons end above the composer and remain independently clickable. Their pressed state retains the same menu icon. The recipient row starts
at X25 to clear both side controls while whispering. Its 131px popup uses
15px rows. The composer includes caret navigation, keyboard selection and
clipboard operations with UTF-8 limits. GLFW receives committed Unicode
text. Native IME preedit rendering is still unverified.

GX keeps 100 entries per pane. Filtering updates the display without deleting
messages. Alpha controls background visibility. The optional auto-hide
setting remains independent. Server delivery routes and cooldowns retain
their existing protocol behavior.

To use the original panel art with a loose data pack, convert
`interface/chat_config1.tga`, `interface/chat_config2.tga`, and
`interface/chat_icon.tga` from the local archive to matching PNG files in
`data/interface/`. The `assets png` command supports all three.

## Raid notice (GX addition over the original packet)

The original sends a raid notice with `0x1112` (`0x59ca70`, typed as `/raid text`
by a raid leader or sub-leader, `0x46fcf0`) and shows the reply (`0x596310`,
u32 sender, u8 length, text) on screen in violet as SysMsg 6606 `[Union] <b>`,
style `0x20` (`0x567980`). GX adds a Raid entry to the chat mode list for a
raid leader or sub-leader, removed when the role is lost, and a one-shot
shortcut, the apostrophe key, that writes a single notice without changing the
selected channel. The server delivers it to every raid member on any map and
ignores it from anyone else.
