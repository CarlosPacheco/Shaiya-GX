-- AI 34: CrypticOne.
-- Selection: Mobs.AI = 34; never select this by MobID.
-- Mob 1700: Cryptic One.
-- Mob 1800: Cryptic One.
-- After>35 s choose eligible attacking user with highest current HP and skill251. After>240 s
-- open a >45 s hit-count phase; each positive count heals `count*50000` via AddEnergy, then
-- resets that count
-- Original behavior is deliberate, including documented anomalies. See README.md.

local machine = require("core.machine")
local num = require("core.numbers")

return machine.define({
    id = 34,
    name = "CrypticOne",
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
                    local p1 = (ctx:find_target() == true)
                    local p2 = (v.stage == 0)
                    return p1 and p2
                end },
                { to = "Say", check = function(ctx, v)
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
                { to = "SelectMaxHpUser", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.phase_started_ms)) > 35000)
                    return p1
                end },
                { to = "CheckHitPoint", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.secondary_clock_ms)) > 240000)
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
                ctx:announce_legacy(25, 200)
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
                ctx:use_mob_skill(251)
            end,
            transitions = {
                { to = "Battle", check = function(ctx, v)
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- SelectMaxHpUser: original named phase, preserving transition priority.
        ["SelectMaxHpUser"] = {
            enter = function(ctx, v)
                v.target_user_id = num.u32(ctx:highest_current_hp_attacker_id())
            end,
            transitions = {
                { to = "UseMobSkill1", check = function(ctx, v)
                    ctx:select_target_user(v.target_user_id)
                    return true
                end },
            },
        },
        -- HpChage: original named phase, preserving transition priority.
        ["HpChage"] = {
            transitions = {
                { to = "ResetCheckHitPoint", check = function(ctx, v)
                    local p1 = (num.u32((ctx:time_ms() - v.secondary_clock_ms)) > 45000)
                    return p1
                end },
                { to = "CheckAttackCount", check = function(ctx, v)
                    local p1 = (ctx:hit_count() > 0)
                    return p1
                end },
            },
        },
        -- CheckHitPoint: original named phase, preserving transition priority.
        ["CheckHitPoint"] = {
            enter = function(ctx, v)
                ctx:begin_hit_count()
                ctx:announce_legacy(26, 200)
            end,
            transitions = {
                { to = "HpChage", check = function(ctx, v)
                    v.secondary_clock_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- ResetCheckHitPoint: original named phase, preserving transition priority.
        ["ResetCheckHitPoint"] = {
            enter = function(ctx, v)
                ctx:end_hit_count()
            end,
            transitions = {
                { to = "Say1", check = function(ctx, v)
                    v.secondary_clock_ms = num.u32(ctx:time_ms())
                    v.phase_started_ms = num.u32(ctx:time_ms())
                    return true
                end },
            },
        },
        -- CheckAttackCount: original named phase, preserving transition priority.
        ["CheckAttackCount"] = {
            enter = function(ctx, v)
                ctx:heal(num.i32((ctx:hit_count() * 50000)))
            end,
            transitions = {
                { to = "HpChage", check = function(ctx, v)
                    ctx:reset_hit_count()
                    return true
                end },
            },
        },
        -- Say1: original named phase, preserving transition priority.
        ["Say1"] = {
            enter = function(ctx, v)
                ctx:announce_legacy(32, 200)
            end,
            transitions = {
                { to = "Battle" },
            },
        },
    },
})
