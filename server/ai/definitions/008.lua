-- AI 8: unassigned_generic_8.
-- Selection: Mobs.AI = 8; never select this by MobID.
-- No current seed assignment; keep the original case available explicitly.
-- No dedicated native branch or current seed assignment; retain generic reactive behavior.
-- Original behavior is deliberate, including documented anomalies. See README.md.
local generic = require("core.generic")

return generic.define({
    id = 8, name = "unassigned_generic_8", family = "generic",
    acquire_proximity = false, stationary = false, help = "none",
})
