# Mob AI in modern Lua

This is the **default, authoritative AI catalog** of GX World. All mob AI
selection and encounter decisions run in these Lua files. The former Go-only
generic decision loop has been removed. Go supplies world primitives: validated
movement, combat, recovery, spawning, removal, speech and instance portals.

The integrated pure-Go [golua](https://github.com/arnodel/golua) v0.4.1 runtime
supports modern Lua 5.5. No external interpreter, shared library, Python bridge
or legacy `.ai/.alo/.afs/.ah` parser is used.

## Selection and deployment

`catalog.resolve(ai_byte)` selects by **Mobs.AI**, never MobID. Each definition's
English comments describe its rule, associated seed mobs/bosses and original
anomalies. These names document the mapping; they do not override selection.

| Files | Behavior |
| --- | --- |
| `definitions/000.lua`–`009.lua` | Former native rules, including distinct social help for2/3/5 |
| `definitions/010.lua`–`036.lua` | All27 original configured machines, with named states and ordinary Lua expressions |
| `definitions/111.lua`–`118.lua` | Eight encounters with private per-mob closures |
| `definitions/104.lua` | Original missing-script case, retaining the common stack without invented mechanics |
| `core/missing.lua` | Original missing-source routes for other unconfigured byte values |
| `core/` | Shared lifecycle, ordered transitions, native numeric boundaries and Lua-family scheduling |
| `tests/` | Policy regression fixtures, executed by Go tests with the production interpreter |

There are46 explicit definitions. All256 byte values resolve; the original did
not provide256 distinct behaviors. Unassigned example scripts are excluded.

The executable embeds this same catalog. At startup, GX loads a sibling `ai/`
directory beside its data directory when present; portable packages include it.
`RAPTURE_AI_DIR` selects an explicit directory. Otherwise the embedded catalog
runs. A broken selected catalog stops startup; it never enables old Go AI.
The sources are read and compiled once, at startup, and never again: edits take effect only on restart. Keep the complete catalog together.

One VM belongs to each map shard; compiled code is reused. Each mob generation
has its own actor/variables. Respawn creates fresh state; normal reset delivers
a separate event (notably, original Lua116 keeps its phase flags on reset).
Calls are serialized with authoritative state. Summons and portal/speech actions
are deferred outside Lua execution, avoiding controller re-entry and mob-slice
mutation during callbacks. Deferred actions cannot outlive their mob generation.

Scripts receive only the documented context and base/math/string/table libraries.
OS/filesystem/debug/load facilities are unavailable. Each call has CPU and memory
limits. A runtime error logs mob/global/AI IDs and stops that actor; it does not
silently replace a boss with generic behavior. See [HOST.md](HOST.md).

## Original semantics

File-machine transitions preserve priority and enter/update/leave order. All
predicates are evaluated, including actions in failed candidates. There is one
transition scan per idle update; repeated Battle callbacks honor attack deadlines.
`core/numbers.lua` preserves float32 thresholds and32-bit machine variables/clocks.
File machines receive HP0..1; encounter callbacks receive integer HP0..100.

`set_physical_defense_multiplier` is the original misleading `SetRec`, not healing.
Attack speed divides the native interval; attack masks have four categories,
with4 enabling all and5 disabling all. Summons retain their requested mask.
The reference executable writes these flags but has no observed combat read of
that field; GX does not invent incoming-damage immunity from this setter.
Native Lua-family attacks continue alongside custom callbacks, even for missing
scripts. Lua selects the scheduling in `core/scripted.lua`.

Original anomalies remain explicit: shadowed phases22/24/36, Empty Battle25,
`IsEndBattle`'s diagnostic stub, Huistaton's skipped bands, controller115's time
comparison/early return, and116's empty reset. They are preserved intentionally.

## Validation

From the repository root:

```sh
tools/check.sh ./server/ai ./server/internal/world
```

The51 policy scenarios cover every numeric route, independent actors, all
27/235/353 machine/state/transition totals, help, HP thresholds, clocks,
non-short-circuit side effects, six controller pairs, splits and resets.
`TestLuaWorld*` also exercises real target acquisition, recruitment, combat
factors/cooldowns, invulnerability before first tick, summons, movement,
removal, portal ownership, stale generations and native speech packet decoding.

A private original-token differential check previously matched all235 states
under32 inputs each (**7,520 cases**). It requires original assets and is not
distributed. Automated coverage establishes the tested rules, not a claim that
every boss has received a complete interactive encounter playthrough.

Research: [original AI inventory](../../docs/mob-ai-research.md).
