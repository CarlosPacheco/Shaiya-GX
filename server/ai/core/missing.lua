-- Missing source is an explicit original case, never an invented boss script.
local generic = require("core.generic")
local M = {}
function M.factory(code)
    local spec = generic.define({ id=code, name="unconfigured_factory", family="factory_missing",
        acquire_proximity=false, stationary=false, help="none", source_available=false })
    return spec
end
function M.lua(code)
    return { id=code, name="missing_lua", family="lua_missing", source_available=false,
        new=function(ctx)
            -- Original loader disables Lua callbacks after the file error but
            -- retains the common native stack; it does not change AI to zero.
            ctx:disable_lua_callbacks(code)
            return { event=function() end, tick=function() end }
        end }
end
return M
