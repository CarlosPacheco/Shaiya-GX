-- One Lua registry for every byte accepted by Mobs.AI. The numeric DB values
-- remain unchanged. Unassigned ranges retain explicit original fallback cases.
local M = {}
function M.resolve(code)
    assert(math.type(code) == "integer" and code >= 0 and code <= 255, "AI must be a byte")
    if code <= 36 or code == 104 or (code >= 111 and code <= 118) then
        return require(("definitions.%03d"):format(code))
    end
    if code < 100 then return require("core.missing").factory(code) end
    return require("core.missing").lua(code)
end
function M.new(code, ctx) return M.resolve(code).new(ctx) end
return M
