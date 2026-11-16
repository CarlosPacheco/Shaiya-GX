-- AI 7: unassigned_generic_7.
-- Selection: Mobs.AI = 7; never select this by MobID.
-- No current seed assignment; keep the original case available explicitly.
-- No dedicated native branch or current seed assignment; retain generic reactive behavior.
-- Original behavior is deliberate, including documented anomalies. See README.md.
local generic = require("core.generic")

return generic.define({
    id = 7, name = "unassigned_generic_7", family = "generic",
    acquire_proximity = false, stationary = false, help = "none",
})
