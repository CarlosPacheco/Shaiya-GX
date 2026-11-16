-- AI 111: Eternal, mob 2464, first-floor Caelum Sacra boss.
-- Selection is Mobs.AI=111. On death, notify instance portal 1001 for country0.
-- This is an instance-controller request, not a global portal-open command.
-- Empty original combat callbacks leave native attacks and movement active.
local scripted = require("core.scripted")
return scripted.define({
    id=111, name="eternal", family="scripted",
    callbacks=function()
        return { death=function(ctx) ctx:update_instance_portal(1001, 0) end }
    end,
})
