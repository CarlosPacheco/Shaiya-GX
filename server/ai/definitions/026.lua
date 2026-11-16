-- AI 26: trollnamed1.
-- Selection: Mobs.AI = 26; never select this by MobID.
-- Mob 1635: Zululu.
-- Mob 1636: Lapalu.
-- Mob 1637: Pantanu.
-- HP≤50% stage0:65;≤30% stage1:66;≤10% stage2: physical-defense2. Skill34 after >180 s in Battle
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 26,
    name = "trollnamed1",
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
                { to = "SetSay", check = function(ctx, v)
                    local p1 = (ctx:find_target() == true)
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "SetSay", check = function(ctx, v)
                    local p1 = (ctx:was_attacked() == true)
                    local p2 = (v.stage == 0)
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
                { to = "UseMobSkill2", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.5))
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "UseMobSkill3", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.3))
                    local p2 = (v.stage == 1)
                    return p1 and p2
                end },
                { to = "BeStrong", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.1))
                    local p2 = (v.stage == 2)
                    return p1 and p2
                end },
                { to = "UseMobSkill1", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 180000)
                    return p1
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- SetSay: original named phase, preserving transition priority.
        ["SetSay"] = {
            enter = function(ctx, v)
                v.random_roll = num.i32(ctx:random_0_99())
            end,
            transitions = {
                { to = "Say1", check = function(ctx, v)
                    local p1 = (v.random_roll > 50)
                    return p1
                end },
                { to = "Say2" },
            },
        },
        -- Say1: original named phase, preserving transition priority.
        ["Say1"] = {
            enter = function(ctx, v)
                ctx:say_legacy(15)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- Say2: original named phase, preserving transition priority.
        ["Say2"] = {
            enter = function(ctx, v)
                ctx:say_legacy(14)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- UseMobSkill1: original named phase, preserving transition priority.
        ["UseMobSkill1"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(34)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- UseMobSkill2: original named phase, preserving transition priority.
        ["UseMobSkill2"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(65)
                v.stage = num.i32(1)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- BeStrong: original named phase, preserving transition priority.
        ["BeStrong"] = {
            enter = function(ctx, v)
                ctx:set_physical_defense_multiplier(num.f32(2.0))
                v.stage = num.i32(3)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- UseMobSkill3: original named phase, preserving transition priority.
        ["UseMobSkill3"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(66)
                v.stage = num.i32(2)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
    },
})
