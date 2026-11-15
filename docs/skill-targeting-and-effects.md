# Skill targeting, reuse and target effects

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

The 800ms attack/skill pause restricts **movement**, not the next skill request.
A shared **300ms per-player cast cadence** prevents rapid packet/key floods.
It applies across different skills, buffs and self-heals, independently of their
reuse timers. An accepted instant release starts this interval; prepared casts
reserve it at the start and renew it on release. Completing a prepared cast is
never rejected by its own request cadence. Rejected requests do not extend the
server interval. Buff cancellation and ordinary item reuse are unaffected.
Client prediction uses the same interval; instant acknowledgments do not add
another network round trip to it. Instant skills can still chain during the
800ms movement pause, at roughly three to four casts per second.
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
