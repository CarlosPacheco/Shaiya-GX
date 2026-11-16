-- AI116: Dios Exiel phase before splitting, mob2470.
-- At integer HP<=50: announce6403, wait1000ms, create one real boss2472 and
-- two clones2473, then request removal of2470. Zero coordinates mean the
-- caller's location through the native API, never world origin.
-- Original reset callback is empty. Keep phase flags across a normal reset,
-- and keep repeated deletion requests if callbacks continue after splitting.
local scripted = require("core.scripted")
return scripted.define({
    id=116, name="dios_exiel_split", family="scripted",
    callbacks=function()
        local announced, created, create_deadline = false, false, 0
        return { combat=function(ctx, hp_percent)
            if hp_percent > 50 then return end
            if not announced then
                create_deadline = ctx:time_ms() + 1000
                ctx:announce(6403, 200.0)
                announced = true
            end
            if ctx:time_ms() >= create_deadline and not created then
                ctx:create_mobs(2472, 1, 0.0, 0.0)
                ctx:create_mobs(2473, 2, 0.0, 0.0)
                created = true
            end
            if created then ctx:remove_mobs(2470, 1, 0.0, 0.0) end
        end }
    end,
})
