-- AI115: Dios Exiel's invulnerable encounter controller, mob2471.
-- Selects ONE of six pairs of first-floor bosses2464..2467 at(220,180),
-- requests movement to(173,179), then holds/removes itself on arrival.
-- Preserve the original defect: speech6402 tests the saved deadline >=5000,
-- not current time >= deadline. It can follow speech6401 immediately.
-- Preserve early return after the summon: a movement request is possible only
-- during the first combat callback. Reset intentionally does not clear flags.
-- RNG remains a host operation; do not introduce os/time/file access in scripts.
local scripted = require("core.scripted")
local pairs = { {2464,2465}, {2464,2466}, {2464,2467}, {2465,2466}, {2465,2467}, {2466,2467} }
return scripted.define({
    id=115, name="dios_exiel_controller", family="scripted",
    load=function(ctx) ctx:seed_lua_random_from_wall_clock() end,
    callbacks=function()
        local speech_count, summoned, move_pending, speech_deadline = 0, false, false, 0
        local function begin(ctx)
            if speech_count == 0 then
                ctx:announce(6401, 500.0)
                speech_count = speech_count + 1
                speech_deadline = ctx:time_ms() + 5000
            end
            move_pending = true
        end
        return {
            init=function(ctx) ctx:set_invulnerable(true) end,
            attacked=begin, target=begin,
            move_end=function(ctx)
                ctx:hold_position(true)
                ctx:remove_mobs(2471, 1, 0.0, 0.0)
            end,
            combat=function(ctx)
                if speech_deadline >= 5000 and speech_count <= 1 then
                    ctx:announce(6402, 500.0)
                    speech_count = speech_count + 1
                end
                if summoned then return end
                local selected = pairs[ctx:lua_random(6)]
                ctx:create_mobs(selected[1], 1, 220.0, 180.0)
                ctx:create_mobs(selected[2], 1, 220.0, 180.0)
                summoned = true
                if move_pending then
                    ctx:fixed_move(1, 173, 179)
                    move_pending = false
                end
            end,
        }
    end,
})
