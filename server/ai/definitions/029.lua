-- AI 29: Parridalis.
-- Selection: Mobs.AI = 29; never select this by MobID.
-- Mob 1701: Parrdalis.
-- Mob 1801: Parrdalis.
-- Skill196 after>45 s; skill95 after>90 s; skill34 after>180 s. Strength2 after>120 s, separate
-- Battle1>30 s, then strength1
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 29,
    name = "Parridalis",
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
        -- Battle: original named phase, preserving transition priority.
        ["Battle"] = {
            enter = function(ctx, v)
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "UseMobSkill1", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 45000)
                    return p1
                end },
                { to = "BeStrong1", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.secondary_clock_ms)) > 120000)
                    return p1
                end },
                { to = "UseMobSkill2", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.third_clock_ms)) > 180000)
                    return p1
                end },
                { to = "UseMobSkill3", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.fourth_clock_ms)) > 90000)
                    return p1
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
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
        -- Say: original named phase, preserving transition priority.
        ["Say"] = {
            enter = function(ctx, v)
                ctx:announce_legacy(18, 200)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    v.secondary_clock_ms = num.u32(ctx:time_ms())
                    v.third_clock_ms = num.u32(ctx:time_ms())
                    v.fourth_clock_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- BeStrong1: original named phase, preserving transition priority.
        ["BeStrong1"] = {
            enter = function(ctx, v)
                ctx:set_damage_multiplier(num.f32(2.0))
            end,
            transitions = {
                { to = "Battle1" },
            },
        },
        -- UseMobSkill1: original named phase, preserving transition priority.
        ["UseMobSkill1"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(196)
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
                ctx:use_mob_skill(34)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.third_clock_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- UseMobSkill3: original named phase, preserving transition priority.
        ["UseMobSkill3"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(95)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.fourth_clock_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- Cancel: original named phase, preserving transition priority.
        ["Cancel"] = {
            enter = function(ctx, v)
                ctx:set_damage_multiplier(num.f32(1.0))
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.secondary_clock_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- Battle1: original named phase, preserving transition priority.
        ["Battle1"] = {
            enter = function(ctx, v)
                v.secondary_clock_ms = num.u32(ctx:time_ms())
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "Cancel", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.secondary_clock_ms)) > 30000)
                    return p1
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
    },
})
