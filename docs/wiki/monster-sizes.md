# Monster sizes

A monster's apparent size comes from its `Model` and `Day` fields in
`Monster/Monster.SData`. The model selects a visual in `Monster/monster.MON`.
`Day` selects a uniform scale for that visual. `Size` is a separate number
that sets the width of the monster's mouse target in world units. Read the
two values together as **visual scale × original model, with a specified
target width**.

## Visual scale and observed combinations

The labels describe the change relative to the same unscaled model. They are
plain-language references for editing the data. Two creatures with the same
`Day` can have very different actual dimensions.

| `Day` | Model multiplier | Relative size | `Size` values in the original EP5.4 data |
| ---: | ---: | --- | --- |
| 6 | 0.4× | Very small | 1, 2 |
| 0 | 0.6× | Small | 1, 2, 3, 4, 5 |
| 1 | 1.0× | Original size | 0, 1, 2, 3, 4, 5, 8 |
| 2 | 1.3× | Slightly enlarged | 1, 2, 3, 4, 5 |
| 3 | 1.6× | Large | 1, 2, 3, 4, 5 |
| 4 | 2.0× | Very large | 1, 2, 3, 4 |
| 5 | 5.0× | Giant | 2, 3, 4, 5 |
| 7–255 | 1.0× | Original size by fallback | 2 at code 7 |

The last column lists the 33 distinct pairs in the audited original catalog
among entries with positive HP, level and MON height. It describes what the
original data uses. The client interprets every other pair with the same
rules. The two fields are unsigned bytes.

## Target width, height and combat reach

- The vertical mouse target faces the camera horizontally. Its full width is
  `Size` and it extends `Size / 2` to either side of the monster's origin.
  `Size=1` is a narrow target, 2 is twice as wide, 3–5 are progressively
  wider, and 8 is very wide in the audited catalog. These are descriptive
  references. `Size=0` produces a zero-width mouse target.
- The target extends from the monster's rendered origin up to
  `MON.Height × scale + 0.2`. The nameplate uses that same height above the
  origin. This height comes from MON metadata. A mesh or animation can
  extend beyond the rectangle.
- GX World uses its own `PS_GameDefs_Mobs.Size` for combat reach. For example,
  targeted skill launch reach includes the target's server `Size`, and a
  monster's attack reach includes its own. GX uses its client catalog to
  estimate skill reach while approaching. Neither value rescales the
  model or defines terrain collision.

## Examples from the original data

| Mob ID and name | Model | `Day / Size` | Interpretation |
| --- | ---: | --- | --- |
| 256 · Curly Troll | 132 | 0 / 1 | Small troll with a one-unit target. |
| 257 · Moe Troll | 132 | 1 / 1 | Same model at original scale with the same width. |
| 1920 · Santa Troll | 132 | 1 / 3 | Same scale as Moe, three times the target width. |
| 1922 · Santa Troll | 132 | 3 / 5 | Large version of the same model with a five-unit target. |
| 356 · Maitreyan Beholder | 86 | 6 / 1 | Very small relative to its base model. Its target is one unit wide. |
| 872 · Cloron Dragon | 151 | 0 / 4 | Small *relative to the dragon model*. Its MON height is about 34.70, so its target still extends about 21.02 units above its origin. |
| 1627 · Stable Erde Tatoramo | 97 | 5 / 4 | Giant scale with the same target width as Cloron Dragon. Its different base model gives it a target height of about 10.06. |
| 2294 · Bacaras | 98 | 7 / 2 | Original model scale via fallback, with a two-unit target. |

## Verification

The EP5.4 `Game.exe` (SHA-256
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`)
reads `Day` at runtime record offset `+0x10` in `0x431869`, applies the
same multiplier on all three axes, and stores the scaled height plus 0.2 at
`0x4318e4`. It copies the unsigned `Size` from `+0x11` at
`0x432e1e–0x432e36`. The monster picker at `0x43b8c0` uses
`−Size/2` and `+Size/2` and ray-tests the two triangles of that rectangle.

GX matches those dimension rules. Comparing the original archive and the GX
loose data found 1,516 eligible shared mob IDs with identical model indices,
scale codes, target widths and MON heights. The
distributed SQLite seed also agreed on `Size` for those IDs. Render tests
covered all authored scale selectors, the fallback and Cloron Dragon picking.
See the [Monster.SData field inventory](../mob-properties-research.md) for the
on-disk offsets and the client/server split.
