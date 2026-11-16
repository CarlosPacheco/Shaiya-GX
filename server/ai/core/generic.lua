-- Shared native-code family. The Lua definition selects acquisition, stationary
-- behavior and social recruitment; the host supplies movement/combat mechanics.
local lifecycle = require("core.lifecycle")
local M = {}
function M.define(spec)
    function spec.new()
        local self = {}
        function self:event(ctx, event, target, request_help)
            if event == "idle" then
                lifecycle.idle(ctx)
                -- The host runs this phase at native idle-update cadence.
                if ctx:status() == "return_home" then ctx:return_home(); return end
                if ctx:status() ~= "idle" then return end
                if spec.acquire_proximity and ctx:acquisition_ready() and ctx:find_target() then return end
                -- Native idle wandering also checks the obelisk flag, normal
                -- step/timer, movement impairments and home radius.
                ctx:idle_wander(spec.stationary)
            elseif event == "combat" then
                if ctx:status() == "combat" then ctx:attack_ai(spec.stationary) end
            elseif event == "target" then
                -- Native SetTarget(..., true) requests social help once.
                -- Recruited helpers use false and cannot recursively recruit.
                if request_help and spec.help ~= "none" then
                    ctx:recruit_idle(target, spec.help)
                end
            elseif event == "reset" then
                ctx:reset_generic()
            elseif event ~= "death" and event ~= "attacked" and event ~= "move_end" and event ~= "return_home" then
                error("unknown generic AI event: " .. tostring(event))
            end
        end
        function self:tick(ctx) self:event(ctx, "idle"); self:event(ctx, "combat") end
        return self
    end
    return spec
end
return M
