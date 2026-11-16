-- Shared world lifecycle. Policies call primitives, never a second Go AI route.
local M = {}
function M.idle(ctx)
    ctx:refresh_target()
    if ctx:status() == "idle" or ctx:recovery_enabled() then ctx:recover_hp() end
end
return M
