# Weapon enchantments

Weapon glows read `Effect/Weapon.Seff` directly from the user's original archive.
No converted Phoenix assets or game textures are bundled. Phoenix's weapon
enchantment documentation and converter provide auxiliary references; native
EP5 routines `0x530790`, `0x530c70` and `0x4cba90` determine selection, parsing
and weapon normalization.

There are six visual tiers for each of fifteen weapon families. Levels below
+5 have no glow; +11 through +49 reuse +10; values at or above 50 represent
armor encoding and are excluded. Selection ignores the individual item model:

`record = 6 * (family - 1) + min(enchant, 10) - 4`

| Family | Weapon |
|---|---|
| 1 | One-handed sword |
| 2 | Two-handed sword |
| 3 | One-handed axe |
| 4 | Two-handed axe |
| 5 | Dual swords |
| 6 | Spear |
| 7 | One-handed mace |
| 8 | Two-handed hammer |
| 9 | Reverse dagger |
| 10 | Dagger |
| 11 | Javelin |
| 12 | Staff |
| 13 | Bow |
| 14 | Crossbow |
| 15 | Claws |

Alternate item types 45–65 normalize to these families. Shields do not emit.
Dual swords and claws attach independent emitters to both weapon bindings.
Each particle follows the current weapon pose, including its item-table binding;
it does not remain behind at a captured spell position.

The parser accepts SEFF versions 5–7, bounded UTF-16 filenames and complete
emitter fields. Unsupported emitters are reported rather than silently
substituted. The original local catalogue has 162 records; its ninety weapon
records contain 186 layers using eighteen textures. Remaining records are not
selected as weapon enchantments.

The renderer implements the point, radial, elongated and rotating emitter shapes
used by those records, their authored colors, sizes, speed, lifetime and alpha
ramp/fade. Native routines `0x52ebd0`, `0x52fa30`, `0x52fe60` and `0x5300a0`
provide the particle reference. Shared recipes and textures are cached; each
layer uses one instanced additive draw. A 2,000-particle cap per layer follows
the native pool. Effects respect the existing skill-effects visibility setting
and are omitted from shadow passes.

The local equipped level comes from the server-confirmed item's craft string.
Other players use the enchant byte already carried by native appearance
packets. Equipment moves and enchant acknowledgements rebuild the affected
appearance, including changes that preserve the same item type and model.
Success, failure, costs and consumption remain server-authoritative. Character
selection packets lack enchant metadata, so an unknown level is not invented.

GX Server's initial `0x0303` and later `0x0507` equipment updates now both
supply the raw craft level. Remote players therefore show their equipped glow
from their first appearance, including after entering visibility range. This
server correction is verified before any equipment move in the two-client test.

Validation covers all ninety recipes against original assets, both dual
bindings, level boundaries, visibility, actual rendered pixels and an eighty
weapon draw benchmark. The isolated two-client GX Server test exercises enchanting
an equipped +5 weapon, server acknowledgement and remote appearance updates,
then unequipping and re-equipping it. Captures and original data stay local.

Random births use stable analytical streams rather than the original random
generator. Exact particle-by-particle visual equivalence and all camera/animation
states still need comparison against the original running client. Phoenix's
additional dynamic lights are not part of this implementation.

Rigid equipment now follows Phoenix's recovered wield profiles for all 16
character prefixes: bows and crossbows use the ranged hand; javelins use the
ordinary weapon hand. Dual secondary reversal is limited to Human/DeathEater
rigs, and the corrected Elf male mage shield uses bone 13. Authored IT2 bindings
still take precedence over the legacy ITM fallback table.

## Innate item glows

Innate glows are independent of enchantment and elemental combat attributes.
The third signed integer of an ITM/IT2 model record selects an original EFT
library: `-1` disables the glow, `0` selects `item_L.EFT` (blue), `1` selects
`item_F.EFT` (fire), `2` selects `item_E.EFT`, and `3` selects `item_W.EFT`.
These are not Weapon.Seff recipes. The selected emitter is the normalized
weapon family minus one; each original library contains fifteen emitters and
no sequences. Unknown selectors are ignored, without substituting another glow.

Native EP5.4 sources: `0x556fd8` reads the selector into record offset `+0x20`;
`0x4282fb–0x4283b6` selects catalogue IDs 13/14/218/219;
`0x42a145–0x42a19c` binds those IDs to the four EFT filenames;
`0x40e5b0` bounds-checks the emitter index. Shields have selectors in their
ITM records, but their family lies outside these original libraries and emits
nothing. Do not manufacture a shield recipe from a weapon family.

The glow attaches to the animated item binding, including both dual weapons,
and coexists with enchantment particles. Native `0x42947b` distinguishes fire
(catalogue 14), whose births are transformed into world space by `0x55de50`,
from the other three glows, simulated locally by `0x55d270`. Those local
particles follow the current item matrix at draw time; transforming their
births into world space incorrectly leaves them behind during movement. It uses the existing EFT runtime and
shared resources, advances once across camera passes, and respects concealment,
weapon-effects visibility and shadow exclusion. Item.SData model selection is
sufficient for local, remote and preview characters; no new packet is needed.

The selector formerly named Alpha was incorrectly used for mesh blending.
The original material flag is instead the last integer in each record
(native `+0x24`, consumed by `0x42909b`/`0x4298a7`): zero enables alpha testing,
nonzero disables it. Glow choice no longer changes mesh transparency.

The rigid equipment pass draws clockwise front faces for individual and
instanced characters. EP5.4 restores Direct3D back-face culling at
`0x415bd5` before the attachment draw at `0x415da7`. Many original meshes,
including claws, swords and bows, have coincident triangles facing opposite
directions. Culling keeps the back surface from competing with the visible
surface while the character moves.

Linux GPU checks load all sixty family/glow combinations, render each of the
four real staff variants at +0, check dual innate/enchantment coexistence and
visibility/update rules, and retain the ninety enchantment-recipe regression.
Native particle random streams and exact frame-by-frame original-client visual
comparison remain outside these assertions.
