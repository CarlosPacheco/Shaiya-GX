> Runtime update: GX now executes the default [modern Lua catalog](../server/ai/README.md)
> through the integrated Go world adapter. The investigation below records the
> original's families; the former generic Go decision loop has been removed.
> Further integration checks: native `MobRemove2` (0x0041e630) scans five cells,
> removes all matching live mobs at distance<100 and leaves its count argument
> unused. The four-byte attack mask at CMob+0x278 has setters/resetters but no
> observed combat reader in this executable; it does not justify damage immunity.

# Original EP4.5 mob AI dispatch and case inventory

Research date: 2026-11-16. Scope: investigation only. No GX runtime, database,
AI implementation or installed executable was changed by this investigation.

## Sources and evidence

The authoritative package is the local EP4.5 `PSM_Client/Bin` directory. Its
`ps_game.exe` SHA-256 is
`0fb7c9f13b5bcf186118f8bfe73ab214226b93e8aa502b8e6fded4a8312bc547`.
The PDB-backed read-only Ghidra project identifies the original functions below.
`Data/mob_ai.ini`, all 180 `Data/AI` files and all eight `Data/Lua` files agree
byte-for-byte with the package's `PSM_Server/ShaiyaServerPatch` copy. The two
copies are not different AI versions.

Counts of assigned mobs below refer to the **current GX seed's 2,747 template
rows**, not live spawn counts or a claim about every original database version.
The client SData is EP5 and can have different assignments.

| Native function | Address | Evidence |
| --- | --- | --- |
| `CMob::Init` | `0x00490040` | Dispatch thresholds, factory success flag, Lua initialization |
| `CMob::Targeting` / `SetTarget` | `0x0049fcf0` / `0x0049fad0` | Generic acquisition and help routing |
| `FindHelpNormal` / `FindHelpNormal2` / `FindHelpAttack` | `0x0049ff70` / `0x004a0200` / `0x004a0470` | Different helper populations |
| `CAIFactory::LoadMobAI` / `InitFSM` / `SetFSM` | `0x004a8df0` / `0x004a98f0` / `0x004a9130` | Config and four runtime files, per-mob instance |
| `CKFSM<CMob>::EvaluateCurrentStateTransitions` | `0x004a83e0` | Ordered transition scan and return on a selected transition |
| `CMob::DoFunction` | `0x004917a0` | Native system-function IDs and their actual effects |
| `CMob::onBattle` / `PAI_Attack` | `0x004b0520` / `0x004af9c0` | File-driven states still call native combat |
| `CMob::LuaRegister` / `InitLuaFSM` | `0x004b1310` / `0x004b4c20` | Exact Lua filename and shared native stack machine |
| `CMob::WhileCombat` | `0x004b55c0` | Native attacks alongside Lua callbacks |
| `LuaDispatcher<LuaMob>::OnError` | `0x004b41c0` | Lua errors disable callback execution |

## The three dispatch families

| AI byte | Route in the supplied executable and files |
| --- | --- |
| 0–9 | Generic code-driven behavior. Individual meanings below. The ten `ai/empty` config lines do not load real FSM files |
| 10–36 | Factory/FSM route, with the configured case-specific assets below |
| 37–99 | Factory is attempted, but this package has no corresponding config entry. `SetFSM` cannot attach that FSM; initialization does not set status 100 on failure. This is not an alias for a named boss AI |
| 100–255 | Native Lua stack machine; file path is `.\Data\Lua\<AI>.lua`. Only 111–118 exist here |

The families are dispatch mechanisms, not three complete replacements for
combat. File FSM states and Lua callbacks both reuse native targeting, movement,
attacks and skills. In particular an empty Lua combat callback does **not** mean
that the mob cannot attack.

### File-driven state cadence

The zone scheduler at `0x407646` calls `CMob::UpdateIdle` via
`0x421a00` and sets its next deadline to current time plus 1000 ms at
`0x40766d`. Status 100 evaluates one FSM transition during this update.
GX evaluates the file-driven cases at that cadence while retaining the
faster movement scheduler. In AI 36, the unconditional transitions
between Battle and UseMobSkill1 take two updates per repeat of skill 174.
Its Idle state acquires players by proximity through `IsFirstAttack`, which
calls `CMob::FindTarget` directly. The generic acquisition warm-up applies
when the policy explicitly checks it.

### Routine spell timing

