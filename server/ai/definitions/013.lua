-- AI 13: troll1.
-- Selection: Mobs.AI = 13; never select this by MobID.
-- Mob 1617: Dark Fear Troll Fighter.
-- Mob 1618: Dark Fear Troll Gladiator.
-- Mob 1619: Dark Fear Troll Warrior.
-- Mob 1620: Dark Fear Troll Commander.
-- Mob 1621: Mistburn HellTooth Fighter....
-- Mob 1622: Mistburn HellTooth Archer.
-- Mob 1623: Mistburn HellTooth Warrior....
-- Mob 1624: Mistburn HellTooth Command....
-- At HP ≤15%, set physical-defense multiplier 2 and advance stage
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 13,
    name = "troll1",
    family = "states",
    initial = "Init",
    variables = function()
        return { stage = 0, phase_started_ms = 0, target_user_id = 0, random_roll = 0, secondary_clock_ms = 0 }
    end,
    states = {
        -- Init: original named phase, preserving transition priority.
        ["Init"] = {
            transitions = {
                { to = "Idle", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
                end },
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
                    local p1 = (ctx:hp_fraction() <= num.f32(0.15))
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
                ctx:set_physical_defense_multiplier(num.f32(2.0))
                v.stage = num.i32(1)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
    },
})
