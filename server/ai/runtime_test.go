package ai

import (
	"embed"
	rt "github.com/arnodel/golua/runtime"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

//go:embed tests/*.lua
var testSource embed.FS

func TestIntegratedCatalog(t *testing.T) {
	p, e := Load(nil)
	if e != nil {
		t.Fatal(e)
	}
	v, e := p.NewVM()
	if e != nil {
		t.Fatal(e)
	}
	defer v.Close()
	for _, name := range []string{"tests.host", "tests.run"} {
		b, e := fs.ReadFile(testSource, strings.ReplaceAll(name, ".", "/")+".lua")
		if e != nil {
			t.Fatal(e)
		}
		p.source[name] = b
	}
	closure, e := v.r.CompileAndLoadLuaChunk("tests.run", p.source["tests.run"], rt.TableValue(v.r.GlobalEnv()))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = rt.Call1(v.r.MainThread(), rt.FunctionValue(closure)); e != nil {
		t.Fatal(e)
	}
}
func TestInvalidScriptsFailLoading(t *testing.T) {
	if _, e := Load(fstest.MapFS{"catalog.lua": {Data: []byte("this is not Lua")}}); e == nil {
		t.Fatal("accepted broken catalog")
	}
}

type testHost struct{ invulnerable bool }

func (h *testHost) Call(name string, args []any) (any, error) {
	if name == "lua_callbacks_enabled" {
		return true, nil
	}
	if name == "set_invulnerable" {
		h.invulnerable = args[0].(bool)
	}
	return nil, nil
}
func TestHostCallsAndBoundedExecution(t *testing.T) {
	p, e := Load(nil)
	if e != nil {
		t.Fatal(e)
	}
	v, e := p.NewVM()
	if e != nil {
		t.Fatal(e)
	}
	defer v.Close()
	h := &testHost{}
	if _, e = v.NewActor(115, h); e != nil {
		t.Fatal(e)
	}
	if !h.invulnerable {
		t.Fatal("init did not call real host")
	}
	closure, e := v.r.CompileAndLoadLuaChunk("loop", []byte("while true do end"), rt.TableValue(v.r.GlobalEnv()))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = v.call(rt.FunctionValue(closure)); e == nil {
		t.Fatal("unbounded script execution")
	}
}
