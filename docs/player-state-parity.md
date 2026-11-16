# Player resource, equipment and consumable parity

Current HP/MP/SP are absolute persisted values. Equipment, lapis, derived attributes
and country bonuses change maxima; querying stats never adds these bonuses again
to current pools. Effective clamping happens after equipment and restored buffs
load. Regeneration 0x0505 and authoritative 0x0521 both send current totals;
healing and spending apply actual deltas. Death clears displayed HP. Resurrection
and online level gains restore the effective maxima (or half for half revivals).

Weapon locomotion follows the original client selectors 0x4cb9a0 (run) and
0x4cb8c0 (ready), resolving the character's action ALT. Spear uses slots 54/48.
The ready stance lasts ten seconds after combat actions. Ordinary idle, walking,
sideways movement, swimming and mounts retain their existing animation routes;
invalid class/weapon placeholders fall back to ordinary locomotion.

## Consumable spell audit

The original server's ItemUse (0x4655e0), UseItemSkill (0x464f00) and BuffUser
(0x450ac0) resolve item Range/AttackTime as skill ID/level. BuffUser emits 0x0511
as well as applying the effect. GX now sends that release through the ordinary
local/remote spell animation, sound and effect path, and reserves the ordinary
movement pause. It does not require a learned skill or charge spell MP/SP for an
item cast. Item reuse groups and stronger-effect rejection remain authoritative.
A rejected use does not emit the spell or consume an item.

The supplied definitions contain 199 consumable skill references:

- 195 ordinary Special 0 buff references, including candies, remedies, charms and
  commercial skills.
- Warehouse Recall Rune 100/54, Special 35, skill 228/1: same buff spell route.
- EXP Reset Wheels 44/213 and 44/214, Special 50, skill 273/1 and 273/2:
  same spell route, restricted to maps 18/30 with at least half a level of EXP.
  Successful use resets EXP to the current level's start; a character-details
  snapshot synchronizes the result without changing level or learned skills.
- Snowball 44/147, skill 193/1, is a targeted attack (Effect 1), not a self buff;
  it is deliberately excluded from this buff path. Targeted snowball use remains
  unsupported.

The implementation resolves definitions dynamically, rather than maintaining an
item-name whitelist. Native addresses identify the locally inspected original
binaries recorded in [spell-fluidity-research.md](spell-fluidity-research.md).
These are static reverse-engineering findings, not a timing measurement of a
running original client/server.

Inventory, action-bar, warehouse, trade, duel, blacksmith and mall item icons show
positive counts, including one, only when the item definition permits stacking
(Count greater than one). Equipment and other non-stackable items have no count label. `/getitem type id` grants one by default;
explicit quantity validation and transactional grants retain their existing rules.
