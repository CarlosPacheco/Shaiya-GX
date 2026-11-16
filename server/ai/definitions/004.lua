-- AI 4: stationary.
-- Selection: Mobs.AI = 4; never select this by MobID.
-- No current seed assignment; keep the original case available explicitly.
-- Suppresses idle wandering and takes native stationary combat routing.
-- Original behavior is deliberate, including documented anomalies. See README.md.
local generic = require("core.generic")

return generic.define({
    id = 4, name = "stationary", family = "generic",
    acquire_proximity = false, stationary = true, help = "none",
})
