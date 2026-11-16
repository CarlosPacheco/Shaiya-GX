-- AI 10: sorp1.
-- Selection: Mobs.AI = 10; never select this by MobID.
-- Mob 1505: Piperr Worker Scorpion.
-- Mob 1508: Capsiwoo Worker Scorpion.
-- Mob 1511: Alimoss Worker Scorpion.
-- Mob 1516: Orichal Ruin Shaman.
-- Mob 1517: Orichal Ruin Great Magicia....
-- Mob 1524: Alimoss Guard Scorpion.
-- Mob 1525: Capsiwoo Guard Scorpion.
-- Mob 1526: Piperr Guard Scorpion.
-- Mob 1555: Erucan Patrol Scorpion.
-- Mob 1556: Erucan Worker Scorpion.
-- Mob 1557: Apisus Patrol Scorpion.
-- Mob 1558: Apisus Worker Scorpion.
-- At HP ≤40%, once per stage counter, set attack-speed multiplier to 2
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 10,
    name = "sorp1",
    family = "states",
    initial = "Init",
    variables = function()
        return { stage = 0, phase_started_ms = 0, target_user_id = 0, random_roll = 0, secondary_clock_ms = 0 }
    end,
    states = {
        -- Init: original named phase, preserving transition priority.
        ["Init"] = {
            transitions = {
                { to = "Idle" },
            },
        },
        -- Idle: original named phase, preserving transition priority.
        ["Idle"] = {
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    local p1 = (ctx:find_target() == true)
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "Battle", check = function(ctx, v)
                    local p1 = (ctx:was_attacked() == true)
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
            },
        },
        -- Battle: original named phase, preserving transition priority.
        ["Battle"] = {
            enter = function(ctx, v)
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "BeStrong", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.4))
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- BeStrong: original named phase, preserving transition priority.
        ["BeStrong"] = {
            enter = function(ctx, v)
                ctx:set_attack_speed_multiplier(num.f32(2.0))
                v.stage = num.i32(1)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
    },
})
