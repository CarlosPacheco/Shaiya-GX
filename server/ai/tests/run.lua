assert(_VERSION == "Lua 5.4" or _VERSION == "Lua 5.5" or _VERSION == "Golua 5.5", "use modern Lua 5.4 or 5.5")
local catalog = require("catalog")
local host = require("tests.host")
local num = require("core.numbers")
local passed = 0
local function test(name, fn)
    local ok, err = pcall(fn)
    assert(ok, name .. ": " .. tostring(err))
    passed = passed + 1
end
local function enter_battle(code)
    local ctx=host.new(); ctx.acquiring=true
    local ai=catalog.new(code,ctx)
    for _=1,12 do
        if ai.state=="Battle" or ai.state=="battle" then return ai,ctx end
        ai:tick(ctx)
    end
    error("could not reach battle for "..code)
end

test("all256 numeric routes, independent instances and available structure",function()
    local states, transitions, definitions = 0,0,0
    for id=0,255 do
        local spec=catalog.resolve(id); assert(spec.id==id)
        local ctx=host.new(); local a=spec.new(ctx); local b=spec.new(ctx); assert(a~=b)
        if spec.family=="states" then
            definitions=definitions+1
            for _,s in pairs(spec.states) do states=states+1;transitions=transitions+#(s.transitions or {}) end
            a.vars.stage=99; assert(b.vars.stage==0)
        end
    end
    assert(definitions==27 and states==235 and transitions==353)
    for _,bad in ipairs({-1,256,1.5,"1"}) do assert(not pcall(catalog.resolve,bad)) end
end)

test("one event contract across all policy mechanisms",function()
    for _,id in ipairs({0,1,2,3,4,5,9,10,28,37,104,111,114,115,116,117,118,255}) do
        local ctx=host.new();local ai=catalog.new(id,ctx)
        ai:event(ctx,"idle");ai:event(ctx,"combat",100,0)
        ai:event(ctx,"target",42,false);ai:event(ctx,"attacked",42)
        ai:event(ctx,"move_end");ai:event(ctx,"return_home",0)
        ai:event(ctx,"death",0);ai:event(ctx,"reset")
    end
end)

test("generic social rules and stationary selection",function()
    local expected={[0]="none",[1]="none",[2]="same_template",[3]="aggressive",[4]="none",[5]="all_idle"}
    for id=0,9 do
        local ctx=host.new(); local ai=catalog.new(id,ctx)
        ai:event(ctx,"target",42,true)
        assert(ctx:count("recruit_idle")==((expected[id] or "none")~="none" and 1 or 0))
        if ctx:last("recruit_idle") then assert(ctx:last("recruit_idle")[3]==expected[id]) end
        ai:event(ctx,"target",42,false); assert(ctx:count("recruit_idle")<=1)
        ai:event(ctx,"idle")
        assert(ctx:count("find_target")==((id==1 or id==3) and 1 or 0))
        assert(ctx:last("idle_wander")[2]==(id==4))
    end
end)

test("float32 threshold and one-shot scorpion speed",function()
    local ai,ctx=enter_battle(10); ctx.hp=.4
    ai:tick(ctx); assert(ai.state=="BeStrong" and ctx:last("set_attack_speed_multiplier")[2]==2)
    ai:tick(ctx); ai:tick(ctx); assert(ctx:count("set_attack_speed_multiplier")==1)
end)

test("failed predicates do not short-circuit native target queries",function()
    local ctx=host.new(); local ai=catalog.new(10,ctx); ai:tick(ctx); ai.vars.stage=1
    ai:tick(ctx); assert(ctx:count("find_target")==1 and ai.state=="Idle")
end)

