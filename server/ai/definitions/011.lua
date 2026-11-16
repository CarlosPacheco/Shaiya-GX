-- AI 11: croco1.
-- Selection: Mobs.AI = 11; never select this by MobID.
-- Mob 1600: Terrtier Bandilus.
-- Mob 1605: Lacerrtier Herius.
-- Mob 1606: Lacerrtier Curius.
-- At HP ≤50%, recruit around current target with radius argument 10 and set strength multiplier
-- 1.5; advance stage
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 11,
    name = "croco1",
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
                    local p1 = (ctx:find_target() == true)
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "battle", check = function(ctx, v)
                    local p1 = (ctx:was_attacked() == true)
                    local p2 = (v.stage == 0)
                    return p1 and p2
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
                { to = "BeStrong", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.5))
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- BeStrong: original named phase, preserving transition priority.
        ["BeStrong"] = {
            enter = function(ctx, v)
                v.target_user_id = num.u32(ctx:target_user_id())
                ctx:recruit_around_user(v.target_user_id, num.f32(10.0))
                v.stage = num.i32(1)
                ctx:set_damage_multiplier(num.f32(1.5))
            end,
            transitions = {
                { to = "battle" },
            },
        },
    },
})