The EP4.5 server's `CMob::UpdateAttack` (`0x004a7db0`) checks the shared
attack deadline. `CMob::AttackAI` (`0x004a1ab0`) and `CMob::AttackAIExp`
(`0x004b66f0`) scan slots 3 through 1 in order, checking the slot deadline,
`Attackok` conditions, and range. The slot scan contains no random cast roll.
A successful `CMob::UseSkill` (`0x004a48e0`) starts the shared 2.5-second
attack delay with its speed adjustment. The slot deadline uses the skill reset
time in seconds. Scripted encounters can make separate random decisions, as
Lua AI 115 does, or explicitly choose a skill through `CMob::LuaAttackAI`
(`0x004b25b0`). This evidence is specific to the audited EP4.5 server binary.
An isolated x86 run with an in-range target selected the highest-priority
ready spell in all 32 repetitions. With both spell slots cooling down it used
the basic attack, and it selected the top spell again at its exact deadline.

GX gives eligible routine spells (`Attackok` 1 or 2) a single 50% roll per
attack decision. On a failed roll, the mob can still use a basic or conditional
attack. If only routine spells are available, it waits for the ordinary attack
retry. A skipped spell stays ready. HP- and target-conditioned actions and
explicit scripted casts keep their authored rules.

### Generic cases, 0–9

| Value | Seed templates | Code-driven distinction |
| --- | --- | --- |
| 0 | 1,344 | Reactive generic behavior; idle wandering remains controlled by movement fields |
| 1 | 1,086 | Proximity acquisition through `Targeting`; no special help routine selected by this value |
| 2 | 2 | Reactive; initial `SetTarget(...,true)` recruits idle mobs with the **same MobID**, within the caller's ChaseRange |
| 3 | 199 | Proximity acquisition; initial target acquisition recruits idle mobs whose AI is **1 or 3**, within the caller's ChaseRange |
| 4 | 0 | Generic stationary case: idle wandering is suppressed and combat takes the non-chasing branch |
| 5 | 23 | Reactive; initial target acquisition recruits **any idle mob** in the scanned neighborhood/radius, without the same-MobID or AI=1/3 restriction |
| 6–9 | 0 each | No dedicated behavior branch found in the audited generic mob code; no file FSM is attached by this range |

Help routines scan adjacent cells and apply eligibility/range checks. Helpers
receive `SetTarget(...,false)`, preventing recursive help propagation. These are
not interchangeable “social AI” flags. Generic wandering/chasing also depends on
NormalStep, ChaseStep, timers, obelisk state and status effects.

## File-driven cases, 10–36

The supplied config is contiguous and ordered0..36. The factory stores its
entries in a vector and selects by AI index; preserving that order matters.
It selects by the **AI value**, not the MobID. Original filenames have mixed
case; a future Linux loader must handle the Windows package's case matching. Runtime
requires `.alo` (local variables), `.afs` (system-function declarations), `.ai`
(states, expressions and transitions) and `.ah` (native state callbacks).
`.PAI`, `.ac`, `_AI_FUNCTION.H`, `.fdb` and `.scc` are not loaded by this
`InitFSM` path. Having the editor/export files does not eliminate the native
callback and system-function implementation.

All 27 configured machines were parsed: **235 states and 353 transitions**.
Their start/target states, `.ah` callback lists and `.alo` variable declarations
were checked. Every variable used by these files has a declaration. HP fractions
below are the values in the files. Timers use milliseconds and strict `>` where
shown; these are eligibility thresholds, not guaranteed wall-clock firing times.
Numbers in skill descriptions are the arguments to native `UseMobSkill`.
The table summarizes scripts; ordered transitions and stage counters can prevent
an apparently declared later phase from executing (see anomalies).

