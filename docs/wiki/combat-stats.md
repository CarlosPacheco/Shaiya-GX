# Combat stats and damage

## What each attribute does

| Stat | Combat effects per point |
|---|---|
| STR | +1.3 melee and ranged attack |
| REC | +1 physical defense and +5 maximum HP |
| INT | +1.3 magic attack |
| WIS | +0.2 magic attack, +1 magic resistance, +5 maximum MP, +2 direct HP healing, and +0.02 percentage points of both magic accuracy and magic evasion |
| DEX | +0.2 melee and ranged attack, +5 maximum SP, and melee/ranged accuracy and evasion at the rates below |
| LUC | +0.5 ranged (archery) attack, +0.2 percentage points of critical chance, and each point of the *target's* LUC removes 0.01 percentage points from the attacker's critical chance |

Ranged (archery) attack gains half a point per LUC. The character sheet shows
it for Hunters and Archers only, because it is the attack range of the bow
they wield; melee and magic classes see no change.

Use your effective attributes, including equipment, active lapis, skills and
the Goddess Blessing. The character sheet's attack range already includes the
weapon and the relevant attributes. Use the attack roll directly when you
start from that range.

## One formula for melee, ranged and magic

1. Roll attack within the displayed range, including both ends.
2. Add the skill's extra damage. A basic attack adds zero.
3. Subtract the defense for that type of hit.
4. Multiply by the element matchup, then by **1.5** if the hit is critical.
5. If a higher-level monster is involved, apply the PvE damage multiplier
   below. Truncate the result and subtract any flat HP absorption.

| Hit type | Attack before the skill bonus | Defense subtracted |
|---|---|---|
| Melee | Equipment attack + 1.3 × STR + 0.2 × DEX + melee attack bonus | Physical defense |
| Ranged | Equipment attack + 1.3 × STR + 0.2 × DEX + ranged attack bonus | Physical defense + ranged-only defense |
| Magic | Equipment attack + 1.3 × INT + 0.2 × WIS + magic attack bonus | Magic resistance |

Physical defense includes REC, equipment defense and existing melee defense
bonuses. All of it also protects against ranged hits. A ranged-only defense
bonus adds protection against ranged attacks. Magic resistance includes WIS,
equipment resistance and magic defense bonuses. Every point of the applicable
defense removes one point before element and critical multipliers. Bows use
the same damage formula at every valid attack distance.

**Worked example.** The attack roll is 1,200, including the weapon. The skill
adds 200 and the target has 500 defense. Fire I against Wind I gives ×1.4.
The hit deals **(1,200 + 200 − 500) × 1.4 = 1,260**. A critical deals
**1,890**. With 30 absorption, the defender loses **1,230** or **1,860** HP,
respectively, before effects that redirect damage to SP or MP.

A weapon-based hit that lands deals at least 1 damage after absorption. The
weapon calculation caps at 30,000 before absorption. Skills with an authored
damage multiplier also apply that multiplier before the cap. Fixed and
resource-based skills use their listed amounts or resource percentages.
These immediate hits still use the hit check and any applicable PvE level
adjustment. See [the element table and damage modes](elements.md).

## Critical chance

**Critical chance = 0.2% × attacker LUC + critical bonuses − 0.01% × target
LUC**, limited to 0–100%. The starting chance at 0 LUC is 0%. The target's LUC
counts for players and monsters alike, on normal hits and on skill hits.

| Attacker LUC | Chance from LUC alone | Against 500 target LUC |
|---:|---:|---:|
| 0 | 0% | 0% |
| 100 | 20% | 15% |
| 475 | 95% | 90% |
| 500 | 100% | 95% |

Every 100 points of the target's LUC take 1 percentage point off the roll,
so 500 LUC on the target removes 5 percentage points. A skill
that grants +5 critical adds **5 percentage points** before that reduction.
For example, 475 LUC plus that skill reaches 100% against a target with no
LUC, and 99.5% against one with 50 LUC. A critical multiplies the damage by 1.5.

## Accuracy and evasion

Skill accuracy and evasion values are percentage-point modifiers by category.
An ability value of 5 adds five points of melee, ranged or magic chance
according to its ability type. A reduction subtracts the listed points.

**Effective evasion = defender's evasion − attacker's accuracy**, limited to
0–100%. **Hit chance = 100% − effective evasion.**

Example. A defender with 50% evasion facing 35% accuracy evades **15%** of
attacks. The attacker hits **85%** of the time. Equal accuracy and evasion
give 0% effective evasion. With 100% evasion and 0% opposing accuracy, the
defender avoids every ordinary hit before any PvE level adjustment.

