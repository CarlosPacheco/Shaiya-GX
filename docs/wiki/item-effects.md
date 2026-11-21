# Item effects and native field meanings

Item names and descriptions explain content to players; they do not implement an
effect. The world server loads SQL item and skill definitions and executes the
supported behavior. Client Item.SData provides names, artwork, descriptions and
presentation fields. Editing only a description cannot grant a stat or skill.

This page describes current GX development behavior. Historical EP5 or custom
server documentation can describe different rules. See [Blacksmith](blacksmith.md)
for exact forge chances, costs, consumption and safety guarantees.

## Definition versus inventory instance

A definition is identified by `(Type, TypeID)`. A packed identifier is
`Type * 1000 + TypeID`; for example Type 25, ID 1 is 25001. An inventory instance
adds its own unique ID, owner, bag/slot, quantity, quality, six gem entries and
Craftname. Multiple copies share a definition but can have different bonuses.

The server combines equipped instances, their definitions, linked lapis,
recreation attributes and enchantment. Carried equipment contributes no equipped
bonuses. Stacks are limited to 255 and the definition may impose a smaller limit.
Stored quality is retained for compatibility; GX does not implement durability
loss or reduce equipment stats because of old quality values.

## Effect icons in item tooltips

Supported attack/debuff lapis show an **Effects** row in silver with the called
NpcSkill icon. Equipment shows the distinct skills from its active linked lapis;
duplicate skills share one icon. Icons resolve the independent NPC definitions,
never a same-numbered player skill. Recovery lapis
do not add an effect icon.

## Experience consumables

The server reads each active skill's ability 35 value as its bonus percentage.
Distinct spells add their bonuses. For example, +100%, +50% and +50% give
300% of the original monster EXP. A repeated cast of the same skill refreshes
its buff according to the usual rank rules. Quest EXP is awarded separately.

Increase EXP and the Sun EXP Stones each give +50%. Blue Dragon, White Tiger
and Red Phoenix each give +20% monster EXP as their sole benefit. Dedicated
EXP-loss protection and mob-gold skills use their own abilities.

## Permanent equipment bonuses

| SQL field | Current use |
|---|---|
| Effect1 / Effect2 | Attack and its added range for weapons, accessories and lapis |
| Effect3 / Effect4 | Defense and resistance for non-weapons |
| ConstStr, ConstDex, ConstRec | Added STR, DEX and REC |
| ConstInt, ConstWis, ConstLuc | Added INT, WIS and LUC |
| ConstHP, ConstMP, ConstSP | Added maximum resources on equipment; see consumables and recovery lapis exceptions below |
| Exp | Flat damage absorption while equipped, not experience gained |
| Attrib | Native element; weapon and upper-body equipment determine offensive/defensive element respectively |
| Slot | Equipment lapis capacity; on a lapis, capacity consumed |

Negative values in the signed stat fields do not subtract stats through the
current equipment accumulation path: it uses nonnegative contributions.
`Exp` is an unsigned byte (0–255 per definition). Equipped items add their
values together and stack with armor enchantment absorption. Moving an item
back into a bag removes its contribution. The client reads the same byte from
`Item.SData` and shows nonzero native absorption on equipment and lapis
tooltips using SysMsg 485. An equipment tooltip shows one total absorption value: native `Exp` plus
active linked lapis plus eligible armor enchantment. The enchantment level
appears separately in the item name. For example, armor enchanted to [20]
(100 absorption) with a 60-absorption lapis shows `Absorption 160`.
Totals are not limited to 255. Keep server `Items.Exp` and client `Item.SData` synchronized when
editing definitions; this is a flat amount, not a percentage or life steal.

Recreation adds its separately stored values. Weapon enchantment adds its total
attack bonus for the current level; armor enchantment adds flat absorption.
The attack and defense enchant tables in `PS_GameDefs` store the total bonus at each level.

The effective combat totals are calculated on the server. Tooltip descriptions
are not a substitute for confirmed stat packets. In particular, a numeric native
field can have different meaning in a consumable, a gem and a piece of armor.

## Linked lapis: passive and triggered effects

