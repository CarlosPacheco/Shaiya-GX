-- AI 33: Kirhiross.
-- Selection: Mobs.AI = 33; never select this by MobID.
-- Mob 1705: Kirhiross.
-- Mob 1706: Nantarios.
-- Mob 1805: Kirhiross.
-- Mob 1806: Nantarios.
-- Five successive Battle phases, each>120 s, call `SerChangeRegist(1..5)` with >1 s transition
-- states. These change defensive element codes, not five generic spell immunities; includes
-- speeches24/27/28/29/30
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 33,
    name = "Kirhiross",
    family = "states",
    initial = "Init",
    variables = function()
        return { stage = 0, phase_started_ms = 0, target_user_id = 0, random_roll = 0, secondary_clock_ms = 0, third_clock_ms = 0, fourth_clock_ms = 0 }
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
                { to = "Say", check = function(ctx, v)
                    local p1 = (v.stage == 0)
                    local p2 = (ctx:find_target() == true)
                    return p1 and p2
                end },
                { to = "Say", check = function(ctx, v)
                    local p1 = (v.stage == 0)
                    local p2 = (ctx:was_attacked() == true)
                    return p1 and p2
                end },
            },
        },
        -- Battle: original named phase, preserving transition priority.
        ["Battle"] = {
            enter = function(ctx, v)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "RegChange1", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 120000)
                    return p1
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- Say: original named phase, preserving transition priority.
        ["Say"] = {
            enter = function(ctx, v)
                ctx:announce_legacy(8, 200)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- RegChange1: original named phase, preserving transition priority.
        ["RegChange1"] = {
            enter = function(ctx, v)
                ctx:set_defensive_element(1)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:announce_legacy(24, 200)
            end,
            transitions = {
                { to = "Battle1", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 1000)
                    return p1
                end },
            },
        },
        -- Battle1: original named phase, preserving transition priority.
        ["Battle1"] = {
            enter = function(ctx, v)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "RegChange2", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 120000)
                    return p1
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- RegChange2: original named phase, preserving transition priority.
        ["RegChange2"] = {
            enter = function(ctx, v)
                ctx:set_defensive_element(2)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:announce_legacy(27, 200)
            end,
            transitions = {
                { to = "Battle2", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 1000)
                    return p1
                end },
            },
        },
        -- Battle2: original named phase, preserving transition priority.
        ["Battle2"] = {
            enter = function(ctx, v)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "RegChange3", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 120000)
                    return p1
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- RegChange3: original named phase, preserving transition priority.
        ["RegChange3"] = {
            enter = function(ctx, v)
                ctx:set_defensive_element(3)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:announce_legacy(28, 200)
            end,
            transitions = {
                { to = "Battle3", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 1000)
                    return p1
                end },
            },
        },
        -- Battle3: original named phase, preserving transition priority.
        ["Battle3"] = {
            enter = function(ctx, v)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "RegChange4", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 120000)
                    return p1
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- RegChange4: original named phase, preserving transition priority.
        ["RegChange4"] = {
            enter = function(ctx, v)
                ctx:set_defensive_element(4)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:announce_legacy(29, 200)
            end,
            transitions = {
                { to = "Battle4", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 1000)
                    return p1
                end },
            },
        },
        -- Battle4: original named phase, preserving transition priority.
        ["Battle4"] = {
            enter = function(ctx, v)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "RegChange5", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 120000)
                    return p1
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- RegChange5: original named phase, preserving transition priority.
        ["RegChange5"] = {
            enter = function(ctx, v)
                ctx:set_defensive_element(5)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:announce_legacy(30, 200)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 1000)
                    return p1
                end },
            },
        },
    },
})
