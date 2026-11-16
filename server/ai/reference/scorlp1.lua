-- Unmapped reference: scorlp1. No original mob_ai.ini assignment.
-- Kept outside the numeric registry; never activate by guessing an AI value.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = nil,
    name = "scorlp1",
    family = "states",
    initial = "init",
    variables = function()
        return { stage = 0, phase_started_ms = 0, target_user_id = 0 }
    end,
    states = {
        -- init: original named phase, preserving transition priority.
        ["init"] = {
            transitions = {
                { to = "idle", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- move: original named phase, preserving transition priority.
        ["move"] = {
            enter = function(ctx, v)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:random_walk()
            end,
            update = function(ctx, v)
                ctx:random_walk()
            end,
            transitions = {
                { to = "battle", check = function(ctx, v)
                    local p1 = (ctx:was_attacked() == true)
                    v.stage = num.i32(0)
                    return p1
                end },
                { to = "init", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 3000)
                    return p1
                end },
            },
        },
        -- idle: original named phase, preserving transition priority.
        ["idle"] = {
            transitions = {
                { to = "battle", check = function(ctx, v)
                    local p1 = (ctx:was_attacked() == true)
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "move", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 5000)
                    return p1
                end },
            },
        },
        -- battle: original named phase, preserving transition priority.
        ["battle"] = {
            enter = function(ctx, v)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:say_legacy(ctx:reference_text("scorlp1.text_1"))
                ctx:battle()
            end,
            update = function(ctx, v)
                ctx:battle()
            end,
            transitions = {
                { to = "runaway", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.1))
                    local p2 = (v.stage < 2)
                    return p1 and p2
                end },
                { to = "applyvalue", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.4))
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "schattackUser", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 10000)
                    return p1
                end },
                { to = "init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- schattackUser: original named phase, preserving transition priority.
        ["schattackUser"] = {
            enter = function(ctx, v)
                v.target_user_id = num.u32(ctx:highest_damage_attacker_id())
            end,
            transitions = {
                { to = "selectuser", check = function(ctx, v)
                    local prototype_check1 = (v.target_user_id ~= 0)
                    return not prototype_check1
                end },
                { to = "battle" },
            },
        },
        -- selectuser: original named phase, preserving transition priority.
        ["selectuser"] = {
            enter = function(ctx, v)
                ctx:select_target_user(v.target_user_id)
                ctx:say_legacy(ctx:reference_text("scorlp1.text_2"))
            end,
            transitions = {
                { to = "battle" },
            },
        },
        -- applyvalue: original named phase, preserving transition priority.
        ["applyvalue"] = {
            enter = function(ctx, v)
                v.stage = num.i32(1)
                ctx:set_damage_multiplier(num.f32(2.0))
                ctx:say_legacy(ctx:reference_text("scorlp1.text_3"))
                ctx:set_attack_speed_multiplier(num.f32(2.0))
            end,
            transitions = {
                { to = "battle" },
            },
        },
        -- runaway: original named phase, preserving transition priority.
        ["runaway"] = {
            enter = function(ctx, v)
                v.stage = num.i32(2)
                ctx:record_escape_origin()
                ctx:random_run()
            end,
            update = function(ctx, v)
                ctx:random_run()
            end,
            transitions = {
                { to = "battle", check = function(ctx, v)
                    local p1 = (ctx:distance_from_escape_origin() >= 20)
                    return p1
                end },
            },
        },
    },
})
