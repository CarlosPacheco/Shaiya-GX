-- AI 1: aggressive.
-- Selection: Mobs.AI = 1; never select this by MobID.
-- Applies to 1086 templates in the audited seed; not a single boss.
-- Acquires nearby targets. Does not select a social help routine.
-- Original behavior is deliberate, including documented anomalies. See README.md.
local generic = require("core.generic")

return generic.define({
    id = 1, name = "aggressive", family = "generic",
    acquire_proximity = true, stationary = false, help = "none",
})
