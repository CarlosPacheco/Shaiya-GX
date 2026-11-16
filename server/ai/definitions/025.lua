-- AI 25: orcnamed3.
-- Selection: Mobs.AI = 25; never select this by MobID.
-- No current seed assignment; keep the original case available explicitly.
-- Stage-gated HP 80/45/25%:123;169 → >1 s →123; attack-speed2. Its Battle state callbacks are
-- **Empty**, unlike the other normal Battle callbacks
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 25,
    name = "orcnamed3",
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
            transitions = {
                { to = "UseMobSkill1", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.8))
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "UseMobSkill2", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.45))
                    local p2 = (v.stage == 1)
                    return p1 and p2
                end },
                { to = "BeStrong", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.25))
                    local p2 = (v.stage == 2)
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
                ctx:say_legacy(13)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- Say2: original named phase, preserving transition priority.
        ["Say2"] = {
            enter = function(ctx, v)
                ctx:say_legacy(9)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- UseMobSkill1: original named phase, preserving transition priority.
        ["UseMobSkill1"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(123)
                v.stage = num.i32(1)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- UseMobSkill2: original named phase, preserving transition priority.
        ["UseMobSkill2"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(169)
                v.phase_started_ms = num.u32(ctx:time_ms())
            end,
            transitions = {
                { to = "UseMobSkill3", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 1000)
                    return p1
                end },
            },
        },
        -- UseMobSkill3: original named phase, preserving transition priority.
        ["UseMobSkill3"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(123)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.stage = num.i32(2)
                    return true
                end },
            },
        },
        -- BeStrong: original named phase, preserving transition priority.
        ["BeStrong"] = {
            enter = function(ctx, v)
                ctx:set_attack_speed_multiplier(num.f32(2.0))
                v.stage = num.i32(3)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
    },
})
