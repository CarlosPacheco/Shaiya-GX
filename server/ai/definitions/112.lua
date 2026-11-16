-- AI 112: Gargadel, mob 2466, first-floor Caelum Sacra boss.
-- Selection is Mobs.AI=112, irrespective of erroneous numbers in old comments.
-- Death notifies instance portal1001/country0. Native combat remains active.
local scripted = require("core.scripted")
return scripted.define({
    id=112, name="gargadel", family="scripted",
    callbacks=function()
        return { death=function(ctx) ctx:update_instance_portal(1001, 0) end }
    end,
})
