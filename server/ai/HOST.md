# AI context operations

This is the adapter contract for the prepared Lua catalog. No production adapter
is installed. An operation is a real world action/query, not an optional callback
that can silently do nothing. Only the test host logs actions without applying them.

## Scheduling and instance ownership

Create one `catalog.new(Mobs.AI, ctx)` per live mob generation. Destroy it when
that generation is removed; a new respawn gets new private state. Deliver normal
reset separately: original Lua116 intentionally keeps its flags on that event.
The adapter must serialize calls with authoritative mob/world state and honor
the existing controller ownership/lock rules for deferred summons and portals.

| Instance event | Arguments after ctx | Meaning |
| --- | --- | --- |
| `idle` | none | Native idle phase; file machines evaluate their ordered transitions here |
| `combat` | integer HP percent, attacked count | Native combat phase; encounter scripts add callbacks alongside native combat |
| `target` | target userID, request_help boolean | Initial/acquired target; generic AI may recruit, encounter callbacks ignore extra arguments |
| `attacked` | attacker userID | Original attacked callback; update native hit/aggro flags before delivering |
| `move_end` | none | Native selected movement completed, not merely a network position refresh |
| `return_home` | attacked count | Original return-home callback; host owns navigation and stack status |
| `death` | attacked count | Original death callback while caller location/controller context is still available |
| `reset` | none | Native normal reset; not script reload or a new generation |

The generic implementation also receives `combat` without using its extra
arguments. File-machine instances ignore combat/target/attacked/death notifications
because their original flag updates and life/status gating belong to the host.
Their numeric value must not accidentally send them through generic target finding.

The original Lua family uses a shared native Idle/Combat/Death/ReturnHome/Move
stack and still performs native attack AI. The bridge must drive that common
mechanism, including conditions that defer callbacks until an attack deadline.
Calling only the custom Lua combat callback would drop the boss's normal attacks.

## Queries and numeric types

| Method | Return / side effect |
| --- | --- |
| `time_ms()` | Original-compatible monotonic uint32 timestamp; keep wall-clock RNG seeding separate |
| `hp_fraction()` | Native float32 currentHP/maxHP,0..1; maxHP must be valid |
| `status()` | `idle`, `combat`, `return_home`, or another non-idle status |
| `acquisition_ready()` | Boolean honoring native acquisition delay and target/visibility rules |
| `find_target()` | Boolean; actively searches and sets a target, including target/deadline side effects |
| `was_attacked()` | Boolean snapshot of the non-consuming native attacked flag |
| `is_in_combat()` | Boolean native battle flag |
| `consume_recent_hit()` | Boolean; consumes the momentary attacked flag |
| `target_user_id()` | uint32,0 when no target |
| `highest_current_hp_attacker_id()` | Eligible attackerID with largest **current** HP; the old MaxHp name did not mean maximum pool |
| `highest_damage_attacker_id()` | Native damage-ledger selection used by the unassigned example |
| `hit_count()` | Native nonnegative counted-hit value while hit checking is enabled |
| `arrived_at_selected_position()` | Boolean native selected-position arrival test |
| `distance_from_escape_origin()` | Native distance from the saved escape point |
| `random_0_99()` | Native integer draw0..99; preserve host RNG ownership/order |
| `legacy_end_battle_stub()` | Nil/untyped unusable result, matching this binary's diagnostic stub; never invent `not is_in_combat()` |
| `reference_text(key)` | Resolve private localized dialogue for unassigned examples; not embedded in this catalog |

## Native mechanisms and world actions

