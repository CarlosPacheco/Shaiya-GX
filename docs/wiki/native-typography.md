# Original client typography

The US EP4 client PS0107 is the primary typography reference for GX.
The EP5.4 client is useful for tracking later changes.

## EP4 PS0107 reference

This audit used the local US client with SHA-256
`5f045bcb4a2e29562ac79effc4108fbaa6404ab29258df02c4dfc6ed7e3401a6`
and its matching game archive. At `0x408c40` the client reads
`[LOGIN] VERSION` from its config. The PE default is locale 6 and the value
`ENGLISH` also selects 6 (`0x408d18–0x408d36`). The local US config has no
VERSION override. `VIETNAM` selects locale 3 (`0x408c8c–0x408caa`).

| Font slot | US EP4 English (locale 6) | EP4 Vietnam branch (locale 3) |
| --- | --- | --- |
| `+0x3a8` regular, used by ordinary chat and names | Arial 13, weight 400 | Arial 15, weight 400 |
| `+0x39c` small | Arial 12, weight 400 | Arial 12, weight 400 |
| `+0x3a0` emphasis | Arial 12, weight 700 | Arial 15, weight 700 |
| `+0x3ac` separate GDI handle | Arial 15, weight 400 | Arial 12, weight 400 |

The renderer starts at `0x5315c0`. The English branch at `0x5319f6`
passes the system face name `arial` to the constructors. Heights are positive
logical cell requests. The separate 15 px GDI handle is distinct from the
13 px draw slot used by the English chat and nameplates. The Vietnam branch
uses the name `Arial` from a separate string. Neither case loads a packaged
font file. The matching EP4 archive contains no TTF, OTF, TTC, FON or FNT files.

