-- AI 114: Huistaton, mob2465, first-floor Caelum Sacra boss.
-- HP is the native integer percentage, not the 0..1 fraction used by old FSMs.
-- Bands (60,80],(40,60],(20,40],[0,20] request skills334,335,334,335.
-- Preserve the single incrementing stage: skipping bands can repeat the skill
-- of the current band until its stage ceiling is reached. Do not catch up by
-- playing each skipped band's skill. Reset clears this stage; death signals
-- instance portal1001/country0. Other native combat remains active.
local scripted = require("core.scripted")
return scripted.define({
    id=114, name="huistaton", family="scripted",
    callbacks=function()
        local stage = 0
        return {
            reset=function() stage = 0 end,
            death=function(ctx) ctx:update_instance_portal(1001, 0) end,
            combat=function(ctx, hp_percent)
                local ceiling, skill
                if hp_percent <= 80 and hp_percent > 60 then ceiling, skill = 1, 334
                elseif hp_percent <= 60 and hp_percent > 40 then ceiling, skill = 2, 335
                elseif hp_percent <= 40 and hp_percent > 20 then ceiling, skill = 3, 334
                elseif hp_percent <= 20 and hp_percent >= 0 then ceiling, skill = 4, 335
                else return end
                if stage >= ceiling then return end
                ctx:use_mob_skill(skill)
                stage = stage + 1
            end,
        }
    end,
})
