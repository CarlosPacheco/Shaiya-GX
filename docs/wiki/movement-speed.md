# Movement speed

GX Server calculates the player's movement tier from equipped items, active
lapis, learned passives and applied skill abilities. The client receives that
tier in `0x051C` and uses it for prediction and presentation. Client positions
and client-supplied speed announcements cannot change the server's allowance.

| Tier | Description | On-foot running speed (world units/s) |
| --- | --- | --- |
| 0 | Very slow | 3 |
| 1 | Slow | 4 |
| 2 | Normal | 5 |
| 3 | Fast | 6 |
| 4 | Very fast | 7 |
| 255 | Movement blocked | 0 |

Equipment `Items.Speed` and skill ability **17** add signed tier offsets to
normal. The total is clamped to tiers 0–4 after all modifiers; additional haste
cannot exceed 7 units/s. Existing walking/backward and swimming rules still
apply. Stun, sleep and movement-stop effects can prohibit movement entirely.
The authorized GM `/turbo` extension remains a separate administrator mode.

## Equipment, spells and consumables

- Sonic Lapis Lv1 (`30/106`) adds one tier; Lv2 (`30/107`) adds two.
  Linking retains the native compatible-target/socket/chance/cost rules: Sonic
  belongs in boots. An equipped linked item contributes immediately; carried
  equipment contributes nothing. Extraction and unequipping refresh the tier.
- Equipped items with their own `Speed` field contribute through the same
  calculation, including the Sonic amulet. Active-lapis selection still applies.
- Haste spells use their server skill definition, costs, cooldown and duration.
  For example, skill `114/1` adds two tiers for 60 seconds.
- Item-cast buffs use the item's `Range` skill ID and `AttackTime` level.
  Apple Candy (`25/177`) applies `210/1`, adding one tier for 3,600 seconds.
  The server consumes the item and enforces its cooldown. A refused weaker
  same-skill buff consumes nothing. Remaining buff time restores on reconnect
  according to the existing offline buff persistence policy.

Changes are sent to the owner and eligible current viewers. A player entering
view receives a speed snapshot, so existing equipment or buffs do not require a
new cast to become visible. The server meters displacement against its own
elapsed time and derived speed, bounds accumulated transport credit (including sub-tolerance packet floods) and removes
faster credit when the tier changes. Movement blocks also clear this credit.
Existing collision, terrain height and jump authority remain in force.

## Original running effect

An on-foot character running above normal emits the original `Effect/Crun.EFT`,
using faction sequence 0 or 1 at the original interval of more than 100 ms.
Native `0x450340` lowers the logical actor origin by 0.9 units, so GX emits at
its already-converted feet position without adding that offset again.
Each emission is a finite effect at its spawn position; it is not a replacement
loop or a custom texture. The owner and remote users use server-confirmed speed.
Standing, walking, backward movement, swimming, sitting, death, concealment and
mounted movement do not emit the running trail. Hidden/removed actors cannot
leave visible ongoing trails. The existing skill-effects display setting applies.
Mount speed tiers and further mount mechanics remain deferred.

## Native evidence and validation

Reference client `Game.exe` SHA-256:
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`.
The movement controller at `0x43bfa0` selects base speed plus −2/−1/0/+1/+2;
`0x409250` initializes that base to 5. The running-effect branch at
`0x41c2fa..0x41c375` checks the movement tier, selects effect library 10 and
faction sequence, and uses the 100-ms timer.

Reference server `ps_game.exe` SHA-256:
`0fb7c9f13b5bcf186118f8bfe73ab214226b93e8aa502b8e6fded4a8312bc547`.
`0x48a530` clamps the raw tier; `0x46a36d..0x46a3a8` selects speed. Direct
isolated x86 execution verifies 54 selector cases, including bounds and vehicle
branches; the latter are evidence only and are not new GX mount behavior.
`ItemEquipmentAdd` (`0x454c70`) and `GemEquipmentAdd` (`0x4560c0`, speed addition
at `0x45612c`) use the item-info movement field at offset `0x48`.

Regression checks cover real seed Sonic/haste/candy definitions, server-time
movement limits for every tier, packet flooding, blocks, equipment changes,
view-entry snapshots, cooldown/refusal, expiration and buff restoration.
The asset-backed OpenGL test renders both original Crun sequences and verifies
remote emission, concealment and finite lifetimes. The isolated N2 test also
exercises equipment, casts, consumables, reconnect and forged movement over the
actual encrypted transport. No asset, geometry or database migration is needed.

Stats uses native movement messages 5273/5274/5276/5278/5279 (and 5282 when
blocked). The highest on-foot tier is 7 units/s even though this client labels it
“very fast”; display text does not introduce another movement tier. Native item
and active socket bonuses appear in tooltips through message 5209.
