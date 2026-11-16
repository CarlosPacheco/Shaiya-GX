-- Native Lua-family scheduling is still supplied by the host's common stack
-- machine: idle/combat/move/return/death. These Lua files define every custom
-- callback, with private per-mob closures instead of legacy global variables.
local M = {}
function M.define(spec)
    function spec.new(ctx)
        if spec.load then spec.load(ctx) end
        local callbacks = spec.callbacks()
        local self = {}
        function self:event(context, name, ...)
            local fn = callbacks[name]
            if fn then return fn(context, ...) end
        end
        function self:tick(context, hp_percent, attacked_count)
            return self:event(context, "combat", hp_percent, attacked_count)
        end
        self:event(ctx, "init")
        return self
    end
    return spec
end
return M
