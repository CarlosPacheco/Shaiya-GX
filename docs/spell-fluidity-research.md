# Original spell timing, Stinger and the two Wind Spins

## Implementation following the investigation

The user subsequently authorized both special states and explicitly required
fast spin → distinct spin / stationary-area combinations. GX now loops the two
spins for two seconds and allows that follow-up after the recovered frame-4
first contact. The server retains the latest early eligible request, preserves
own reuse/costs and invalidates the intent on movement/another committed action.
This cancellation boundary is an implementation choice grounded in contact
metadata; the static EP4.5 audit below did not establish it as native server law.
Stinger now uses its windup/frozen travel/contact phases with a server-authored,
collision-swept trajectory and synchronized client presentation (GX 0x7F11).
Launch through walls is refused before cost/reuse, client movement cannot alter
the dash and terminal notifications stop remote prediction too. Matching GX
builds are required. Other audited differences below remain research findings.

## Original investigation record

Static investigation, 2026-11-16. This records original behavior and current GX
differences; it does not change gameplay or claim a native runtime comparison.
The two skills requested are **Wind Spin** and **Reinforced Wind Spin**. The
local English table calls the latter `Enhance Wind Spin`.

## Reference identity

Both Ghidra projects' executable hashes match the files inspected with objdump:

| Reference | SHA-256 |
| --- | --- |
| EP5.4 `Game.exe` | `8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263` |
| EP4.5 `ps_game.exe` with PDB | `0fb7c9f13b5bcf186118f8bfe73ab214226b93e8aa502b8e6fded4a8312bc547` |

The server is the existing GX parity reference, not evidence for every later
episode or modified server. Some imported CUser PDB field names/offsets are
misleading. The offsets below were checked against instructions, call-site base
addresses and smaller structures. Binaries and extracted data stay outside Git.

## Four separate clocks

1. Preparation: `ReadyTime × 250 ms`.
2. Per-skill reuse: its own reset deadline, normally `ResetTime × 1000 ms`,
   with the native cooldown-reduction branches.
3. Server scheduling: an absolute next-execution deadline using the attack-speed
   table, separately from the skill's reset deadline.
4. Presentation: frames, transition blending, special state lifetime, release
   effects and target feedback.

An animation being replaced does not establish when damage may execute. A
request reaching the server is not proof that the spell has executed.

## Repeated spells and combinations

The original reuse check in `CUser::EnableUseSkillUser` (`0x44eb40`) and
`EnableUseSkillMob` (`0x44ee50`) reads the selected CSkill instance's deadline.
The predecessor branch requires `wPrevSkill` to identify a learned skill whose
reset deadline is still in the future. This is a specific dependency, not a
generic faster-combo rule. GX loads `PrevSkillID` but currently does not enforce
it in world skill execution.

There **is also a shared execution deadline** in this original server:

- `CUser::UpdateAttack`, `0x48ca63`, compares the tick to CUser `+0x1450` and
  returns without executing an action while the deadline is in the future.
- Instant `UseSkillUser` / `UseSkillMob` write `tick + table[index]` at
  `0x44f2c1` / `0x44f4aa`, after resolving the skill. The index is clamped 0–9;
  `g_ConstUserAttackSpeed` is at `0x5424e0`.
- The table is 4000, 3000, 2750, 2500, 2250, 2000, 1750, 1500, 1250, 1000 ms.
  No same-ID versus different-ID exception was found in this execution path.
- Prepared skills first schedule their preparation deadline. After release,
  `UpdateAttack` writes a new attack interval at `0x48cf71`. GX starts skill
  recovery at preparation start and reserves basic attack recovery on release.
- GX reproduces the stored native deadline for basic attacks. GX skill recovery
  re-evaluates elapsed time against the live speed tier.

Incoming `0x0511` / `0x0517` requests in `CUser::PacketPC` (`0x46a230`) select
skill/target without rejecting solely because `+0x1450` is in the future. The
accepted branches write action kind 2 and the slot at `0x46a9b2..0x46a9bc` /
`0x46aacd..0x46aad7`. They do not clear the deadline: the scheduler executes
the latest selection when allowed. This is one pending selection, not a queue
of accumulated attacks. The target-validation callees do not clear it either.

Client skill-entry paths (`0x4deda0`, `0x500c90`, combination bar `0x4e77d0`)
check each skill's remaining timer; their exclusions do not reject special
state 9 merely because a spin remains visible. GX queues locally until its
shared clock expires; its server discards early requests instead of retaining
the next selection. These are different scheduling flows.

**Conclusion:** alternating avoids the same skill's reuse and can replace a
still-running visual action. This reference does not support removing the
shared execution interval or inventing a universal 300/800 ms combo constant.
Faster-than-table execution needs a matching native runtime trace or another
identified server build. The player's observation must not be presented as
disproven by static inspection of a different server episode.

