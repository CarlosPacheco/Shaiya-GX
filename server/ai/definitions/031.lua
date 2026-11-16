-- AI 31: ZinAlcaria.
-- Selection: Mobs.AI = 31; never select this by MobID.
-- Mob 1703: Alcarian The Flame.
-- Mob 1803: Alcarian The Flame.
-- Initial mask5 for>5 s, restore, then engage. Skill58 after>45 s;95 after>105 s; strength2.5
-- after>90 s for>30 s, then strength1
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 31,
    name = "ZinAlcaria",
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
                    local p1 = (num.u32((ctx:time_ms() - v.secondary_clock_ms)) > 45000)
                    return p1
                end },
                { to = "UseMobSkill2", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.third_clock_ms)) > 105000)
                    return p1
                end },
                { to = "BeStrong", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 90000)
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
            enter = function(ctx, v)
                ctx:select_attack_type(5)
                v.phase_started_ms = num.u32(ctx:time_ms())
            end,
            transitions = {
                { to = "UnInvincible", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 5000)
                    return p1
                end },
            },
        },
        -- Say: original named phase, preserving transition priority.
        ["Say"] = {
            enter = function(ctx, v)
                ctx:announce_legacy(21, 200)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    v.secondary_clock_ms = num.u32(ctx:time_ms())
                    v.third_clock_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- BeStrong: original named phase, preserving transition priority.
        ["BeStrong"] = {
            enter = function(ctx, v)
                ctx:set_damage_multiplier(num.f32(2.5))
            end,
            transitions = {
                { to = "Battle1" },
            },
        },
        -- UseMobSkill1: original named phase, preserving transition priority.
        ["UseMobSkill1"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(58)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.secondary_clock_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- UseMobSkill2: original named phase, preserving transition priority.
        ["UseMobSkill2"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(95)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.third_clock_ms = num.u32(ctx:time_ms())
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
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
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
                { to = "Cancel", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 30000)
                    return p1
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- UnInvincible: original named phase, preserving transition priority.
        ["UnInvincible"] = {
            enter = function(ctx, v)
                ctx:enable_all_attack_types()
            end,
            transitions = {
                { to = "Say", check = function(ctx, v)
                    local p1 = (ctx:find_target() == true)
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "Say", check = function(ctx, v)
                    local p1 = (v.stage == 0)
                    local p2 = (ctx:was_attacked() == true)
                    return p1 and p2
                end },
            },
        },
    },
})
