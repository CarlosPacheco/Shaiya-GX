-- AI 2: reactive_same_template_help.
-- Selection: Mobs.AI = 2; never select this by MobID.
-- Mob 2369: Silverpick Goblin Spirit.
-- Mob 2370: Silverpick Goblin Spirit.
-- Reactive; initial acquisition recruits idle mobs with the same MobID within ChaseRange.
-- Original behavior is deliberate, including documented anomalies. See README.md.
local generic = require("core.generic")

return generic.define({
    id = 2, name = "reactive_same_template_help", family = "generic",
    acquire_proximity = false, stationary = false, help = "same_template",
})
