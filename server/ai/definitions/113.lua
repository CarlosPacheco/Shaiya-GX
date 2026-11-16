-- AI 113: Luhiel, mob 2467, first-floor Caelum Sacra boss.
-- Death notifies instance portal1001/country0. Native combat remains active.
local scripted = require("core.scripted")
return scripted.define({
    id=113, name="luhiel", family="scripted",
    callbacks=function()
        return { death=function(ctx) ctx:update_instance_portal(1001, 0) end }
    end,
})
