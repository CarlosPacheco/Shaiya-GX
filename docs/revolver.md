# Revolver bar

The Revolver bar is the small three-cell control with the R marker. Its center
cell is the current action and its two dimmed neighbors show the preceding and
following entries. This bar has four independent pages of three to six
configurable cells. The Z/X potion shortcuts are a separate control.

| Control | Behavior |
| --- | --- |
| R or click the center | Use the current action |
| Alt + R or the reset button | Return to the first cell of the current page |
| Alt + 1–4 | Select a page and reset its cursor |
| Click the page number | Cycle pages |
| Arrow buttons | Step through the current sequence |
| Editor button | Show or hide the complete sequence for editing |
| + and − in the editor | Change the number of visible cells |
| Orientation button | Rotate around the bar’s center, with the frame kept inside the screen |

Drag a learned skill from K or a basic action from B to any of the three compact
cells or to an editor cell. You can also copy a shortcut from a regular action bar. Drag between editor
cells to swap their contents, or drag outside the Revolver panel to remove a
shortcut. Click an editor cell to select it. Drag the frame to move it.

GX stores pages, lengths and bindings per character and server in
`revolver.json` below the existing user configuration directory. Skills are
stored by ID and resolved to their current learned level and slot during use.
Position and orientation use the existing remembered UI layout. Combat uses
the same requests, lookup, reach and reuse checks as the normal action bars.

## Original client evidence

- EP5.4 `0x4e6a12` installs the window vtable. The draw method `0x4e6c80`
  renders the previous/current/next icons, with `0xff646464` on the outer icons.
  `0x4e65c0` reads position, orientation, current page and editor state from
  the INI. `0x4e7be0` handles the mouse and R, Alt + R and Alt + 1–4.
- `0x4e77d0` accepts learned skills and basic actions. Basic actions and
  cooldown feedback advance the cursor directly. Skill acknowledgments at
  `0x57bdd8`, `0x57e383` and `0x57fc54` check the caster and skill before
  advancing. GX also checks the originally requested page and cell before
  responding to a delayed result.
- `0x4e6540` resets the cursor, `0x4e6510` advances with wraparound and the
  native array uses a six-slot page stride. The stock plus handler allows a
  larger value even though the draw and lookup paths use that stride. GX
  bounds each page at six entries.
- `Revolver_View1` and `Revolver_View2` are the compact frames. The horizontal
  editor uses `RevolverAdd1–3` and the vertical editor uses
  `RevolverAdd5–7`. `RevolverAdd4` marks the selected cell. All nine TGAs
  are byte-identical in the inspected PS107 and EP5.4 archives. Import them
  into `data/interface/` as RGBA PNGs with their original names. GX can draw
  a functional fallback frame with older data packs.
