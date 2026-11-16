-- AI 19: sorp2.
-- Selection: Mobs.AI = 19; never select this by MobID.
-- Mob 1507: Piperr Clan Tation.
-- Mob 1510: Capsiwoo Clan Laosha.
-- Mob 1513: Alimoss Clan Tiennya.
-- Mob 1543: Erucan Clan Eremi.
-- Mob 1545: Apisus Clan Loyiore.
-- Reactive entry; HP ≤40%: strength 1.5; HP ≤10% takes priority, records escape origin, recruits
-- around target with radius 10, runs until escape distance ≥20
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 19,
    name = "sorp2",
    family = "states",
    initial = "init",
    variables = function()
        return { stage = 0, phase_started_ms = 0, target_user_id = 0, random_roll = 0, secondary_clock_ms = 0 }
    end,
    states = {
        -- init: original named phase, preserving transition priority.
        ["init"] = {
            transitions = {
                { to = "idle", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- idle: original named phase, preserving transition priority.
        ["idle"] = {
            transitions = {
                { to = "battle", check = function(ctx, v)
                    local p1 = (ctx:was_attacked() == true)
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "init", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 5000)
                    return p1
                end },
            },
        },
        -- battle: original named phase, preserving transition priority.
        ["battle"] = {
            enter = function(ctx, v)
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "runaway", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.1))
                    local p2 = (v.stage < 2)
                    return p1 and p2
                end },
                { to = "applyvalue", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.4))
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- applyvalue: original named phase, preserving transition priority.
        ["applyvalue"] = {
            enter = function(ctx, v)
                v.stage = num.i32(1)
                ctx:set_damage_multiplier(num.f32(1.5))
            end,
            transitions = {
                { to = "battle" },
            },
        },
        -- runaway: original named phase, preserving transition priority.
        ["runaway"] = {
            enter = function(ctx, v)
                v.stage = num.i32(2)
                ctx:record_escape_origin()
                v.target_user_id = num.u32(ctx:target_user_id())
                ctx:recruit_around_user(v.target_user_id, num.f32(10.0))
                ctx:random_run()
            end,
            update = function(ctx, v)
                ctx:random_run()
            end,
            transitions = {
                { to = "battle", check = function(ctx, v)
                    local p1 = (ctx:distance_from_escape_origin() >= 20)
                    return p1
                end },
            },
        },
    },
})
