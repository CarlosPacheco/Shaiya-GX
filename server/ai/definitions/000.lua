-- AI 0: reactive.
-- Selection: Mobs.AI = 0; never select this by MobID.
-- Applies to 1344 templates in the audited seed; not a single boss.
-- Reactive wandering. Retaliates without calling nearby helpers.
-- Original behavior is deliberate, including documented anomalies. See README.md.
local generic = require("core.generic")

return generic.define({
    id = 0, name = "reactive", family = "generic",
    acquire_proximity = false, stationary = false, help = "none",
})
