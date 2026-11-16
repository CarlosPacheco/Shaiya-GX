-- Small Lua state machine, not an interpreter for .ai files or token bytecode.
-- Each definition contains ordinary Lua functions and named state variables.
local lifecycle = require("core.lifecycle")
local M = {}

function M.define(spec)
    assert(spec.states[spec.initial], "missing initial state")
    for _, state in pairs(spec.states) do
        for _, transition in ipairs(state.transitions or {}) do
            assert(spec.states[transition.to], "missing transition destination")
        end
    end
    function spec.new(ctx)
        local self = { state = spec.initial, vars = {} }
        function self:reset(context)
            self.vars = spec.variables()
            self.state = spec.initial
            local enter = spec.states[self.state].enter
            if enter then enter(context, self.vars) end
        end
        function self:tick(context)
            local current = spec.states[self.state]
            -- Native order: scan transitions once, select the first eligible
            -- destination, leave/enter, then update the resulting state.
            for _, transition in ipairs(current.transitions or {}) do
                local eligible = not transition.check or transition.check(context, self.vars)
                if eligible and transition.to ~= self.state then
                    if current.leave then current.leave(context, self.vars) end
                    self.state = transition.to
                    current = spec.states[self.state]
                    if current.enter then current.enter(context, self.vars) end
                    break
                end
            end
            if current.update then current.update(context, self.vars) end
        end
        function self:event(context, name)
            if name == "reset" then self:reset(context)
            elseif name == "idle" then
                lifecycle.idle(context)
                if context:status() == "return_home" then context:return_home() else self:tick(context) end
            elseif name ~= "combat" and name ~= "attacked" and name ~= "target" and name ~= "death" and name ~= "move_end" and name ~= "return_home" then
                error("unknown state-machine event: " .. tostring(name))
            end
        end
        self:reset(ctx)
        return self
    end
    return spec
end
return M