| AI | Base file | Seed templates | Case-specific behavior |
| --- | --- | --- | --- |
| 10 | `sorp1` | 12 | At HP ≤40%, once per stage counter, set attack-speed multiplier to 2 |
| 11 | `croco1` | 3 | At HP ≤50%, recruit around current target with radius argument 10 and set strength multiplier 1.5; advance stage |
| 12 | `croco2` | 3 | At HP ≤50%, draw `GetRandInt`; values >60 take radius-7 recruitment plus strength 1.5, otherwise consume the stage without that boost |
| 13 | `troll1` | 8 | At HP ≤15%, set physical-defense multiplier 2 and advance stage |
| 14 | `spink1` | 6 | HP ≤35%: skill 95; HP ≤10% at the next stage: strength 2 and physical defense .5. Skill 65 after >60 s in Battle. Random opening speech 17/18 |
| 15 | `goldenpig` | 2 | Attack-type mask 0-only and battle type 1; idle/random walking, run while attacked, consume momentary-hit flag. A fresh hit after >7 s enables summon `(1967,10,1,0)`; >10 s running without returning to the hit branch resets target state |
| 16 | `sorp3` | 5 | At HP ≤20%, seek a non-battle mob with radius argument 12, move to selected position, then recruit around target with radius 7 and advance stage |
| 17 | `sorpnamed1` | 6 | HP ≤50%: attack mask 5, masked Battle for >5 s, then restore. HP ≤25% before final stage: skill 42, directional movement argument 15 after >1 s, await arrival, skill 63, then >1 s and final stage |
| 18 | `sorpnamed2` | 5 | HP ≤45%: skill 95, >1.5 s, skill 40, >1 s, advance stage. HP ≤20%: strength 1.5 |
| 19 | `sorp2` | 5 | Reactive entry; HP ≤40%: strength 1.5; HP ≤10% takes priority, records escape origin, recruits around target with radius 10, runs until escape distance ≥20 |
| 20 | `croconamed1` | 2 | Stage-gated HP 80/50/30/15%: skill 127; skill 95 → >1 s →65; skill127; strength2. Script calls the native `IsEndBattle` stub (see caveat) |
| 21 | `croconamed2` | 2 | Stage-gated HP 80/60/30/10%: skill18;178;95 → >1 s →72; attack-speed multiplier2 |
| 22 | `croconamed3` | 2 | HP 85% at stage0:34;60% at stage1:115; **ungated** HP ≤35%:95 → >1 s →65. Later HP≤10%/stage3 defense2 branch can be shadowed |
| 23 | `orcnamed1` | 0 | Stage-gated HP 75/60/30%:89;113;95 → >1.5 s →147 → >1 s |
| 24 | `orcnamed2` | 0 | **Ungated** HP≤70%:85; later declared HP≤45%/stage1:170 → >1 s →196; HP≤15%/stage2: mask5, wait>1 s, masked Battle>2.5 s, restore. First branch can shadow later phases |
| 25 | `orcnamed3` | 0 | Stage-gated HP 80/45/25%:123;169 → >1 s →123; attack-speed2. Its Battle state callbacks are **Empty**, unlike the other normal Battle callbacks |
| 26 | `trollnamed1` | 3 | HP≤50% stage0:65;≤30% stage1:66;≤10% stage2: physical-defense2. Skill34 after >180 s in Battle |
| 27 | `helltouthnamed1` | 4 | HP≤75% stage0:130;≤30% stage1:170. Skill104 after >180 s in Battle |
| 28 | `Deinos` | 2 | Independent clocks: skill253 after>90 s, skill246 after>180 s; strength2 after>150 s, maintained in a separate Battle1 for>30 s, then strength1 |
| 29 | `Parridalis` | 2 | Skill196 after>45 s; skill95 after>90 s; skill34 after>180 s. Strength2 after>120 s, separate Battle1>30 s, then strength1 |
| 30 | `Alcarian` | 2 | Skill58 after>45 s;95 after>120 s; strength2 after>90 s for>30 s. Once at HP≤10%, speech then>2 s, summon `(1703,1,1,4)`, skill254, advance stage |
| 31 | `ZinAlcaria` | 2 | Initial mask5 for>5 s, restore, then engage. Skill58 after>45 s;95 after>105 s; strength2.5 after>90 s for>30 s, then strength1 |
| 32 | `Belizabeth` | 2 | Skill18 after>60 s. After>300 s: enable recovery, interval1000 ms; after>1 s set recovery multiplier .35 and physical-defense .55; after>35 s restore both multipliers to1, disable recovery and reset both clocks |
| 33 | `Kirhiross` | 4 | Five successive Battle phases, each>120 s, call `SerChangeRegist(1..5)` with >1 s transition states. These change defensive element codes, not five generic spell immunities; includes speeches24/27/28/29/30 |
| 34 | `CrypticOne` | 2 | After>35 s choose eligible attacking user with highest current HP and skill251. After>240 s open a >45 s hit-count phase; each positive count heals `count*50000` via AddEnergy, then resets that count |
| 35 | `Haruhion` | 0 | Skill174 after>30 s and182 after>105 s on independent clocks |
| 36 | `Freezing` | 0 | First Battle transition is **unconditional** to skill174; later >105 s skill172 transition can be shadowed |