GX's `ReuseSeconds` erases authored one-second cooldowns. The native server
writes them, and client response paths `0x57bdc0` / `0x57e320` use
`(ResetTime - cooldownReduction) × 1000` when reset exceeds reduction. No general
one-second-placeholder exemption appears in these recovered branches.

## Wind Spin and Reinforced Wind Spin

| Skill | Local ID | Preparation | Reuse, levels 1–3 | Action |
| --- | --- | --- | --- | --- |
| Wind Spin | 38 | 0 | 10 s | 102 |
| Reinforced Wind Spin (`Enhance Wind Spin`) | 156 | 0 | 15 s | 102 |

`0x57bdc0` / `0x57e320` special-case these IDs: play `ef_in_spin01.wav` and
call `0x40ff00`, setting actor state `+0xdc = 9`, queuing action 102, starting
the blend and recording `timeGetTime()` in `+0x2ac`.

`0x413e90`, called by actor update `0x41b300`, leaves that state when elapsed
time is **strictly greater than 2000 ms**, then requests the idle transition.
This is the un-replaced visual state lifetime, not a two-second cooldown or an
uninterruptible cast. The next confirmed spin re-enters the initializer and
refreshes its timer.

Human fighter and DeathEater fighter action-102 clips have frames 0–6 and ALT
mode 1. In `0x41b300`, that mode rewinds on completion; special state 9 uses
30 frames/s. A roughly **0.2 s cycle repeats during the two-second state**, with
separate blend/exit behavior. A following distinct spin can become executable
before that presentation ends without bypassing the server interval.

GX plays action 102 once: `Character.Update` unlocks on clip duration and ignores
this ALT looping mode and spin-state timer. This is a confirmed presentation gap.
The executable branch also contains ID 341, absent from the loaded local table;
it is not counted as a third available spin.

## Stinger: special dash, not ordinary walking

**Stinger**, ID 56 (`0x38`), uses action 101. Local levels 1–3 have 19/20/22-unit
ranges, 50/40/30-second reuse, zero preparation and stun detail 65.

On the result path, `0x57bdc0` / `0x57e320` special-case ID 56 and call
`0x417c50` with the target position. It starts the activation effect, sets
state 9/action 101, copies destination XYZ to actor `+4/+8/+0xc`, enables flag
`+0x70` and records the start tick.

`0x4199f0`, called before ordinary animation progression in `0x41b300`:

- Re-orients horizontally toward the stored destination.
- After the cursor exceeds **frame 10**, advances toward it at **100 units/s**,
  using the full destination vector, independently of ordinary walk/run speed.
- While travelling, returns before advancing the animation cursor. Updates actor
  position, auxiliary position and, for self, camera coordinates.
- Uses a **0.005-unit** arrival tolerance, then resumes cursor progression.
  Feedback and special-state completion use the ALT hit-frame metadata.
- A separate 300 ms check emits a movement effect; it is neither dash duration
  nor a spell recovery constant.

Human fighter clips have 30 frames, but hit markers differ: 26 for male, 24 for
female. Reading only the one-second ANI duration loses the travel pause and
metadata-dependent completion. Blending is separate too; no single exact
wall-clock duration is claimed here.

The server also handles the dash. Against players and mobs, `AttackUser(skill)`
(`0x44f4c0`) / `AttackMob(skill)` (`0x44fd90`) special-case ID 56 and subtract
`AttackRange` from CUser `+0x54b8`. `PacketPC` passes CUser `+0x5494` to
`stUserMoveChk::Update` (`0x46a120`); structure offset `+0x24` is **fTotalDist**.
This is movement-distance credit, not speed reduction, teleport or an exemption
for every stun. The CUser PDB label misleadingly calls the accessed member
fMaxSpeed; raw offsets resolve it. Both hit and failure-result branches subtract.

Before this implementation, GX had no equivalent ID-56 dash state or dedicated movement allowance. Playing
action 101 and applying stun does not implement Stinger. A future port must
reconcile dash movement with GX authority, geometry and replication rather than
copy an old client-trusted allowance blindly.

## Validation and follow-up boundary

Evidence: read-only Ghidra decompilation and reference/operand scans; objdump
against the hashed files; existing SData/ALT/ANI parsers against local playable
data. Private captures and probe: `work/spell-fluidity/`. No original runtime
playthrough, packet-timing measurement, gameplay implementation, personal DB
change or build installation was performed for this investigation.

Before changing cadence, measure request/response/damage timestamps for the
same instant spell, alternating instant spells, prepared spells, Wind Spin →
Reinforced Wind Spin and Stinger → follow-up at identified speed tiers on a
matched original server. Separate animation replacement from damage timing.
