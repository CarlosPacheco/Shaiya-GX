-- AI115: Dios Exiel's invulnerable first-floor encounter controller, mob2471.
local scripted = require("core.scripted")
local num = require("core.numbers")
local guards = { {2464,2465}, {2464,2466}, {2464,2467}, {2465,2466}, {2465,2467}, {2466,2467} }

return scripted.define({
    id=115, name="dios_exiel_controller", family="scripted",
    load=function(ctx) ctx:seed_lua_random_from_wall_clock() end,
    callbacks=function()
        local hit_at, retreat_at, removed
        local function leave(ctx)
            if not retreat_at or removed then return end
            ctx:hold_position(true)
            ctx:remove_mobs(2471, 1, 0.0, 0.0)
            removed = true
        end
        return {
            init=function(ctx) ctx:set_invulnerable(true) end,
            attacked=function(ctx)
                if hit_at then return end
                hit_at = ctx:time_ms()
                ctx:announce(6401, 500.0)
            end,
            idle=function(ctx)
                if not hit_at or removed then return end
                local now = ctx:time_ms()
                if not retreat_at then
                    if num.i32(now - hit_at) < 5000 then return end
                    ctx:announce(6402, 500.0)
                    local selected = guards[ctx:lua_random(#guards)]
                    ctx:create_mobs(selected[1], 1, 220.0, 180.0)
                    ctx:create_mobs(selected[2], 1, 220.0, 180.0)
                    ctx:fixed_move(1, 173, 179)
                    retreat_at = now
                elseif num.i32(now - retreat_at) >= 30000 then
                    -- A blocked retreat must still remove the controller.
                    leave(ctx)
                end
            end,
            move_end=leave,
        }
    end,
})
