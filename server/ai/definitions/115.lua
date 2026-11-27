-- AI115: Dios Exiel's invulnerable first-floor encounter controller, mob2471.
-- The false Exiel never moves and cannot be hurt. The first hit announces
-- 6401; five seconds later it announces 6402, disappears and two of the
-- four guards appear.
local scripted = require("core.scripted")
local num = require("core.numbers")
local guards = { {2464,2465}, {2464,2466}, {2464,2467}, {2465,2466}, {2465,2467}, {2466,2467} }

return scripted.define({
    id=115, name="dios_exiel_controller", family="scripted",
    load=function(ctx) ctx:seed_lua_random_from_wall_clock() end,
    callbacks=function()
        local hit_at, removed
        local function vanish(ctx)
            if not hit_at or removed then return end
            if num.i32(ctx:time_ms() - hit_at) < 5000 then return end
            removed = true
            ctx:announce(6402, 500.0)
            local selected = guards[ctx:lua_random(#guards)]
            ctx:create_mobs(selected[1], 1, 220.0, 180.0)
            ctx:create_mobs(selected[2], 1, 220.0, 180.0)
            ctx:remove_mobs(2471, 1, 0.0, 0.0)
        end
        return {
            init=function(ctx)
                ctx:set_invulnerable(true)
                ctx:hold_position(true)
            end,
            attacked=function(ctx)
                if hit_at then return end
                hit_at = ctx:time_ms()
                ctx:hold_position(true)
                ctx:announce(6401, 500.0)
            end,
            idle=vanish,
            combat=vanish,
        }
    end,
})
