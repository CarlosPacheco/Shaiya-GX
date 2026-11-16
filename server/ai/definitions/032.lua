-- AI 32: Belizabeth.
-- Selection: Mobs.AI = 32; never select this by MobID.
-- Mob 1702: Blizabeth Eathory.
-- Mob 1802: Blizabeth Eathory.
-- Skill18 after>60 s. After>300 s: enable recovery, interval1000 ms; after>1 s set recovery
-- multiplier .35 and physical-defense .55; after>35 s restore both multipliers to1, disable
-- recovery and reset both clocks
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 32,
    name = "Belizabeth",
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
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "UseMobSkill1", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 60000)
                    return p1
                end },
                { to = "BattleStop", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.secondary_clock_ms)) > 300000)
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
                ctx:announce_legacy(22, 200)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    v.secondary_clock_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- UseMobSkill1: original named phase, preserving transition priority.
        ["UseMobSkill1"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(18)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- HpReChage: original named phase, preserving transition priority.
        ["HpReChage"] = {
            enter = function(ctx, v)
                ctx:set_recovery_multiplier(num.f32(0.35))
                ctx:set_physical_defense_multiplier(num.f32(0.55))
                v.secondary_clock_ms = num.u32(ctx:time_ms())
            end,
            transitions = {
                { to = "HpChageCancel", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.secondary_clock_ms)) > 35000)
                    return p1
                end },
            },
        },
        -- BattleStop: original named phase, preserving transition priority.
        ["BattleStop"] = {
            enter = function(ctx, v)
                v.secondary_clock_ms = num.u32(ctx:time_ms())
                ctx:announce_legacy(23, 200)
                ctx:enable_recovery(1)
                ctx:set_recovery_interval_ms(1000)
            end,
            transitions = {
                { to = "HpReChage", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.secondary_clock_ms)) > 1000)
                    return p1
                end },
            },
        },
        -- HpChageCancel: original named phase, preserving transition priority.
        ["HpChageCancel"] = {
            enter = function(ctx, v)
                ctx:set_recovery_multiplier(num.f32(1.0))
                ctx:set_physical_defense_multiplier(num.f32(1.0))
                ctx:enable_recovery(0)
            end,
            transitions = {
                { to = "Say1", check = function(ctx, v)
                    v.secondary_clock_ms = num.u32(ctx:time_ms())
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- Say1: original named phase, preserving transition priority.
        ["Say1"] = {
            enter = function(ctx, v)
                ctx:announce_legacy(33, 200)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
    },
})
