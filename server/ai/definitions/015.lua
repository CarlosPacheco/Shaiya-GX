-- AI 15: goldenpig.
-- Selection: Mobs.AI = 15; never select this by MobID.
-- Mob 1940: Error Monster1940.
-- Mob 1968: Gold Pig.
-- Attack-type mask 0-only and battle type 1; idle/random walking, run while attacked, consume
-- momentary-hit flag. A fresh hit after >7 s enables summon `(1967,10,1,0)`; >10 s running
-- without returning to the hit branch resets target state
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 15,
    name = "goldenpig",
    family = "states",
    initial = "Init",
    variables = function()
        return { stage = 0, phase_started_ms = 0, target_user_id = 0, random_roll = 0, secondary_clock_ms = 0 }
    end,
    states = {
        -- Init: original named phase, preserving transition priority.
        ["Init"] = {
            enter = function(ctx, v)
                ctx:select_attack_type(0)
                ctx:set_battle_type(1)
            end,
            transitions = {
                { to = "returninit" },
            },
        },
        -- Move: original named phase, preserving transition priority.
        ["Move"] = {
            enter = function(ctx, v)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:random_walk()
            end,
            update = function(ctx, v)
                ctx:random_walk()
            end,
            transitions = {
                { to = "settime", check = function(ctx, v)
                    local p1 = (ctx:was_attacked() == true)
                    return p1
                end },
                { to = "returninit" },
            },
        },
        -- Idle: original named phase, preserving transition priority.
        ["Idle"] = {
            transitions = {
                { to = "settime", check = function(ctx, v)
                    local p1 = (ctx:was_attacked() == true)
                    return p1
                end },
                { to = "Move", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 2000)
                    return p1
                end },
            },
        },
        -- SummonMonster: original named phase, preserving transition priority.
        ["SummonMonster"] = {
            enter = function(ctx, v)
                ctx:summon_legacy(1967, num.f32(10.0), 1, 0)
            end,
            transitions = {
                { to = "settime" },
            },
        },
        -- Battle: original named phase, preserving transition priority.
        ["Battle"] = {
            enter = function(ctx, v)
                v.secondary_clock_ms = num.u32(ctx:time_ms())
                ctx:random_run()
            end,
            update = function(ctx, v)
                ctx:random_run()
            end,
            transitions = {
                { to = "IsFirst", check = function(ctx, v)
                    local p1 = (ctx:consume_recent_hit() == true)
                    return p1
                end },
                { to = "returninit", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.secondary_clock_ms)) > 10000)
                    return p1
                end },
            },
        },
        -- IsFirst: original named phase, preserving transition priority.
        ["IsFirst"] = {
            transitions = {
                { to = "SummonMonster", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 7000)
                    return p1
                end },
                { to = "Battle" },
            },
        },
        -- settime: original named phase, preserving transition priority.
        ["settime"] = {
            enter = function(ctx, v)
                v.phase_started_ms = num.u32(ctx:time_ms())
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- returninit: original named phase, preserving transition priority.
        ["returninit"] = {
            enter = function(ctx, v)
                ctx:clear_target_flags()
            end,
            transitions = {
                { to = "Idle", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
    },
})
