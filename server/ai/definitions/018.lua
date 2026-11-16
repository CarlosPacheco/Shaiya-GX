-- AI 18: sorpnamed2.
-- Selection: Mobs.AI = 18; never select this by MobID.
-- Mob 1519: Shultan Talat Alimoss.
-- Mob 1520: High Priest Doorumnan Pulo....
-- Mob 1523: Shultan Saladis Piper.
-- Mob 1560: Champion Keratis.
-- Mob 1562: Champion Hyodoras.
-- HP ≤45%: skill 95, >1.5 s, skill 40, >1 s, advance stage. HP ≤20%: strength 1.5
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 18,
    name = "sorpnamed2",
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
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "UseMobSkill1", check = function(ctx, v)
                    local p1 = (v.stage == 0)
                    local p2 = (ctx:hp_fraction() <= num.f32(0.45))
                    return p1 and p2
                end },
                { to = "BeStrong", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.2))
                    local p2 = (v.stage < 2)
                    return p1 and p2
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
                ctx:say_legacy(3)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- Say2: original named phase, preserving transition priority.
        ["Say2"] = {
            enter = function(ctx, v)
                ctx:say_legacy(4)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- UseMobSkill1: original named phase, preserving transition priority.
        ["UseMobSkill1"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(95)
                v.phase_started_ms = num.u32(ctx:time_ms())
            end,
            transitions = {
                { to = "UseMobSkill2", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 1500)
                    return p1
                end },
            },
        },
        -- UseMobSkill2: original named phase, preserving transition priority.
        ["UseMobSkill2"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(40)
                ctx:say_legacy(9)
                v.phase_started_ms = num.u32(ctx:time_ms())
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 1000)
                    v.stage = num.i32(1)
                    return p1
                end },
            },
        },
        -- BeStrong: original named phase, preserving transition priority.
        ["BeStrong"] = {
            enter = function(ctx, v)
                ctx:set_damage_multiplier(num.f32(1.5))
                v.stage = num.i32(2)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
    },
})
