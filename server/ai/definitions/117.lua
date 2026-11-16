-- AI117: real Dios Exiel after splitting, mob2472; clones are template2473.
-- HP<=30% announces6404 once, then skill337 after3000ms and every3000ms
-- while still in that HP band. Normal reset clears timer/speech; death
-- requests removal of two clones near the caller through native semantics.
local scripted = require("core.scripted")
return scripted.define({
    id=117, name="dios_exiel_real", family="scripted",
    callbacks=function()
        local announced, attack_deadline = false, 0
        return {
            reset=function() announced, attack_deadline = false, 0 end,
            death=function(ctx) ctx:remove_mobs(2473, 2, 0.0, 0.0) end,
            combat=function(ctx, hp_percent)
                if hp_percent > 30 then return end
                if not announced then
                    ctx:announce(6404, 200.0)
                    announced = true
                    attack_deadline = ctx:time_ms() + 3000
                end
                if ctx:time_ms() >= attack_deadline then
                    ctx:use_mob_skill(337)
                    attack_deadline = ctx:time_ms() + 3000
                end
            end,
        }
    end,
})