test("transition assignments run before a deadline succeeds",function()
    local spec=catalog.resolve(20); local ctx=host.new(); local ai=spec.new(ctx)
    ai.state="UseMobSkill1"; ai.vars.phase_started_ms=ctx.now
    ai:tick(ctx); assert(ai.state=="UseMobSkill1" and ai.vars.stage==1)
    ctx.now=ctx.now+1000; ai:tick(ctx); assert(ai.state=="UseMobSkill1")
    ctx.now=ctx.now+1; ai:tick(ctx); assert(ai.state=="Battle")
end)

test("original shadowed phases are not repaired",function()
    for _,case in ipairs({{22,.1,3,"UseMobSkill3"},{24,.1,2,"UseMobSkill1"},{36,.5,0,"UseMobSkill1"}}) do
        local ctx=host.new();local ai=catalog.new(case[1],ctx)
        ai.state="Battle";ai.vars.stage=case[3];ctx.hp=case[2]
        ai:tick(ctx);assert(ai.state==case[4])
    end
end)

test("orc25 Empty Battle callbacks remain empty",function()
    local spec=catalog.resolve(25); assert(not spec.states.Battle.enter and not spec.states.Battle.update)
end)

test("Deinos clocks, strict boundaries, strength duration and reset",function()
    local ai,ctx=enter_battle(28); local start=ctx.now
    ctx.now=start+90000;ai:tick(ctx);assert(ai.state=="Battle")
    ctx.now=ctx.now+1;ai:tick(ctx);assert(ctx:last("use_mob_skill")[2]==253)
    ai:tick(ctx);ctx.now=start+150001;ai:tick(ctx);assert(ai.state=="BeStrong")
    ai:tick(ctx); assert(ai.state=="Battle1")
    ctx.now=ctx.now+30000;ai:tick(ctx);assert(ai.state=="Battle1")
    ctx.now=ctx.now+1;ai:tick(ctx);assert(ctx:last("set_damage_multiplier")[2]==1)
    ai:reset(ctx);assert(ai.vars.stage==0 and ai.state=="Init")
end)

test("unsigned native clocks and float32 representation",function()
    assert(num.u32(0xffffffff+1)==0 and num.u32(-1)==0xffffffff)
    assert(num.i32(0xffffffff)==-1 and num.f32(.4)~=.4)
end)

for _,id in ipairs({111,112,113}) do
    test("portal boss"..id,function()
        local ctx=host.new(); local ai=catalog.new(id,ctx); ai:event(ctx,"death")
        local e=ctx:last("update_instance_portal"); assert(e[2]==1001 and e[3]==0)
    end)
end

