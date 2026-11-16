-- Native Lua-family scheduling: idle/combat/move/return/death.
-- This Lua driver selects phases; the host supplies mechanics. Define custom
-- callback, with private per-mob closures instead of legacy global variables.
local lifecycle = require("core.lifecycle")
local M = {}
function M.define(spec)
    function spec.new(ctx)
        if spec.load then spec.load(ctx) end
        local callbacks = spec.callbacks()
        local self = {}
        function self:event(context, name, ...)
            if name == "idle" then
                lifecycle.idle(context)
                if context:advance_selected_move() then return end
                if context:status() == "return_home" then context:return_home(); return end
                if context:status() == "idle" then
                    if context:acquisition_ready() and context:find_target(true) then return end
                    context:idle_wander(false)
                end
            elseif name == "combat" then
                if context:status() ~= "combat" or not context:can_act() then return end
                if context:native_attack_ready() then context:battle(); return end
                if context:lua_callbacks_enabled() and callbacks.combat then return callbacks.combat(context, ...) end
            elseif context:lua_callbacks_enabled() then
                local fn = callbacks[name]
                if fn then return fn(context, ...) end
            end
        end
        -- Direct callback helper for policy-only traces; world drives event().
        function self:tick(context, hp_percent, attacked_count)
            if callbacks.combat then return callbacks.combat(context, hp_percent, attacked_count) end
        end
        self:event(ctx, "init")
        return self
    end
    return spec
end
return M
