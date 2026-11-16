-- AI 16: sorp3.
-- Selection: Mobs.AI = 16; never select this by MobID.
-- Mob 1506: Piperr Clan Lautursi.
-- Mob 1509: Capsiwoo Clan Sidolk.
-- Mob 1512: Alimoss Clan Perpic.
-- Mob 1542: Erucan Clan Garemi.
-- Mob 1544: Apisus Clan Pylas.
-- At HP ≤20%, seek a non-battle mob with radius argument 12, move to selected position, then
-- recruit around target with radius 7 and advance stage
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 16,
    name = "sorp3",
    family = "states",
    initial = "Init",
    variables = function()
        return { stage = 0, phase_started_ms = 0, target_user_id = 0, random_roll = 0, secondary_clock_ms = 0 }
    end,
    states = {
        -- Idle: original named phase, preserving transition priority.
        ["Idle"] = {
            enter = function(ctx, v)
                v.phase_started_ms = num.u32(ctx:time_ms())
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    local p1 = (ctx:find_target() == true)
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "Battle", check = function(ctx, v)
                    local p1 = (ctx:was_attacked() == true)
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 10000)
                    return p1
                end },
            },
        },
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
                { to = "Runaway", check = function(ctx, v)
                    local p1 = (v.stage == 0)
                    local p2 = (ctx:hp_fraction() <= num.f32(0.2))
                    return p1 and p2
                end },
                { to = "Init", check = function(ctx, v)
                    local p1 = (ctx:is_in_combat() == false)
                    return p1
                end },
            },
        },
        -- Runaway: original named phase, preserving transition priority.
        ["Runaway"] = {
            enter = function(ctx, v)
                ctx:select_idle_ally_position(num.f32(12.0))
            end,
            transitions = {
                { to = "Arrival" },
            },
        },
        -- Help: original named phase, preserving transition priority.
        ["Help"] = {
            enter = function(ctx, v)
                v.target_user_id = num.u32(ctx:target_user_id())
                ctx:recruit_around_user(v.target_user_id, num.f32(7.0))
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.stage = num.i32(1)
                    return true
                end },
            },
        },
        -- Arrival: original named phase, preserving transition priority.
        ["Arrival"] = {
            enter = function(ctx, v)
                ctx:move_to_selected_position()
            end,
            update = function(ctx, v)
                ctx:move_to_selected_position()
            end,
            transitions = {
                { to = "Help", check = function(ctx, v)
                    local p1 = (ctx:arrived_at_selected_position() == true)
                    return p1
                end },
            },
        },
    },
})
