# Attack speed

GX Server derives attack speed from the weapon's native `Items.AttackTime`,
equipped item bonuses, active lapis, learned weapon masteries and applied ability
**16**. The final index is clamped to 0–9 after adding all signed modifiers.
The weapon field is the index itself: subtracting one selects the wrong delay.

| Index | Native status description | Server interval (ms) |
| --- | --- | --- |
| 0 | None / unarmed | 4000 (table fallback; a valid weapon is required to attack) |
| 1 | Extremely slow | 3000 |
| 2 | Very slow | 2750 |
| 3 | Slow | 2500 |
| 4 | A bit slow | 2250 |
| 5 | Normal | 2000 |
| 6 | A bit fast | 1750 |
| 7 | Fast | 1500 |
| 8 | Very fast | 1250 |
| 9 | Extremely fast | 1000 |
| 255 | Cannot attack | Blocked |

Both PvE and PvP autoattacks use this table and the last server-side swing time.
Restart requests, target switches, stop/start floods and client speed packets do
not reset that clock. Removing or expiring a bonus changes the next allowed
swing according to the new derived interval. Stun, sleep, mounting and active
cast preparation prohibit normal attacks. Beginning preparation stops an existing
autoattack; attack speed is published as blocked until preparation ends or is
cancelled. Spell cooldowns and preparation times retain their own skill rules.

## Native item and skill fields

- `Items.AttackTime` is a weapon's base attack index. On an equipped accessory or
  an active lapis it is an additive attack-speed bonus. It is counted once.
- Oblivion Flash Ring (`22/56`) adds one index; Flash Ring (`22/213`) adds six.
  Flash Lapis (`30/64`, `30/65`, `30/66`) add one, two and three respectively.
- `Items.Speed` is the movement-tier bonus, including Oblivion Sonic Amulet
  (`23/57`) and Sonic Lapis. See [Movement speed](movement-speed.md).
- Learned mastery detail **8** adds its `Weaponvalue` only for matching weapon
  families. General passive abilities and active buffs use ability **16** for
  attack and **17** for movement. Debuffs subtract their configured value.
- Equipment kept in carried bags grants no equipment bonus. Native compatible
  targets, equipment requirements and active-lapis selection still apply.
- Delicious Candy (`25/175`) applies native skill `211/1`, adding one attack
  index with normal consumption, cooldown and expiry.
- These fields are overloaded for consumables: `Range` and `AttackTime` can
  identify an item-cast skill and its level. A consumable's skill level is never
  displayed or applied as an equipped attack-speed bonus.

The owner and eligible viewers receive the final indices through native `0x051C`.
View-entry snapshots include them. Equipping, linking, extracting, learning a
mastery, applying/cancelling/expiring effects and reconnecting refresh the derived
state without allowing the client to supply its own rate.

## Stats and tooltips

The original Stats fields at panel coordinates `(92,345)` and `(92,365)` show
attack and movement descriptions from `sysmsg-uni.txt`. They display confirmed
server indices, with an unknown placeholder before confirmation. Attack uses
messages 5271–5281; movement uses the original selected-message switch and 5282
for blocked movement.

Weapon tooltips use native attack descriptions 5192–5201, including active linked
lapis; socket-enhanced attack text is yellow. Additive item/lapis bonuses use the
localized attack-speed prefix and `+N`. Movement bonuses use message 5209 and
`+N`. Tooltips combine native item and active socket fields while preserving the
cached definition and the server's independent calculation. The descriptions
remain available for carried items, shops and the Item Mall.

## Native evidence

Reference `ps_game.exe` SHA-256:
`0fb7c9f13b5bcf186118f8bfe73ab214226b93e8aa502b8e6fded4a8312bc547`.
`InitEquipment` clears attack speed at `0x4545e2`; weapon `ItemEquipmentAdd`
adds `stItemInfo +0x44` directly at `0x454fac`. Accessories add the same field at
`0x454d0a`; gem addition is at `0x456122`. `GetAttackSpeed`/`GetAttackSpeedNow`
(`0x48a460`/`0x48a4b0`) clamp to 0–9, and `g_ConstUserAttackSpeed` at `0x5424e0`
contains the ten millisecond intervals above. Isolated execution verifies 108
weapon/accessory/Flash/Sonic accumulation combinations.

Reference `Game.exe` SHA-256:
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`.
Direct execution of the status dispatch at `0x506b57` and `0x506da1` confirms
11 attack and 6 movement lookups, including blocked indices. Status field
positions are the native constants at `0x70adb8..0x70adc4`.

Regression coverage includes real seed item/passive/buff definitions, raw index
and cap behavior, cast blocking, all ten cooldowns under restart/target floods,
confirmed-only UI, socket aggregation and original-asset OpenGL captures. Isolated
N2 also measures encrypted PvP swing intervals of 1500/1250/1000 ms while changing
Flash Ring and a native attack buff and flooding stop/start requests.
