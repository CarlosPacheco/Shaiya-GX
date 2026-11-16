-- AI104: assigned to mob1890 in the current seed, but BOTH original EP4.5
-- package copies lack 104.lua. There is no verified custom boss behavior.
-- Preserve the missing-script case: disable custom Lua callbacks while the
-- host retains the native stack. Never silently replace this with AI0 or
-- manufacture encounter rules. This file makes the omission explicit.
return require("core.missing").lua(104)
