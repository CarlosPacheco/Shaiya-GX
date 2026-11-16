-- Logging-only host. Tests explicitly name every supported operation so a typo
-- cannot become a silently successful no-op world action.
local num = require("core.numbers")
local M = {}
function M.new()
    local h = { log={}, now=10000, hp=1, acquiring=false, attacked=false, combat=true,
        current_status="idle", random=0, pair=1, target=42, hits=0, arrival=false }
    local function record(name)
        return function(self, ...) self.log[#self.log+1] = {name,...} end
    end
    for _, name in ipairs({"set_damage_multiplier","set_physical_defense_multiplier",
        "set_attack_speed_multiplier","recruit_around_user","summon_legacy","select_attack_type",
        "set_battle_type","say_legacy","announce_legacy","select_escape_direction",
        "enable_all_attack_types","record_escape_origin","set_recovery_multiplier","enable_recovery",
        "set_recovery_interval_ms","set_defensive_element","select_target_user","begin_hit_count",
        "end_hit_count","reset_hit_count","heal","use_mob_skill","battle","random_walk","random_run",
        "move_to_selected_position","clear_target_flags","return_home","idle_wander","attack_ai",
        "recruit_idle","reset_generic","disable_lua_callbacks","update_instance_portal","announce",
        "create_mobs","remove_mobs","fixed_move","hold_position","set_invulnerable",
        "seed_lua_random_from_wall_clock","select_idle_ally_position"}) do h[name]=record(name) end
    function h:time_ms() return num.u32(self.now) end
    function h:hp_fraction() return num.f32(self.hp) end
    function h:was_attacked() return self.attacked end
    function h:is_in_combat() return self.combat end
    function h:find_target() self.log[#self.log+1]={"find_target"}; return self.acquiring end
    function h:legacy_end_battle_stub() self.log[#self.log+1]={"legacy_end_battle_stub"}; return nil end
    function h:random_0_99() return self.random end
    function h:target_user_id() return self.target end
    function h:consume_recent_hit() local result=self.attacked; self.attacked=false; return result end
    function h:arrived_at_selected_position() return self.arrival end
    function h:distance_from_escape_origin() return self.escape_distance or 0 end
    function h:highest_current_hp_attacker_id() return self.target end
    function h:highest_damage_attacker_id() return self.target end
    function h:hit_count() return self.hits end
    function h:reference_text(key) return key end
    function h:status() return self.current_status end
    function h:acquisition_ready() return self.ready ~= false end
    function h:lua_random(maximum) assert(self.pair>=1 and self.pair<=maximum); return self.pair end
    function h:count(name)
        local n=0; for _, e in ipairs(self.log) do if e[1]==name then n=n+1 end end; return n
    end
    function h:last(name)
        for i=#self.log,1,-1 do if self.log[i][1]==name then return self.log[i] end end
    end
    return h
end
return M
