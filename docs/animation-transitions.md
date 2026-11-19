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

## What GX currently does

- `Character.SetClip` and `Character.PlayAction` swap the clip and reset its
  playback time. `Clip.finalsInto` interpolates keys within a single clip.
- When a player one-shot expires, the update unlocks it. The sampler wraps
  at the duration, so a render between unlocking and the next locomotion
  selection can briefly sample the opening pose of the outgoing clip.
- Creature action changes reset their clip immediately. Their one-shot
  handling clamps past the end, which already avoids the player wrap case.

A GX implementation can keep the outgoing local pose, blend into the next
clip's reference pose over five animation frames, then start the new clip.
The same resulting palette should drive rendering, attachments and pose-bound
effects. Keep visual interpolation independent from gameplay authority,
damage execution and cooldowns. First validate ordinary movement, one-shot
exit, cast interruption and death handling for self and remote actors. A
mid-blend interruption deserves a separate check because the normal EP5.4
action-request helper guards against replacing a pending transition while
several special action paths write the pending state directly.

## Verification

The observations come from Ghidra and disassembly of the original binaries,
plus a read-only probe of installed ALT/ANI data. For the sampled human
fighter clips, including idle, movement, attacks, Stinger, death and a
gesture, the evaluated pose at the first frame matched the stored reference
pose to the precision printed by the probe. This is a sample of the catalog.
Native playthrough and visual comparison remain to be done.