| Method | Required meaning |
| --- | --- |
| `idle_wander(stationary)` | Native idle movement/home tests using NormalStep/NormalTime, obelisk, impairments and home radius; true suppresses wandering |
| `attack_ai(stationary)` | Native attack/slot/deadline/retarget/chase mechanics; stationary routing also honors ChaseStep0 and obelisk state |
| `battle()` | Original file-machine Battle callback, honoring the native next-attack time and target/reset rules |
| `random_walk()`, `random_run()` | Original file-machine movement callbacks; retain native timing and selected movement semantics |
| `move_to_selected_position()` | Continue selected movement and its arrival rules |
| `return_home()` | Native return navigation, recovery and status handling |
| `clear_target_flags()` | Native ResetUser: clear target, attacked and battle flags as permitted by vision-fix rules |
| `reset_generic()` | Native generic reset behavior; not a respawn |
| `recruit_idle(user, kind)` | kind=`same_template`, `aggressive` or `all_idle`; native adjacent-cell/range/eligibility rules, helpers acquire without recursively recruiting |
| `recruit_around_user(user, radius)` | Original AttackUser recruitment command around the selected user; not a direct damage action |
| `select_idle_ally_position(radius)` | Original SuchFrendlyandRun selection/movement setup |
| `record_escape_origin()` | Save native escape point |
| `select_escape_direction(extra_distance)` | Original SetDrMove selected direction/position |
| `select_target_user(id)` | Native selected target update and eligibility behavior |
| `use_mob_skill(id)` | Authoritative original mob-skill lookup and execution; resolve costs, range, cooldowns and effects, never a raw animation request |
| `set_damage_multiplier(f)` | Native m_fStr damage factor |
| `set_attack_speed_multiplier(f)` | Divide selected native attack delay by this factor, not a speed-tier index |
| `set_physical_defense_multiplier(f)` | Native m_fPhysicalDefense and combat-stat rebuild |
| `set_recovery_multiplier(f)` | Native recovery ratio; do not reinterpret as immediate percentage-maxHP healing |
| `enable_recovery(flag)` | Integer0/1, original recovery enable |
| `set_recovery_interval_ms(ms)` | Original recovery period |
| `set_defensive_element(code)` | Native m_nAttrDef, including original Kirhiross1..5 sequence |
| `select_attack_type(code)` | Four-category mask: normal index selects one;4 enables all;5 clears all |
| `enable_all_attack_types()` | Restore the four native attack-category flags |
| `set_battle_type(code)` | Original PAI battle-type field; independent of skill slots |
| `summon_legacy(mob, distance, count, mode)` | Original SetSummonMob/PAI_CreateMob argument semantics, including mode and validation |
| `begin_hit_count()`, `end_hit_count()`, `reset_hit_count()` | Native attack-count bookkeeping |
| `heal(amount)` | Original AddEnergy semantics and authoritative HP publication |
| `say_legacy(value)` | Original PAI_Say mapping, including localized reference strings where applicable |
| `announce_legacy(index, radius)` | Original PAI_Say2 mapping, kept distinct from LuaSayByIndex |
| `announce(index, radius)` | Original LuaSayByIndex semantics |
| `create_mobs(id, count, x, z)` | Original LuaCreateMob validation/controller ownership; x=z=0 means caller's location |
| `remove_mobs(id, count, x, z)` | Original LuaDeleteMob selection near requested/caller position, not a global template deletion |
| `fixed_move(mode, x, z)` | Original LuaFixedMove mode and arrival event |
| `hold_position(boolean)` | Original LuaSetHoldPosion behavior |
| `set_invulnerable(boolean)` | Original LuaSetUnBeatable flag |
| `update_instance_portal(index, country)` | Original instance-controller portal request; never a global open-all |
| `seed_lua_random_from_wall_clock()` | Original Lua115 load-time RNG seeding, owned by host |
| `lua_random(max)` | Original Lua-family integer draw1..max; native RNG sequence replication is a separate host concern |
| `disable_lua_callbacks(ai)` | Missing-script error path: disable Lua hooks while preserving the native stack, not AI0 substitution |

Implement the common mechanisms against the original functions listed in the
[research](../../docs/mob-ai-research.md). The interface names are clearer than
the original ones; their names alone do not authorize replacing native behavior
with an approximate implementation. The prepared scripts are not a claim that
this bridge, its scheduling or all world side effects have already been ported.
