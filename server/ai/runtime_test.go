package ai

import (
	"embed"
	rt "github.com/arnodel/golua/runtime"
	"io/fs"
	"runtime"
	"strings"
	"sync/atomic"
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

// Every call used to chain its termination to the continuation of the previous
// completed call (rt.Call1 starts from Thread.CurrentCont), so the runtime kept
// all of them alive: about 1 KB per AI event, on every tick, without end.
func TestCompletedCallsAreNotRetained(t *testing.T) {
	p, e := Load(nil)
	if e != nil {
		t.Fatal(e)
	}
	v, e := p.NewVM()
	if e != nil {
		t.Fatal(e)
	}
	defer v.Close()
	host := &testHost{}
	actor, e := v.NewActor(1, host)
	if e != nil {
		t.Fatal(e)
	}
	heap := func() int64 {
		runtime.GC()
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return int64(m.HeapAlloc)
	}
	for range 100 {
		if e := actor.Event(host, "reset"); e != nil {
			t.Fatal(e)
		}
	}
	before := heap()
	for range 50000 {
		if e := actor.Event(host, "reset"); e != nil {
			t.Fatal(e)
		}
	}
	if grew := heap() - before; grew > 2<<20 {
		t.Fatalf("the heap grew %d bytes over 50000 AI events: completed calls are retained", grew)
	}
}

// The Lua sources are read once, when the program loads; creating machines and
// running AI afterwards never opens a file again.
func TestLuaSourcesAreReadOnlyAtLoad(t *testing.T) {
	counting := &countingFS{FS: sources}
	p, e := Load(counting)
	if e != nil {
		t.Fatal(e)
	}
	opened := counting.opens.Load()
	if opened == 0 {
		t.Fatal("nothing was read")
	}
	host := &testHost{}
	for range 3 {
		v, e := p.NewVM()
		if e != nil {
			t.Fatal(e)
		}
		actor, e := v.NewActor(1, host)
		if e != nil {
			t.Fatal(e)
		}
		if e := actor.Event(host, "reset"); e != nil {
			t.Fatal(e)
		}
		v.Close()
	}
	if counting.opens.Load() != opened {
		t.Fatal("the sources were read again after loading")
	}
}

type countingFS struct {
	fs.FS
	opens atomic.Int64
}

func (c *countingFS) Open(name string) (fs.File, error) {
	c.opens.Add(1)
	return c.FS.Open(name)
}