Seed examples include AI15 mob1968 (Gold Pig), AI28 mobs1704/1804, AI34
mobs1700/1800, AI33 mobs1705/1706/1805/1806. The EP5 client assigns AI35 to
mob1978 and AI36 to mobs1716/1816/1977, despite no current seed row using 35/36.
This illustrates why file presence, client metadata, DB assignments and active
spawns must be inventoried separately.

Two additional machines exist but are **not configured**: `scorlp1` has eight
states (retarget strongest attacker, HP40% strength/speed boosts and HP10%
escape); `testshaiya` has six demonstration states (walking, battle and speech).
They do not activate merely because a matching filename exists. Both were parsed
as well: 29 available machines total, 249 states; the configured subset remains
27/235/353. Private detailed annexes retain every state, action and ordered
transition rather than replacing the machines with HP-only summaries.

### Native functions whose names are misleading

The original `DoFunction` was traced against all 36 function IDs used by the
configured files. A declared `.afs` signature is not by itself the runtime
implementation; several prototype declarations have inconsistent types.
Some token ParamCount fields even contain the debug-fill value0xcdcdcdcd;
function arity must come from the actual declaration/evaluator, not that token field.

| Script call | Native effect / constraint |
| --- | --- |
| `GetEnergyPercent` | Returns float HP/maxHP in 0..1, unlike Lua's integer 0..100 argument |
| `GetCurrentMaxHpUser` | Despite the name, FindAttackUserMaxHP compares signed current HP at CUser+0x1238, not maximum health; the Damage function subtracts from that same field |
| `GetRandInt` | `rand()%100`, values 0..99; `>50` is not a symmetric 50/50 code split |
| `SetStr(f)` | Writes attack damage multiplier `CMob+0x268`; attack damage multiplies by it |
| `SetAttackSpeed(f)` | Writes `+0x26c`; the selected native attack delay is divided by this factor, not interpreted as a speed-tier index |
| `SetRec(f)` | Writes physical-defense multiplier `+0x270`, then rebuilds combat values; **not regeneration** |
| `SetHpChage(f)` | Writes recovery multiplier `+0x274`; recovery enable and interval are separate controls |
| `SerChangeRegist(n)` | Writes `m_nAttrDef`, initially loaded from mob Attrib; Kirhiross cycles codes1..5 |
| `SetEnableOnlyAttackType(n)` | Native four-byte mask: n=4 enables all, n=5 clears all, a normal index selects that entry. It is not a request to use attack slot5 |
| `SetEnableATAll` | Restores all four attack-type flags |
| `AttackUser(user,radius)` | Calls the native mob recruitment command around the selected user; not a direct single damage hit |
| `UseMobSkill(n)` | Uses the server's mob-skill lookup/native execution; not client animation slot n |
| `IsFirstAttack` | Actively invokes native FindTarget; it is not a passive “was the first hit mine?” flag |
| `IsAttackMeAtMoment` | Consumes/clears the momentary attacked flag when true |
| `IsEndBattle` | The branch in this binary is a diagnostic printf stub, without a useful typed result. AI20's use must not be silently reinterpreted as a working battle-ended predicate |

Native isolated x86 checks confirmed HP ratios and direct setter effects,
including attack mask5 and restore-all. SetRec's defense recalculation and
attack-speed division were additionally checked in their consumer instructions.

## Lua cases, 100 and above

`InitLuaFSM` supplies Idle, Combat, Death, ReturnHome and Move states and native
transitions. Each mob has its own Lua state and globals. The callbacks receive
native time and IDs; WhileCombat receives integer HP percent and attacked count.
Native attacks continue unless `LuaSetOnlyLuaAttack` changes the relevant flag.
The callback is not necessarily executed on the same update as a native attack:
`WhileCombat`'s native attack deadline and status guards affect dispatch.