Lapis use Type 30. Linked gem entries store their TypeID and resolve the matching
Type 30 definition. Their ordinary stat bonuses contribute while the containing
item is equipped. Range, AttackTime and Speed on a gem additionally modify reach, attack speed
and movement speed through the equipment path.

Special 16 is the recovery-gem exception: ConstHP/SP/MP contribute to standing and
sitting recovery rather than increasing maximum pools. Other supported passive
stats on that gem continue to apply, including `Exp` absorption.

An active linked lapis adds its `Items.Exp` to equipment absorption, just as
`CUser::GemEquipmentAdd` (`0x004560c0`) did in the original. Multiple active
lapis stack with native equipment absorption and armor enchantment. Extraction
or unequipping removes their contribution; an inactive elemental lapis adds
no absorption.

A nonzero `Special` selects the **NpcSkill** with that ID, except Special 16
(the recovery case). NpcSkills have their own namespace: the shared SQL table
stores rank one as **skill level 100**, and the client renders it from
`Monster/NpcSkill.SData`. A same-numbered player Skill is never a fallback.
Only attack/debuff effects enter this trigger path. SuccessType/SuccessValue
of the called NpcSkill determine success; there is no additional lapis roll.
The current Lv6 effects use 8%, and Lv7 use 13% except Slumber Lv7 at 15%.
Changing a definition changes the chance after restarting world.

All 26 Lv6/Lv7 debuff lapis are supported: stun, movement stop, blindness,
silence, movement/attack-speed reduction, defense/attack reduction, HP/SP/MP
damage over time, accuracy reduction and sleep. Duration, affected categories
and numeric values come from the called NpcSkill, rather than item-description
text. Silence also blocks healing/buff spells of the disabled category and
interrupts matching prepared casts. Expiry removes the restriction; only actual
HP damage wakes sleep. NPC and player effects with the same ID can coexist.
Periodic damage debits the relevant pools on players and mobs.

Endure's durability bonus remains unused because GX has no durability loss.
The 15 catalog Anti-Lapis use their Special value as the exact NpcSkill ID
they block while equipped. They block every hit from that NPC skill. Other
IDs still resolve normally, including skills with the same StateType and
player spells with the same numeric ID. For example, Anti-Sleep blocks the
Sleep Lapis proc and leaves Hypnosis eligible to hit. Lapis and equipment
tooltips list the matching skill icons in a separate **Immune to** row. The
offensive counterpart supplies the icon for legacy records without a skill icon.

After a landed hit, current trigger routing is:

| Situation | Gem effects considered |
|---|---|
| Player attacks player | Attacker weapon gems target defender; defender armor gems target attacker |
| Player attacks mob | Attacker weapon gems target mob |
| Mob attacks player | Defender armor gems target mob |

A gem-triggered attack cannot recursively trigger another gem attack. Both
participants must still be alive. Armor scanning follows native order: shield,
helmet, upper armor, gloves, lower armor, shoes, and stops at the first empty
slot. This last detail is implementation behavior, not a recommendation to
leave equipment slots empty.

Direct incoming mob skills now apply the same flat absorption as normal mob
attacks, with a minimum of one HP damage for a positive hit. Pure resource loss
and misses do not create HP damage. Periodic damage remains exempt. Skill
result packets and drain healing use the post-absorption HP amount, including
PvP skill drains.

Source: `server/internal/world/itemoption.go`, `character.go`, `combat.go`,
`pvpskill.go` and `playereffects.go`.

## Elements

See [Elements](elements.md) for the complete original matrix, tier semantics,
spells, player/monster paths, temporary effects and verification evidence.

An intrinsic equipment element always wins. Otherwise the first elemental
lapis in socket order supplies the element. Later elemental lapis stay linked
but inactive and gray; non-elemental lapis remain active. Invalid element values
are ignored. This is GX's explicit priority override; different elements no
longer cancel the equipment to neutral. Element text or a colored icon does not bypass that calculation.

Source: `itemElement` in `server/internal/world/character.go`. Damage interaction
is handled by combat rules; this page does not invent percentages from item
names.

## Healing and item-cast buffs

Supported consumables must be in a valid carried slot and have remaining count.
Ordinary healing uses positive ConstHP/MP/SP as flat restoration. Special 17
interprets those values as percentages of the effective maximum resource. The
server caps the resulting resources through its normal healing path and consumes
one unit for a valid use.

