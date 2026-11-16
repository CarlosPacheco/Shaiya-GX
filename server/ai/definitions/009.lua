-- AI 9: unassigned_generic_9.
-- Selection: Mobs.AI = 9; never select this by MobID.
-- No current seed assignment; keep the original case available explicitly.
-- No dedicated native branch or current seed assignment; retain generic reactive behavior.
-- Original behavior is deliberate, including documented anomalies. See README.md.
local generic = require("core.generic")

return generic.define({
    id = 9, name = "unassigned_generic_9", family = "generic",
    acquire_proximity = false, stationary = false, help = "none",
})
