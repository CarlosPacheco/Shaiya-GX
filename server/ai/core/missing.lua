-- Missing source is an explicit original case, never an invented boss script.
local generic = require("core.generic")
local M = {}
function M.factory(code)
    local spec = generic.define({ id=code, name="unconfigured_factory", family="factory_missing",
        acquire_proximity=false, stationary=false, help="none", source_available=false })
    return spec
end
function M.lua(code)
    return require("core.scripted").define({
        id=code, name="missing_lua", family="lua_missing", source_available=false,
        load=function(ctx) ctx:disable_lua_callbacks(code) end,
        callbacks=function() return {} end,
    })
end
return M