A Special 0 item with nonzero Range and AttackTime can instead cast a skill:
Range selects SkillID and AttackTime selects skill level. The definition must
resolve to a buff skill. Buff duration and abilities come from that skill's row,
not from an invented universal potion multiplier.

Healing and item-cast buffs use ReqIg as a shared cooldown-group key:

| ReqIg | Cooldown |
|---:|---:|
| 1 | 15 seconds |
| 2 | 20 seconds |
| 3 | 25 seconds |
| 4 | 30 seconds |
| 5 | 60 seconds |
| 6 | 120 seconds |
| 10 | 600 seconds |
| 11 | 2 seconds |
| Other | No delay from this lookup |

This is a different use of ReqIg from the forge gold table. Items in the same
cooldown group share that session's group timer. Unsupported Special values are
rejected by the generic use path; they are not silently treated as potions and
are not consumed by that rejection.

Source: `handleUseItem` and `cooldownForReqIg` in
`server/internal/world/inventory.go`.

## Status cures and protection scrolls

EP4.5 `CUser::ItemUseRemState` (0x4653e0) chooses which active effects to
remove by the skill definition's `StateType`. The broad Perfect Cure uses
the debuff category instead. These rules apply to consumables with no complete
item-skill reference.

| Item | Native effect |
|---|---|
| Single-state cures (Special 1–6 and 9–15) | Remove the matching StateType |
| Scroll Protected Faint / Death (7 / 8) | Remove state 7 / 8 and block the next incoming skill with that state |
| Cure (18) | Remove states 9–11 |
| Dispel (19) | Remove states 1, 2, 5 and 6 |
| Abolishing (20) | Remove states 3 and 4 |
| Blessing (21) | Remove states 14 and 15 |
| Holy (22) | Remove states 12 and 13 |
| Combined protection (23) | Remove states 7 and 8 and arm both protections |
| Perfect Cure (26) | Remove debuffs and arm both protections |

GX retains map curses and shares the item's existing `ReqIg` cooldown. A
usable cure consumes one even when there is no matching effect. The two
protection flags are independent, have no timer, and do not stack. The
player and monster skill hit checks consume the matching flag before the
success roll. GX keeps these flags in the current character session.

## Service and utility Special values

| Special | Implemented role / service |
|---:|---|
| 0 | Ordinary healing, or a supported item-cast buff |
| 1–15 on lapis | Trigger a defined attack/debuff skill at level 100 |
| 16 on lapis | Resource recovery gem |
| 17 | Percentage healing potion |
| 33 | Reset learned skills and restore the original skill point budget for your level |
| 34 | Reset allocated character stats through the item-use path |
| 36 | Linking hammer; ReqVg adds percentage points |
| 44 | Appearance change; validated through the appearance workflow |
| 45 | Sex change; validated through the appearance workflow |
| 46 | Launch the item’s nearby firecracker notice, visual effect and positional sound |
| 51 | Enable the character rename workflow |
| 58 | Extraction hammer; ReqVg is direct success percent |
| 62 | Recreation rune; consumed by the recreation service |

This is a catalog of supported paths, not a claim that every Special value from
other servers works. A service item is not necessarily usable by double-clicking
it. For example, recreation requires equipment and a rune at the blacksmith;
extraction requires a selected socket and an extraction hammer.

### Skill Reset Stone

The active item catalog has two entries, ItemID `100047` (type 100, variant 47) and ItemID `100089` (type 100, variant 89). Both use Special 33.

The skill reset restores the original EP4.5 budget of five starting skill
points plus three per level gained (four in Hard and five in Ultimate).
Learned skills and their shortcuts are cleared. Cooldown deadlines remain
keyed by skill ID in SQLite so relearning or reconnecting cannot reset them.

### Restored consumables

Untouchable Potions apply spell 276 at ranks 1–3. Grape, Melon and Vanilla
Ice Cream use rank 2 of spells 268–270. Continuous HP, SP and MP potions
use spells 277–279, restoring the corresponding resource every three seconds
for one minute. Remedy and Nostrum variants use their matching later spell IDs.
GX expands the legacy byte-sized references when loading the client catalog.

