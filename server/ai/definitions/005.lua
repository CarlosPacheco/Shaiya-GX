-- AI 5: reactive_all_idle_help.
-- Selection: Mobs.AI = 5; never select this by MobID.
-- Applies to 23 templates in the audited seed; not a single boss.
-- Reactive; initial acquisition recruits any eligible idle mob within ChaseRange.
-- Original behavior is deliberate, including documented anomalies. See README.md.
local generic = require("core.generic")

return generic.define({
    id = 5, name = "reactive_all_idle_help", family = "generic",
    acquire_proximity = false, stationary = false, help = "all_idle",
})
