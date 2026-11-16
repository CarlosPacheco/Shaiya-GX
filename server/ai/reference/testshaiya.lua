-- Unmapped reference: testshaiya. No original mob_ai.ini assignment.
-- Kept outside the numeric registry; never activate by guessing an AI value.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = nil,
    name = "testshaiya",
    family = "states",
    initial = "Init",
    variables = function()
        return { stage = 0, phase_started_ms = 0 }
    end,
    states = {
        -- Init: original named phase, preserving transition priority.
        ["Init"] = {
            transitions = {
                { to = "Move", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- Move: original named phase, preserving transition priority.
        ["Move"] = {
            enter = function(ctx, v)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:random_walk()
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    local p1 = (ctx:was_attacked() == true)
                    v.stage = num.i32(0)
                    return p1
                end },
                { to = "ArrivePosition", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 3000)
                    return p1
                end },
            },
        },
        -- ArrivePosition: original named phase, preserving transition priority.
        ["ArrivePosition"] = {
            transitions = {
                { to = "Init" },
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
                { to = "Say", check = function(ctx, v)
                    local p1 = (v.stage == 0)
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
                ctx:say_legacy(ctx:reference_text("testshaiya.text_1"))
            end,
            transitions = {
                { to = "MoveAnywhere", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- MoveAnywhere: original named phase, preserving transition priority.
        ["MoveAnywhere"] = {
            enter = function(ctx, v)
                ctx:random_walk()
            end,
            update = function(ctx, v)
                ctx:say_legacy(ctx:reference_text("testshaiya.text_2"))
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.stage = num.i32(1)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 3000)
                    return p1
                end },
            },
        },
    },
})
