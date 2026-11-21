# Original client typography

This audit follows the EP5.4 `Game.exe` with SHA-256
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`
and its matching game archive. The font choice can vary between builds and
locale branches. The matching archive has no TTF, OTF, TTC, FON or FNT files.
The executable requests system faces by name.

## Faces and sizes

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

## How text is displayed

| Surface | Observed EP5.4 path | Current GX |
| --- | --- | --- |
| Chat history | `0x470b00–0x470fa5` uses the text wrappers at `0x5419e0` and `0x541aa0`. Ordinary lines use the default font slot. The original lays out the history on a 16 px row pitch, with channel color and text runs separate from the font. | A dedicated 15 px nominal cell for both histories. The external primary face is Wine Tahoma. History text has a dark shadow at `(+1,+1)`. |
| Chat entry and selector | The channel label path at `0x4711ab` calls `0x543d30` with a width constraint. That helper measures successive text prefixes and selects a font slot by style, falling back to the default. The selector text starts at `(+7,+6)` relative to its control. | Same 15 px face as the histories. The entry also supports UTF-8 shaping and cluster-aware editing. |
| Names above actors | The base outlined name texture is generated at `0x54a770`. At `0x54a8de–0x54a933` the routine draws four black copies at one pixel up, down, left and right of the white center. Monster name placement measures the width at `0x44c3ff`, halves it, and places the texture at the projected anchor. | Projects the nameplate and draws directly with the 13 px nominal face. It adds a half-pixel foreground stroke and extends the right outline to 1.5 px. |
| Window and tooltip text | The common formatting and draw wrappers select existing font slots. `0x541b30` and `0x541b70` explicitly select the emphasis and small slots. `0x5418c0` selects the default slot. This lets individual controls choose their typography. | The general UI and tooltip faces both load Wine Tahoma at a 13 px nominal cell. Individual controls can also apply layout scaling and paint styles. |
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
| Wine GDI, Arial 15 | 15 | 12 / 3 | 81 px |
| GX, Tahoma 12 | 12 | 10 / 3 | 64 px |
| GX, Tahoma 13 | 13 | 11 / 3 | 68 px |
| GX, Tahoma 15 | 15 | 13 / 3 | 79 px |

Wine's font mapper and rasterizer are specific to the test machine. The
probe measured the standalone APIs and the GX fast path. A matched
in-game frame on Windows remains necessary to establish pixel parity,
including coverage, hinting and D3D9 compositing.
