# Skill targeting, reuse and target effects

The timing rules below describe current GX behavior. The
[original spell-timing investigation](spell-fluidity-research.md) records
confirmed differences in scheduling, prepared-cast recovery, one-second reuse,
the two Wind Spins and Stinger. Full native timing parity is not established.

Single-target heals and buffs use the caster when an enemy is selected. The
combat selection remains on the enemy, so an offensive skill can follow without
reselecting it. Valid friendly players keep their selected recipient; stale,
dead or out-of-range friendly recipients still fail normally. The server resolves
this fallback before preparing a cast, retaining the resolved recipient even if
the selected enemy subsequently disappears. Server team/duel rules remain final.

Player skills with authored reset times of zero or one second have no reuse
cooldown. Reset times above one second retain their full duration, including a
one-second *remaining* portion of a longer cooldown. Entry snapshots, server
reuse validation/persistence, confirmed client timers and action-bar duration
use this same rule. Mob AI reuse is unchanged.

Basic attacks, skills and combos now use **one shared attack-speed action clock**.
The live server index selects the native interval: 4000ms unarmed, 2000ms Normal,
1500ms Fast, 1250ms Very Fast and **1000ms Extremely Fast**. The full 0–9 table is
in [Attack speed](wiki/attack-speed.md). Equipment, mastery, haste and slows affect
this same recovery immediately. Ordinary alternation and stop/start requests
cannot bypass it; the spin follow-up exception below is intentional. The temporary fixed 300ms spell interval has been removed.

An accepted instant skill reserves the action clock at release. A prepared skill
reserves it at the start; its release completes without charging a second full
attack interval. Authored preparation times and longer per-skill cooldowns still
apply. The separate 800ms movement stop is unchanged.

Wind Spin and Reinforced Wind Spin repeat their native action-102 cycle for
up to two seconds. After the first contact (ALT frame 4 at 30 fps), another
**distinct** spin or an instant hostile area centred on the caster can replace
the spin and execute before ordinary recovery ends. Both endpoints enforce the
same eligibility and contact boundary; each skill keeps its own cooldown,
costs, targeting and obstruction checks. Recasting the same spin, ordinary
attacks, buffs and prepared spells cannot use this exception. The server retains
one early eligible follow-up until contact, replacing rather than accumulating
requests. Movement or another committed action invalidates that pending intent.
This implements the user's observed combo behavior; the exact cancellation
boundary is based on recovered first-contact metadata, not a measured native
server combo interval.

Stinger (ID 56) has its native action-101 windup, 100 units/s dash and
race/gender contact frames. The server captures a validated destination, sweeps
collision, simulates travel and publishes a trajectory/terminal notification
through GX 0x7F11. Self and remote clients present the same frozen travel pose.
Client move packets do not move the actor during the dash; no free movement
credit remains afterwards. Walls reject launch before costs and reuse; a dynamic
obstruction or disabling effect can stop travel. Matching client/server builds
are required. See [the recovered state machines](spell-fluidity-research.md).

The client retains only the latest spell pressed during recovery. It stops basic
auto-repeat without clearing the shared clock, then revalidates and sends that
spell when recovery permits it. This avoids automatic basics consuming every
available action slot. Manual movement, death, mounting, expiry or a changed
non-self recipient cancel the queued intent; repeated keys replace it rather than
building a burst. Buff cancellation and ordinary item reuse are unaffected.
Ordered instant predictions keep older acknowledgments from restarting the next
combo animation; area/multi-hit replies likewise preserve newer predictions.
Real cast times, MP/SP costs, action restrictions and each longer skill cooldown
remain server-authoritative. Casting a sequence keeps renewing the movement
pause; it does not permit attacking while moving.

Reach failures use the client's runtime SysMsg table: **458**, "There is an
obstacle.", for blocked spell line of sight; **312**, "Insufficient range", for
range failures. The previous hardcoded combined center-screen message is gone.
`GXSkillFailure` (`0x7F0E`) is a server-only notification carrying skill ID and
native message ID, each uint16 after the opcode. It uses ordinary framing and
avoids assigning invented meanings to native skill-result codes. No resource
cost or cooldown starts on a reach refusal.

Sacrificial attacks such as Eraser (TypeDetail 23) consume the caster's whole
HP after an accepted cast resolves its targets. This applies to single-target
and area casts against monsters and players, even if every hit misses. A
refused cast leaves the caster alive.

## Native debuff pulse

The original `0x4c0267–0x4c02c4` folds a one-second phase at 0.5, then multiplies
by 255 and truncates to the overlay alpha byte. The visible pulse is consequently
0 → 127/255 → 0. It is not an opaque, descending sawtooth. Own negative-effect
icons and selected-target negative effects share this pulse. Native type-effect 3
and NPC wire-level 100 mark negative icons; ordinary healing effects do not flash. Adding or refreshing
an effect does not reset the global phase or its authoritative lifetime.

## Effects below the selected target

The native target HUD `0x50b6a0` draws 16×16 skill icons at 18px pitch, wrapping
after eight icons over its 140px bar. The recovered frame-local anchor is
(40,54), from `0x50c979–0x50c994` and floats `0x70af4c/0x70af50`. GX applies its
existing 8px target-art inset, placing this row at frame origin + (40,62), below
its current HP bar. Tooltips show skill information and remaining time. These
icons cannot cancel another actor's effects.

Native reader `0x5667a0` supports the following server payloads after the opcode:

| Opcode | Fields |
| --- | --- |
| `0x030B` snapshot | kind uint8, target uint32, count uint8; each entry: skill uint16, level uint8, remaining seconds uint32 |
| `0x030C` add/refresh | kind uint8, target uint32, skill uint16, level uint8 |
| `0x030D` remove | kind uint8, target uint32, skill uint16, level uint8 |

Kind 1 is a player; kind 2 is a mob. GX explicitly requests the selected actor
using `GXTargetEffectsQuery` (`0x7F0F`): kind uint8 + target uint32. Kind/ID zero
clears the subscription. This request is a GX extension; no native request opcode
is claimed. The server replies using the recovered native snapshot layout.

Only the selected, visible actor on the same map may be queried. The server
tracks actual player buffs or mob applied effects, rechecks visibility/life, and
sends changed snapshots at most four times per second. Expiry, cure, replacement,
death and disappearance clear the displayed state through server snapshots;
local countdowns never invent application or removal. Switching selection clears
the old row immediately, and late snapshots cannot appear under a different target.

Checks cover enemy-to-self fallback, prepared-cast recipient stability, reuse
restoration/chaining, localized framed obstacle failures, the native alpha
curve, native target-effect codecs, visibility filtering and removal. GL checks
use original icons; the disposable N2 suite queries real player and mob effects
through the encrypted connection.
