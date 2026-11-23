# Original skeletal animation transitions

Static investigation of the PS107 client and the EP5.4 client. The two
executables agree on the normal character transition: **five animation frames
at 30 frames per second**, nominally **167 ms**. Both clients use the same
per-bone blend for their creature render paths.

## Transition from one action to the next

1. The actor records the requested action and resets a separate transition clock.
2. During the blend, it holds the outgoing action at its last playback cursor.
   The destination is the incoming ANI's reference pose. The incoming playback
   cursor does not advance during the blend.
3. Each bone's local rotation is interpolated with quaternion slerp. Its local
   translation is interpolated linearly. The weight is transition frames
   divided by five. Parent transforms are composed after local interpolation.
4. The update promotes the requested action when the transition clock exceeds
   five frames. The new playback cursor starts at zero. The threshold follows
   elapsed frame time, independent of the action's playback rate.

| Evidence | EP4 PS107 | EP5.4 |
| --- | --- | --- |
| Character transition clock and action promotion | `0x41A036–0x41A073` | `0x41B3A6–0x41B3E3` |
| Render dispatch to transition sampling | `0x413CE3–0x413D14` | `0x415033–0x415064` |
| Skeletal interpolator | `0x53FAE0` | `0x552CB0` |
| Seconds-to-frames multiplier and threshold | `0x6A6648 = 30`, `0x6A6670 = 5` | `0x6BC870 = 30`, `0x6BC898 = 5` |
| Blend normalization | `0x6A859C = 0.2` | `0x6BE7FC = 0.2` |

The EP5.4 ANI loader (`0x553305–0x553376`) derives each bone's local
reference transform from its file matrix and its parent's inverse. The blend
reads that matrix on the incoming side. It uses the outgoing playback cursor
only when sampling the source. The resulting local transform is then composed
with the already evaluated parent. Creature transition clocks use the same
constants in the verified EP5.4 update paths at `0x432548` and `0x434208`.

Action endings are separate from blending. In EP5.4's character update, ALT
mode byte 1 restarts playback and mode byte 2 requests the action at
`Next` through `0x4105B0`. Death has its own final-frame handling. In the
installed human fighter data, the gesture in slot 116 leads to idle (0), the
attack in slot 38 leads to the weapon-ready action (34), and sitting down in
slot 10 leads to the seated loop (12). Mode comparisons use the low byte.

## GX implementation

Player transitions blend the matching prefix of the two skeletons over five
frames at 30 Hz. This covers locomotion, posture, gestures, weapon attacks,
and rider clips with trailing bones on either side. Incoming-only bones use
their authored local pose. Creature blends require the same skeleton size.
The resulting palette also drives attachments, shadows, and skinned bounds.

Normal idle and one-shot entries use the incoming ANI reference pose.
Walk, run, directional steps, swim, and float blend to their first keyed pose
so playback starts continuously. A new locomotion request waits for the current
locomotion blend to finish and the controller then uses the latest input.
Explicit actions may still interrupt from the visible intermediate pose.
EP5.4 requests normal idle at `0x441d32` and samples the shared bones at
`0x552cf0`. The male Human Fighter's weapon runs in slots 29, 40, 47 and 54
have 38 bones. Normal idle and walking have the same first 36 bones.

The on-foot jump (ALT 8) cancels an active blend and starts immediately.
When it finishes, it requests normal idle even while movement is held.
That blend also accepts a matching prefix of the jump skeleton and completes
before the next locomotion clip. PS107 uses `0x43f125` / `0x54e4f3` for
local / remote takeoff and `0x41aacd` for the ALT successor. EP5.4 uses
`0x43f01d` / `0x560f31` and `0x41be49` respectively.

The jump clip can finish after entering water. The character then resumes the
applicable water locomotion. Ladder entry activates slot 18 immediately
(`0x443a12` locally and `0x56492b` remotely in EP5.4). Lateral evasion
displacement follows the first 15 playback frames after the entry blend.

One-shot actions retain their final keyed pose when blending to their
successor. Posture, lateral evasion, backstep, and social gestures honor their
recorded ALT successors for normal idle and seated idle. The local playback
cursor waits for the blend. Externally
driven action frames, cast countdowns, and the spin lifetime keep their own
clocks. The existing controllers still choose successor actions.

## Verification

The observations come from Ghidra and disassembly of the original binaries,
plus a read-only probe of installed ALT/ANI data. For the sampled human
fighter clips, including idle, movement, attacks, Stinger, death and a
gesture, the evaluated pose at the first frame matched the stored reference
pose to the precision printed by the probe. This is a sample of the catalog.
Native playthrough and visual comparison remain to be done.

## Playback rate by speed index

The blend clock is fixed at 30 frames per second, but the frame cursor of an
action advances by seconds times a rate that depends on the actor's speed
indices (EP5.4 `0x41b300` for characters, `0x432520` for monsters):

| Actor and state | Index | Frames per second |
| --- | --- | --- |
| Character walking/running/stepping | move 0..4 (0x051C) | 24, 27, 30, 33, 36 |
| Character basic attack | attack index minus the weapon's AttackTime, plus 5 | 15, 18, 21, 24, 27, 30, 34, 38, 42, 45; outside 0..9 gives 50; no weapon gives 30 |
| Monster moving (walk and chase) | move 0..4 (0x0609) | (index + 1) x 10 |
| Monster idle, attack, damage, death | attack 0..6 (0x0609) | 15, 20, 25, 30, 35, 40, 45; others 30 |
| Everything else | none | 30 |

GX applies these rates to the frame cursor and to the one-shot cue timer of
monsters, leaving the blend at real time. With default indices (move 2, attack
equal to the weapon base, monster 3/2) nothing changes; buffs, debuffs and
mastery now speed up or slow down the animations as in the original.
