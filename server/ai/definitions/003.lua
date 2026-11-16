-- AI 3: aggressive_aggressive_help.
-- Selection: Mobs.AI = 3; never select this by MobID.
-- Applies to 199 templates in the audited seed; not a single boss.
-- Acquires nearby targets and recruits idle AI1/3 helpers within ChaseRange.
-- Original behavior is deliberate, including documented anomalies. See README.md.
local generic = require("core.generic")

return generic.define({
    id = 3, name = "aggressive_aggressive_help", family = "generic",
    acquire_proximity = true, stationary = false, help = "aggressive",
})
