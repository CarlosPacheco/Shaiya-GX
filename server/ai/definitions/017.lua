-- AI 17: sorpnamed1.
-- Selection: Mobs.AI = 17; never select this by MobID.
-- Mob 1518: High Priest Canis Femin.
-- Mob 1521: Shultan Erpolrin Capsiwoo.
-- Mob 1522: High Priest Iona Cragion.
-- Mob 1559: Caliph Atemis Erucan.
-- Mob 1561: Caliph Lacryma Apisus.
-- Mob 1563: Prophet Kipella Turusicer.
-- HP ≤50%: attack mask 5, masked Battle for >5 s, then restore. HP ≤25% before final stage:
-- skill 42, directional movement argument 15 after >1 s, await arrival, skill 63, then >1 s and
-- final stage
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 17,
    name = "sorpnamed1",
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
                { to = "OverCoat", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.5))
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "HitandRun", check = function(ctx, v)
                    local p1 = (ctx:hp_fraction() <= num.f32(0.25))
                    local p2 = (v.stage < 2)
                    return p1 and p2
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- HitandRun: original named phase, preserving transition priority.
        ["HitandRun"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(42)
                v.phase_started_ms = num.u32(ctx:time_ms())
                ctx:select_escape_direction(num.f32(15.0))
            end,
            transitions = {
                { to = "Runaway", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 1000)
                    return p1
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- MagicFire: original named phase, preserving transition priority.
        ["MagicFire"] = {
            enter = function(ctx, v)
                ctx:use_mob_skill(63)
                v.target_user_id = num.u32(ctx:target_user_id())
                v.phase_started_ms = num.u32(ctx:time_ms())
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 1000)
                    v.stage = num.i32(3)
                    return p1
                end },
            },
        },
        -- Say1: original named phase, preserving transition priority.
        ["Say1"] = {
            enter = function(ctx, v)
                ctx:say_legacy(1)
            end,
            transitions = {
                { to = "Battle" },
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
        -- Say2: original named phase, preserving transition priority.
        ["Say2"] = {
            enter = function(ctx, v)
                ctx:say_legacy(2)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
        -- OverCoat: original named phase, preserving transition priority.
        ["OverCoat"] = {
            enter = function(ctx, v)
                ctx:select_attack_type(5)
            end,
            transitions = {
                { to = "Battle1" },
            },
        },
        -- ReTurn: original named phase, preserving transition priority.
        ["ReTurn"] = {
            enter = function(ctx, v)
                ctx:enable_all_attack_types()
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.stage = num.i32(1)
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
                { to = "ReTurn", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 5000)
                    return p1
                end },
            },
        },
        -- Runaway: original named phase, preserving transition priority.
        ["Runaway"] = {
            enter = function(ctx, v)
                ctx:move_to_selected_position()
            end,
            update = function(ctx, v)
                ctx:move_to_selected_position()
            end,
            transitions = {
                { to = "MagicFire", check = function(ctx, v)
                    local p1 = (ctx:arrived_at_selected_position() == true)
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