| AI / file | Seed mob | Exact script-specific addition |
| --- | --- | --- |
| 111 / `111.lua` | 2464 | OnDeath requests instance portal update `(1001,0)`; other declared callbacks empty |
| 112 / `112.lua` | 2466 | Same portal request on death |
| 113 / `113.lua` | 2467 | Same portal request on death |
| 114 / `114.lua` | 2465 | Counter-limited skills: HP(60,80]→334 with counter<1; (40,60]→335 with counter<2; (20,40]→334 with counter<3; [0,20]→335 with counter<4. Each call increments by1; normal reset clears counter; death updates portal1001 |
| 115 / `115.lua` | 2471 | Init sets invulnerability. First attacked/acquired event says6401 and records time+5000. First combat callback says6402 under the literal timestamp condition, picks one of all six unordered pairs from mobs2464–2467, spawns the pair at(220,180), and requests fixed movement to(173,179). Later combat callbacks return after one-shot flag; OnMoveEnd holds position and requests removal of2471 |
| 116 / `116.lua` | 2470 | At HP≤50%, one speech6403 and deadline now+1000; once due, create one2472 and two2473, then request removal of2470. Subsequent callbacks can request removal again; declared normal-reset callback is empty |
| 117 / `117.lua` | 2472 | At HP≤30%, speech6404 once, first skill337 after3000 ms and then every3000 ms while the HP condition remains true. Normal reset clears timer/speech state; death removes two2473 |
| 118 / `118.lua` | 2469 | OnDeath creates one2470 at the native zero-coordinate fallback location |
| 104 / missing `104.lua` | 1890 | Assigned in seed but no script in either original package copy. Loader reports the error and clears Lua execution flags; the initialized native stack is not replaced with AI0 |
| Other100–255 | none in seed | Same filename construction; no matching script in this package. Do not invent behavior for absent files |

Comments inside some Lua headers mention a different AI number; **the actual
AI byte and filename determine dispatch**, not that comment. Zero coordinates in
the native create/delete APIs select the caller's location rather than blindly
using world origin. Portal update `(1001,0)` calls the instance controller, not a
universal “open every portal” operation.

## Literal behavior versus apparent intention

These are findings to decide upon later, not corrections applied to GX:

- **AI22:** HP≤35% lacks a stage guard; its ordered earlier transition can keep
  selecting skills95/65 and prevent the later HP10% defense phase.
- **AI24:** the first HP≤70% transition lacks a stage guard and can similarly
  prevent the declared lower-HP phases. AI25's Empty Battle callbacks must also
  be preserved as data, not automatically filled in as normal attacks.
- **AI36:** unconditional first Battle transition shadows the timed second one.
- **Lua114:** skipping directly to the last HP band can issue335 on successive
  callbacks until the counter reaches4. It does not reconstruct the missed bands.
- **Lua115:** its condition compares `dwNextSayTime` to5000, not `dwTime` to the
  deadline. Message6402 can occur in the first combat callback without a5s wait.
  The early return after the summon flag also makes callback ordering significant
  for its one-shot movement request.
- **Lua116:** its empty reset callback does not clear the phase flags; deletion
  requests repeat if the controller keeps invoking the callback after spawning.
- **AI20 / IsEndBattle:** file intent cannot turn the diagnostic native stub into
  a valid predicate. Declared library functions and real implementations differ.

## Validation and remaining limits

Read-only binary/file analysis established the dispatcher, all configured FSM
structures and their native call names. All start states and transition targets
exist; `.ai` callbacks match `.ah` in order. Original files were fingerprinted,
and duplicate package copies compared. Isolated LuaJIT5.1 tests ran the eight
unaltered scripts against logging-only native API stubs: **13 cases passed**,
including all six AI115 pairs, phase deadlines, skipped HP bands and reset behavior.

GX corrects AI115 in Caelum Sacra. A hit on the invulnerable controller starts
the first message, followed five seconds later by the second message, guard
summons and retreat. The controller is removed on arrival or after 30 seconds
if it cannot finish the move. The next-floor portal opens after both summoned
guards die in the instance.
An isolated x86 execution of original server code passed **12 API checks**.
Four unused prototype names (`Addint`, `Func`, `GetBloodType`, `TestArg`) have no
matching dispatcher branch in this binary; declarations alone do not add them.

This is not a run of the original world server. Lua tests establish script-level
call sequences, not network packets, movement arrival, spawn validity, portal
state, server scheduling or exact RNG sequences. The recovered FSM graph is
complete for these files; a future implementation must still reproduce the
native expression evaluator, type coercion, state callbacks, reset/return paths
and side effects. Where timing or malformed/stub behavior depends on the full
native runtime, this report does not claim dynamic parity.

Private evidence and a Spanish state-by-state annex live in ignored
`work/mob-ai-research/`. No original AI files or scripts are redistributed here.
No gameplay changes, rebuilds, installations or service restarts were performed.

## Integrated modern Lua catalog

The [Lua catalog](../server/ai/README.md) is now the default world AI, with
explicit host operations, per-mob state, boss associations and documented
original anomalies. The pure-Go interpreter runs policy and world-adapter tests;
external Lua tools and unassigned example scripts have been removed. The old Go
AI decision loop no longer runs. All numeric source families select Lua policies.