Use the ratings for the actual hit category. Melee and ranged draw from the
same DEX curve and can have different skill bonuses. Magic uses WIS. Each
rating starts at zero before attributes and bonuses.

| DEX band | Accuracy and evasion per point in the band | Total from DEX alone |
|---|---:|---:|
| 1–500 | +0.10 percentage points | 50% at 500 |
| 501–800 | +0.05 percentage points | 65% at 800 |
| 801 onward | +0.01 percentage points | 100% at 4,300 |

The bands add together. For example, 600 DEX gives 50% for the first 500
points and 5% for the next 100, for **55%**. The lower rate applies only to
points in the new band. SP and attack keep their full per-point scaling.

For magic, **every WIS gives +0.02 percentage points** of both accuracy and
evasion. For example, 500 WIS gives **10%** in each rating. Magic accuracy
skills can raise the attacker's rating directly.

Each chance modifier on a skill is a whole number of percentage points with
a magnitude of 1–100. Older catalog values above 100 contribute 100 per
modifier. Debuffs reduce the corresponding rating. Ratings can accumulate
above 100; the final evasion after subtraction is the value capped at 100%.
A skill's listed success chance is an additional roll. For example, an 80%
hit chance and a 50% skill success chance give a 40% chance to land, assuming
the target can be affected. Blocks and monster skill immunities still apply.

## Fighting a higher-level monster

For each level the monster is above you, the adjustment increases by **5%**,
up to **50%** at ten levels. The adjustment multiplies the calculated result.

| Result from the player's perspective | Multiplier |
|---|---|
| Chance to hit the monster | 1 − adjustment |
| Chance to evade the monster | 1 − adjustment |
| Direct damage dealt to the monster | 1 − adjustment |
| Direct damage received from the monster | 1 + adjustment |

Example. A monster four levels higher gives a 20% adjustment. An 80% hit
chance becomes **64%** and 30% evasion becomes **24%**. A 1,000-damage hit you
deal becomes **800**, and a 1,000-damage hit you receive becomes **1,200**,
before absorption. The adjustment applies to melee, ranged and magic,
including immediate fixed and resource-based damage.

An equal-level or lower-level monster gives a zero adjustment. Player
levels do not enter PvP hit or damage calculations. The same ratings can
give a level 1 character a hit against a level 60 character.

## Healing and periodic effects

**Direct HP healing = the skill's healing amount + 2 × the caster's WIS**
for a skill with a positive HP heal. The target recovers up to its missing
HP. For example, a 300-HP heal cast with 120 WIS can restore **540 HP**.

Periodic healing uses the listed tick amounts. Periodic damage uses the
effect's own tick calculation, including any resource percentages it lists.
Periodic effects skip the weapon damage, critical and higher-monster-level
multipliers.

## Soul Burn

Priests and Oracles can learn Soul Burn at level 22 for three skill points.
It has one rank, a two-second cast, a cost of 430 MP, a 24-unit base
`AttackRange`, and a five-minute cooldown. It has a fixed 100% success rate
against eligible targets.

Select an enemy player's corpse. The caster and corpse must be in the same
instance and the cast follows standard launch reach and visibility checks.
On success, the enemy immediately returns to their normal respawn point with
full pools and the usual death penalty. The visual effect
plays where the corpse lay. There is no lingering debuff or resurrection timer
added by the spell.

## Server configuration

Set the two DEX thresholds in the environment or in the server's
`.local/runtime.env`, then restart World. Both must be positive integers,
with the second greater than the first.

```ini
RAPTURE_COMBAT_DEX_FIRST_THRESHOLD=500
RAPTURE_COMBAT_DEX_SECOND_THRESHOLD=800
```

[Server operations](server-operations.md) · [Combat implementation](../../server/internal/world/combatformula.go)


## Transformation, Disguise, and Evolution

Transformation copies the selected living monster. Disguise copies the
appearance and visible name of the selected living opposing player. Both require
a template in range at the caster’s level or lower and end on combat or expiry.
The real character ID, equipment and faction continue to govern game rules.

Evolution uses `ChangeLevel` for the monster on each side. The client selects
the model from its player skill catalog. The server selects the NPC skills from
the monster's three attack slots, skipping basic attacks and unused slots. The
recipient retains its stats and basic attack, and uses the temporary NPC skills
by clicking the bar above quick potions. An empty slot stays empty. The NPC
skill uses its catalog’s damage, effects, range and reuse. The grant expires
with the form, and recasting the buff does not reset an active reuse timer.
