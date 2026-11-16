-- AI 36: Freezing.
-- Selection: Mobs.AI = 36; never select this by MobID.
-- No current seed assignment; keep the original case available explicitly.
-- Client reference only: mob 1716, Cryptic the Immortal.
-- Client reference only: mob 1816, Cryptic the Immortal.
-- Client reference only: mob 1977, Freezing Mirage.
-- First Battle transition is **unconditional** to skill174; later >105 s skill172 transition can
-- be shadowed
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 36,
    name = "Freezing",
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
        -- Say: original named phase, preserving transition priority.
        ["Say"] = {
            enter = function(ctx, v)
                ctx:announce_legacy(31, 200)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    v.secondary_clock_ms = num.u32(ctx:time_ms())
                    return true
                end },
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
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "UseMobSkill1" },
                { to = "UseMobSkill2", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.secondary_clock_ms)) > 105000)
                    return p1
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- UseMobSkill1: original named phase, preserving transition priority.
        ["UseMobSkill1"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(174)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- UseMobSkill2: original named phase, preserving transition priority.
        ["UseMobSkill2"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(172)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.secondary_clock_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
    },
})
