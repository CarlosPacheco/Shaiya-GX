-- AI118: initial Dios Exiel phase, mob2469.
-- Death creates one next-phase mob2470 at the caller's location. The host
-- validates the spawn and supplies native zero-coordinate semantics.
-- Original callbacks add no other rules; native combat remains active.
local scripted = require("core.scripted")
return scripted.define({
    id=118, name="dios_exiel_initial", family="scripted",
    callbacks=function()
        return { death=function(ctx) ctx:create_mobs(2470, 1, 0.0, 0.0) end }
    end,
})