| Surface | EP4 PS0107 evidence |
| --- | --- |
| Chat history | `0x46f883`, `0x46fb43` and `0x46fc49` use the regular draw wrapper `0x52eed0` for ordinary lines. The layout multiplies the row index by 16 (`0x46f7a7`), and the text origin is X25 relative to the history. |
| Chat input | The selector calls the width-limited draw helper at `0x46fe7b → 0x5312c0`. Its label starts at X7/Y6 inside the control. The helper can select alternate font slots for styled text. |
| World names | The texture generator at `0x5373b0` calls the regular draw wrapper five times (`0x53751e–0x537573)). Four black passes sit at ±1 px along the axes around a single white center. |
| Window text | The wrappers around `0x52f030` and `0x52f070` use the emphasis and small slots. The ordinary path uses `+0x3a8`. |
| Floating combat text | `0x42747f` loads `interface/count.TGA`. The extracted 512×32 atlas is byte-identical to the EP5.4 version. |

The D3DX draw wrapper at `0x538c90` uses the ANSI entry with `DT_NOCLIP`.
The measurement routine uses the wide entry with `DT_CALCRECT`
(`0x52f380–0x52f393`). GX uses 13 px for chat and names, 12 px for small tooltip text, and
12 px for tooltip headings. Its nameplates use the native five drawing passes.
Place Arial regular and bold under `data/fonts/arial.ttf` and
`data/fonts/arialbd.ttf` to select the local EP4 faces. The packaged fallback is
Wine Tahoma. The UTF-8 shaping and emoji coverage apply to either setup.

## EP5.4 comparison

The secondary audit follows the EP5.4 `Game.exe` with SHA-256
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`
and its matching game archive. The font choice can vary between builds and
locale branches. The matching archive has no TTF, OTF, TTC, FON or FNT files.
The executable requests system faces by name.

### Faces and sizes

The renderer initializes the font objects at `0x544620`. The locale field
at `0x708598` starts at **6** in this executable's PE data. The separate
locale-3 branch requests Arial. Positive heights are requests for a logical
font cell. The actual raster metrics can depend on the font mapper.
Weights 400 and 700 are regular and bold.

| Font slot and use | Locale 6, Tahoma | Locale 3, Arial |
| --- | --- | --- |
| `+0x3a8` default draw slot, including routine chat and the outlined nameplate | 13, regular | 15, regular |
| `+0x39c` small slot, selected by text style 3 or a dedicated wrapper | 12, regular | 12, regular |
| `+0x3a0` emphasis slot, selected by text style 2 or a dedicated wrapper | 12, bold | 15, bold |
| `+0x3ac` separately created GDI font handle | 15, regular | 12, regular |

The two branches are at `0x544a5d` and `0x544912`. The constructors at
`0x54bdf0` and `0x5e8da9` build D3DX font objects with explicit family,
height and weight. The helper requests width 0, one mip level, no italic
and the default charset and quality. A separate `CreateFontA` call creates
the GDI handle. Other branches include SimSun, `courier`, Verdana and locale
cases that read a `FontName` setting. Those cases have their own sizes.

### How text is displayed

| Surface | Observed EP5.4 path | Current GX |
| --- | --- | --- |
| Chat history | `0x470b00–0x470fa5` uses the text wrappers at `0x5419e0` and `0x541aa0`. Ordinary lines use the default font slot. The original lays out the history on a 16 px row pitch, with channel color and text runs separate from the font. | A 13 px nominal cell for both histories. The primary face comes from the external font pack. History text has a dark shadow at `(+1,+1)`. |
| Chat entry and selector | The channel label path at `0x4711ab` calls `0x543d30` with a width constraint. That helper measures successive text prefixes and selects a font slot by style, falling back to the default. The selector text starts at `(+7,+6)` relative to its control. | Same 13 px face as the histories. The entry also supports UTF-8 shaping and cluster-aware editing. |
| Names above actors | The base outlined name texture is generated at `0x54a770`. At `0x54a8de–0x54a933` the routine draws four black copies at one pixel up, down, left and right of the white center. Monster name placement measures the width at `0x44c3ff`, halves it, and places the texture at the projected anchor. | Projects the nameplate and draws directly with the 13 px nominal face. The four outline offsets are one pixel. |
| Window and tooltip text | The common formatting and draw wrappers select existing font slots. `0x541b30` and `0x541b70` explicitly select the emphasis and small slots. `0x5418c0` selects the default slot. This lets individual controls choose their typography. | The general UI uses a 13 px nominal cell and tooltip text uses 12 px. Individual controls can also apply layout scaling and paint styles. |
| Floating combat numbers | `0x42709f` loads `interface/count.TGA`. Digits, Critical and Miss use authored regions of the texture. | The loose `count.png` atlas supplies those shapes. Changes to the UI font do not change the floating digits. |

The common draw wrapper `0x54be30` calls the D3DX font's ANSI draw entry,
passing the text color and `DT_NOCLIP`. Clipping and width limits can be
applied before that call. The measure path `0x541cb0` converts to wide
characters and invokes the wide entry with `DT_CALCRECT`. A UTF-16 message
resource and the final drawing API are separate parts of the pipeline.

GX loads `data/fonts/ui.ttf` through its external font pack. Its Latin-1
fast path uses Go OpenType with full hinting and rounded glyph advances.
Its Unicode path shapes text and lazily loads fallback faces, including
a separate color emoji layer. The nominal cell is used for layout.
Ascent, descent, ink bounds and advances can differ from those returned by GDI.

## Local metric probe

A standalone Win32 probe under Wine 11.21 asked GDI for regular Tahoma and
Arial using the native height convention. The GX probe duplicated the
Latin-1 font setup against the distributed Tahoma file. The sample was
`Cloron Dragon`.

| Backend and request | Cell height | Ascent / descent | Horizontal advance |
| --- | ---: | ---: | ---: |
| Wine GDI, Tahoma 12 | 12 | 10 / 2 | 65 px |
| Wine GDI, Tahoma 13 | 13 | 11 / 2 | 74 px |
| Wine GDI, Arial 13 | 13 | 10 / 3 | 67 px |
| Wine GDI, Arial 15 | 15 | 12 / 3 | 81 px |
| GX, Tahoma 12 | 12 | 10 / 3 | 64 px |
| GX, Tahoma 13 | 13 | 11 / 3 | 68 px |
| GX, Tahoma 15 | 15 | 13 / 3 | 79 px |

Wine's font mapper and rasterizer are specific to the test machine. The
probe measured the standalone APIs and the GX fast path. A matched
in-game frame on Windows remains necessary to establish pixel parity,
including coverage, hinting and D3D9 compositing.