test("Huistaton bands, skipped bands, isolation and reset",function()
    local ctx=host.new(); local ai=catalog.new(114,ctx)
    for _,hp in ipairs({70,50,30,10}) do ai:tick(ctx,hp) end
    local skills={};for _,e in ipairs(ctx.log) do if e[1]=="use_mob_skill" then skills[#skills+1]=e[2] end end
    assert(table.concat(skills,",")=="334,335,334,335")
    ai:event(ctx,"reset");ctx.log={};for _=1,5 do ai:tick(ctx,10) end
    assert(ctx:count("use_mob_skill")==4);for _,e in ipairs(ctx.log) do assert(e[2]==335) end
end)

for pair,ids in ipairs({{2464,2465},{2464,2466},{2464,2467},{2465,2466},{2465,2467},{2466,2467}}) do
    test("Dios controller pair"..pair,function()
        local ctx=host.new();ctx.pair=pair;local ai=catalog.new(115,ctx)
        assert(ctx:last("set_invulnerable")[2]==true)
        ai:event(ctx,"target");ai:event(ctx,"idle")
        assert(ctx:count("announce")==0 and ctx:count("create_mobs")==0)
        ai:event(ctx,"attacked")
        assert(ctx:count("announce")==1 and ctx:last("announce")[2]==6401)
        ctx.now=14999;ai:event(ctx,"idle")
        assert(ctx:count("fixed_move")==0 and ctx:count("create_mobs")==0)
        ctx.now=15000;ai:event(ctx,"idle")
        assert(ctx:count("announce")==2 and ctx:last("announce")[2]==6402)
        local mobs={};for _,e in ipairs(ctx.log) do if e[1]=="create_mobs" then mobs[#mobs+1]=e[2];assert(e[4]==220 and e[5]==180) end end
        assert(mobs[1]==ids[1] and mobs[2]==ids[2] and ctx:count("fixed_move")==1)
        ai:event(ctx,"attacked");ai:event(ctx,"idle");assert(ctx:count("create_mobs")==2)
        ai:event(ctx,"move_end");assert(ctx:last("remove_mobs")[2]==2471)
        ai:event(ctx,"move_end");assert(ctx:count("remove_mobs")==1)
    end)
end

test("Dios controller leaves if withdrawal stalls, even across clock wrap",function()
    local ctx=host.new();local ai=catalog.new(115,ctx)
    ctx.now=0xffffffff-1000;ai:event(ctx,"attacked")
    ctx.now=3998;ai:event(ctx,"idle");assert(ctx:count("fixed_move")==0)
    ctx.now=3999;ai:event(ctx,"idle");assert(ctx:count("fixed_move")==1)
    ctx.now=33998;ai:event(ctx,"idle");assert(ctx:count("remove_mobs")==0)
    ctx.now=33999;ai:event(ctx,"idle");assert(ctx:count("remove_mobs")==1)
end)

test("Dios split delay and deliberately empty reset",function()
    local ctx=host.new();local ai=catalog.new(116,ctx)
    ai:tick(ctx,51);assert(#ctx.log==0);ai:tick(ctx,50)
    ctx.now=10999;ai:tick(ctx,50);assert(ctx:count("create_mobs")==0)
    ctx.now=11000;ai:tick(ctx,50);assert(ctx:count("create_mobs")==2)
    assert(ctx:last("create_mobs")[2]==2473 and ctx:last("create_mobs")[3]==2)
    ai:event(ctx,"reset");ai:tick(ctx,40);assert(ctx:count("create_mobs")==2 and ctx:count("remove_mobs")==2)
end)

test("Dios real boss skill interval, clone cleanup and reset",function()
    local ctx=host.new();local ai=catalog.new(117,ctx)
    ai:tick(ctx,30);ctx.now=12999;ai:tick(ctx,30);assert(ctx:count("use_mob_skill")==0)
    ctx.now=13000;ai:tick(ctx,30);ctx.now=16000;ai:tick(ctx,30)
    assert(ctx:count("use_mob_skill")==2 and ctx:last("use_mob_skill")[2]==337)
    ai:event(ctx,"death");assert(ctx:last("remove_mobs")[2]==2473)
    ai:event(ctx,"reset");ai:tick(ctx,30);assert(ctx:count("announce")==2)
end)

test("Dios initial death and missing source cases",function()
    local ctx=host.new();local ai=catalog.new(118,ctx);ai:event(ctx,"death")
    assert(ctx:last("create_mobs")[2]==2470)
    catalog.new(104,ctx);assert(ctx:last("disable_lua_callbacks")[2]==104)
    assert(catalog.resolve(37).family=="factory_missing" and catalog.resolve(255).family=="lua_missing")
end)

-- Smoke every configured machine with changing input, catching misspelled API
-- names and state-local variables without requiring original proprietary files.
for id=10,36 do
    test("changing-input smoke"..id,function()
        local ctx=host.new();local ai=catalog.new(id,ctx);ctx.acquiring=true;ctx.attacked=true
        for i=1,120 do
            ctx.now=10000+i*10001;ctx.hp=(i%101)/100;ctx.hits=i%7;ctx.random=i%100
            ctx.arrival=true;ctx.escape_distance=30;ai:tick(ctx)
        end
    end)
end
print(("AI catalog: %d scenarios passed on %s; no world server was started."):format(passed,_VERSION))