Resurrection runes and item/EXP drop protection consumables have been retired.
Unconfigured legacy boxes are retained as inert items with a zero box reference.

### Firecrackers

Victory, Love, Conquest and Butterfly firecrackers use the corresponding
1–4 flarebomb effects, with the explosion audio cues verified against Phoenix.
The server consumes one item and sends the launch to both factions within
40 units in the same instance. The client shows the localized SysMsg 835.
The four items share the 20-second item cooldown configured in the catalog.

## Compatible targets in gem tooltips

Both lapis (Type 30) and lapisias (Type 95) list compatible equipment categories
under native SysMsg 5181, **Item possible to equip**. Category labels come from
original system messages, and placement uses the same native flags:

| Equipment category | Lapis/lapisia flag |
|---|---|
| All weapon families | ReqLevel != 0 |
| Helmets | Country != 0 |
| Upper armor | First class flag != 0 |
| Lower armor | Second class flag != 0 |
| Gauntlets | Third class flag != 0 |
| Shoes | Fourth class flag != 0 |
| Shields | Fifth class flag != 0 |
| Rings | Sixth class flag != 0 |
| Necklaces | Grow != 0 |
| Coat / cape | Accepted by the current native-family compatibility rule |
| Bracelets | ReqStr != 0 |

These flags do not impose player level, faction, profession, mode or STR
requirements on linking. The tooltip therefore does not display their ReqLevel
flag as a character level requirement. Normal equipment retains its usual level
label. Alternate equipment Types are normalized to their native family.

Compatibility is only one requirement. The item must also have room, not already
contain the same lapis TypeID, and satisfy the selected service's remaining
checks. A listed category does not grant sockets or make every item of that
category forgeable. Neither lapis nor lapisias show sockets of their own.

## Visual effects versus gameplay

Weapon enchant glows are presentation. They start at +5; +5 through +10 select
six native tiers, and +11 through +49 reuse the +10 visual tier. The glow does
not set chance, cost or damage. Those remain authoritative server rules.
See [Weapon effects](weapon-effects.md) for family selection, particles,
remote-player updates and limitations.

## How to add or correct an effect

1. Identify the exact Type/TypeID and inspect its SQL definition and matching
   client SData row. Check field overloading for that item family.
2. Confirm that the intended server handler exists. A new Special behavior
   requires code and verification; description edits alone are insufficient.
3. Change the selected database and client definition consistently. Stop the
   affected component and preserve a database backup before content migrations.
4. For skills, validate SkillID, level, effect category, duration and abilities.
5. Verify consumption, rejection, persistence, relogging and the displayed
   tooltip. Use disposable test state for destructive scenarios.

See [Item data](item-data.md), [Building and testing](building-and-testing.md),
and [Server operations](server-operations.md).

Movement speed from equipped `Items.Speed`, Sonic lapis and ability 17 is
server-authoritative. Equipment and active lapis `AttackTime` also contribute
to [Attack speed](attack-speed.md); weapons supply the native base index.
See [Movement speed](movement-speed.md) for tiers, caps,
consumables and the original running effect.


## Teleportation Stone

Using a Special 39 stone opens the classic Recorded Point window. Every
character has four slots, with foreground buttons on each row. Use **Store
the Mark** to save the current location and **Move** on a recorded row to
travel. The window uses the external `waypoint.png`, `way_button.png`,
`talk1_close_button.png` and `gatekeeper_button.png` interface textures. Saving and replacing points costs nothing. Successful travel consumes
one stone. Travel has the original five-second cast. Movement, damage,
other character actions and Escape interrupt the cast before the stone is spent.
Dungeons and instanced zones are excluded, and portal-gated fields
must have an eligible entrance. Points survive logout and are local to the
character.


## Party Member Summon Rune

A Special 31 rune is consumed when its five-second cast starts. Damage,
movement, or another character action can interrupt it. A completed cast
offers a 60-second summon to party members. In raids, members summon their own
group, and the raid leader can summon the entire raid.
The classic confirmation names the summoner and shows the origin map in yellow.
Accepting moves the recipient immediately to the summoner's current position,
subject to map access and the party membership still being valid.
