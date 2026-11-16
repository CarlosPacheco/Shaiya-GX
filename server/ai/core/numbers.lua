-- Numeric boundaries of the original 32-bit server, independent of Lua's
-- 64-bit integers/doubles. Keep these conversions at the data boundary.
local M = {}
function M.f32(value)
    return (string.unpack("<f", string.pack("<f", value)))
end
function M.u32(value)
    return math.tointeger(value) & 0xffffffff
end
function M.i32(value)
    value = M.u32(value)
    return value >= 0x80000000 and value - 0x100000000 or value
end
return M
