# Mob AI in modern Lua

This directory is a **prepared, executable policy catalog**, not yet connected
to the GX world server. Its tests execute Lua directly with a logging host.
Adding or editing these files currently changes no installed mob behavior.
Activation and the Go world bridge are a separate step.

The catalog targets [Lua 5.4](https://www.lua.org/manual/5.4/) or later and is
validated on Lua 5.4 and 5.5. It has no LuaJIT/5.1 dependency. No Go interpreter
library, runtime dependency, database value or executable was changed to prepare it.

## Selection and layout

`catalog.resolve(ai_byte)` selects a definition; `catalog.new(ai_byte, ctx)`
creates its private state. Selection always uses `Mobs.AI`, never MobID.
Comments in each definition identify the rule, the current seed's associated
mobs, boss names and any client-only references. Names are descriptive references;
they do not change which database value selects the script.

| Location / values | Purpose |
| --- | --- |
| `definitions/000.lua`–`009.lua` | Former code-driven rules, including the distinct help rules2/3/5 |
| `definitions/010.lua`–`036.lua` | All27 configured original file machines, now ordinary Lua functions with named states and variables |
| `definitions/111.lua`–`118.lua` | Eight encounter scripts rewritten with per-mob closures and explicit host operations |
| `definitions/104.lua` | Documented missing-original-script case for mob1890; no invented boss mechanics |
| `core/missing.lua` | Explicit handling of other unconfigured factory37–99 and missing Lua100–255 values |
| `reference/` | Two unconfigured original examples. They have no guessed numeric AI assignment |
| `core/` | Shared Lua mechanics: state transitions, generic phases, callback instances and native numeric boundaries |
| `tests/` | A strict logging host and behavior regressions; no world-server side effects |

There are46 explicit numeric definitions and two unassigned references. All256
byte values resolve, including original missing-source cases; this does not mean
that the original supplied256 distinct behaviors.

The old `.ai/.alo/.afs/.ah` formats and token interpreter are not runtime inputs.
The new files contain direct Lua expressions and calls. State names retain their
original correspondence for review, while variables use names such as `stage`,
`secondary_clock_ms` and `target_user_id`. Calls say what they mean, for example
`set_physical_defense_multiplier` rather than the misleading original `SetRec`.

## Reading a boss script

In `definitions/028.lua`, Deinos uses independent clocks for skills253 and246,
and a separate empowered-combat state for its30-second damage multiplier.
Each transition is listed in original priority order; its predicates, timer
updates and destination are visible in the same block. The header identifies
mobs1704 and1804. The shared helper performs one transition scan per idle update,
then updates the resulting state; it does not hide boss conditions in Go.

In `definitions/116.lua`, Dios Exiel's splitting phase names its announcement,
creation flag and deadline directly. The comments identify real boss2472 and
clones2473, explain zero coordinates, and point out the deliberately empty reset.

## Original semantics retained

- Each mob gets independent variables/closures. Module loading does not share
  encounter progress between mobs. Recreating an instance and normal reset are
  distinct operations, especially for legacy Lua cases with empty reset callbacks.
- File-machine transition priority, named states, strict time comparisons and
  native enter/update/leave callback order are preserved. Actions in a candidate
  transition execute even when a later/earlier predicate fails; predicate calls
  are evaluated without introducing Lua short-circuit behavior.
- The state machine's native callbacks still need their native mechanics:
  calling `battle()` during both entry and update must respect attack deadlines,
  not cause two unconditional swings.
- `core/numbers.lua` keeps float32 thresholds and32-bit clock/variable arithmetic
  where the original file evaluator used them. Legacy Lua timers stay Lua-number
  arithmetic on the host's supplied native timestamp, matching those scripts.
- HP fraction0..1 for file machines differs from the integer percentage0..100
  supplied to encounter combat callbacks.
- Recruitment, recovery, defensive element and attack-type mask operations have
  explicit meanings. In particular `SetRec` was defense, not healing, and mask5
  clears the four attack-type flags rather than selecting attack slot5.
- Original anomalies remain visible: shadowed phases22/24/36, Empty Battle25,
  `IsEndBattle`'s unusable native stub, Huistaton's skipped-band behavior,
  controller115's timestamp comparison and early return, and116's empty reset.
  Altering any of these is a gameplay decision, not part of this translation.
- The two unassigned examples reference original dialogue through
  `reference_text(key)` instead of embedding proprietary localized text.

## Host integration contract

See [HOST.md](HOST.md). A future Go bridge supplies world operations and native
mechanics while Lua selects AI behavior. It must not parse old AI files or add
per-AI-code behavior branches behind these calls.

All instances expose `event(ctx, name, ...)` and `tick(...)`. Event names are
`idle`, `combat`, `target`, `attacked`, `move_end`, `return_home`, `death`, and
`reset`; only applicable events perform work in a given definition. The host
must preserve original scheduling/status guards, target acquisition side effects
and the common native Lua stack behavior. The `tick` helpers are test conveniences,
not permission to call every phase unconditionally at rendering cadence.

The catalog is not a complete Go world adapter. Actual movement, damage, skill
validation, summons, portal updates, packet ordering, script loading, error
handling and instance lifetime still require integration and game-level tests.
A missing host operation must fail explicitly; the logging test host is never a
production implementation. Scripts do not access OS time/files directly; clock,
randomness, announcements and world effects go through the context.

## Validation

From the repository root, with a Lua5.4 shared library installed:

```sh
python3 server/tools/check_ai_lua.py
```

Or select another modern Lua shared library:

```sh
python3 server/tools/check_ai_lua.py --library /path/to/liblua.so
```

With a Lua CLI, from this directory:

```sh
lua tests/run.lua
```

The53 checked-in scenarios cover every numeric route, independent instances,
original graph totals27/235/353, help policies, HP thresholds, strict clocks,
non-short-circuit side effects, shadowed phases, all six controller115 pairs,
phase splitting, resets and missing-source cases. All configured machines and
both unassigned examples receive changing-input smoke checks.

A separate local differential check compared all235 configured states against
a reference evaluation of original tokens under32 inputs each: **7,520 cases**
agree on destination, variable values and host-call trace. That check requires
private original assets and is not redistributed. Neither that comparison nor
the public mock tests establish integrated world-server or packet parity.

Research and native evidence: [original AI inventory](../../docs/mob-ai-research.md).
