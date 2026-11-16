// Package ai executes the authoritative mob policies in modern, pure-Go Lua.
package ai

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"

	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/lib/base"
	"github.com/arnodel/golua/lib/mathlib"
	"github.com/arnodel/golua/lib/stringlib"
	"github.com/arnodel/golua/lib/tablelib"
	rt "github.com/arnodel/golua/runtime"
)

//go:embed catalog.lua core/*.lua definitions/*.lua
var sources embed.FS

// Host supplies world primitives. Lua owns decisions; unavailable primitives
// must return an error rather than silently bypass an encounter rule.
type Host interface {
	Call(string, []any) (any, error)
}

type Program struct {
	acquire [256]bool
	mu      sync.Mutex
	source  map[string][]byte
	units   map[string]*code.Unit
}

func Load(source fs.FS) (*Program, error) {
	if source == nil {
		source = sources
	}
	p := &Program{source: map[string][]byte{}, units: map[string]*code.Unit{}}
	err := fs.WalkDir(source, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != "." && path != "core" && path != "definitions" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".lua") {
			b, e := fs.ReadFile(source, path)
			if e != nil {
				return e
			}
			p.source[strings.TrimSuffix(strings.ReplaceAll(path, "/", "."), ".lua")] = b
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	v, err := p.NewVM()
	if err != nil {
		return nil, err
	}
	defer v.Close()
	for name := range p.source {
		if _, err = v.module(name); err != nil {
			return nil, err
		}
	}
	resolve := v.catalog.AsTable().Get(rt.StringValue("resolve"))
	for id := range 256 {
		result, e := v.call(resolve, rt.IntValue(int64(id)))
		if e != nil {
			return nil, fmt.Errorf("AI %d: %w", id, e)
		}
		t, ok := result.TryTable()
		if !ok || t.Get(rt.StringValue("id")) != rt.IntValue(int64(id)) {
			return nil, fmt.Errorf("AI %d has invalid definition", id)
		}
		p.acquire[id], _ = t.Get(rt.StringValue("acquire_proximity")).TryBool()
	}
	return p, nil
}

var defaultOnce sync.Once
var defaultProgram *Program
var defaultError error

func Default() (*Program, error) {
	defaultOnce.Do(func() { defaultProgram, defaultError = Load(nil) })
	return defaultProgram, defaultError
}

type VM struct {
	mu      sync.Mutex
	program *Program
	r       *rt.Runtime
	modules map[string]rt.Value
	catalog rt.Value
	host    Host
	context rt.Value
}

func (p *Program) NewVM() (*VM, error) {
	v := &VM{program: p, r: rt.New(io.Discard), modules: map[string]rt.Value{}}
	base.Load(v.r)
	for name, load := range map[string]func(*rt.Runtime) (rt.Value, func()){"math": mathlib.LibLoader.Load, "string": stringlib.LibLoader.Load, "table": tablelib.LibLoader.Load} {
		value, _ := load(v.r)
		v.r.SetEnv(v.r.GlobalEnv(), name, value)
	}
	for _, name := range []string{"dofile", "loadfile", "load", "collectgarbage"} {
		v.r.SetEnv(v.r.GlobalEnv(), name, rt.NilValue)
	}
	requireFn := v.r.SetEnvGoFunc(v.r.GlobalEnv(), "require", func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
		name, e := c.StringArg(0)
		if e != nil {
			return nil, e
		}
		x, e := v.module(name)
		if e != nil {
			return nil, e
		}
		return c.PushingNext1(t.Runtime, x), nil
	}, 1, false)
	rt.SolemnlyDeclareCompliance(rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe, requireFn)
	ctx := v.r.NewTable()
	for _, name := range methods {
		name := name
		fn := v.r.SetEnvGoFunc(ctx, name, func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
			t.RequireCPU(100)
			if len(c.Etc()) > 8 {
				return nil, fmt.Errorf("AI %s: too many arguments", name)
			}
			if v.host == nil {
				return nil, fmt.Errorf("AI %s outside world event", name)
			}
			args := make([]any, 0, len(c.Etc()))
			for _, a := range c.Etc() {
				switch a.Type() {
				case rt.NilType, rt.IntType, rt.FloatType, rt.BoolType, rt.StringType:
					args = append(args, a.Interface())
				default:
					return nil, fmt.Errorf("AI %s: unsupported argument %s", name, a.TypeName())
				}
			}
			result, e := v.host.Call(name, args)
			if e != nil {
				return nil, fmt.Errorf("%s: %w", name, e)
			}
			x, e := luaValue(result)
			if e != nil {
				return nil, e
			}
			return c.PushingNext1(t.Runtime, x), nil
		}, 1, true)
		rt.SolemnlyDeclareCompliance(rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe, fn)
	}
	v.context = rt.TableValue(ctx)
	var err error
	v.catalog, err = v.module("catalog")
	if err != nil {
		v.Close()
		return nil, err
	}
	return v, nil
}
func (v *VM) Close() { var err error; v.r.Close(&err) }
func (v *VM) module(name string) (rt.Value, error) {
	if x, ok := v.modules[name]; ok {
		return x, nil
	}
	source, ok := v.program.source[name]
	if !ok {
		return rt.NilValue, fmt.Errorf("missing AI module %q", name)
	}
	v.program.mu.Lock()
	unit := v.program.units[name]
	if unit == nil {
		var e error
		unit, _, e = v.r.CompileLuaChunk(name, source)
		if e != nil {
			v.program.mu.Unlock()
			return rt.NilValue, e
		}
		v.program.units[name] = unit
	}
	v.program.mu.Unlock()
	chunk := v.r.LoadLuaUnit(unit, rt.TableValue(v.r.GlobalEnv()))
	result, err := v.call(rt.FunctionValue(chunk))
	if err != nil {
		return rt.NilValue, fmt.Errorf("AI module %s: %w", name, err)
	}
	v.modules[name] = result
	return result, nil
}
func (v *VM) call(fn rt.Value, args ...rt.Value) (result rt.Value, err error) {
	_, err = v.r.MainThread().CallContext(rt.RuntimeContextDef{HardLimits: rt.RuntimeResources{Cpu: 500000, Memory: 8 << 20}}, func() error { var e error; result, e = rt.Call1(v.r.MainThread(), fn, args...); return e })
	return
}
func luaValue(value any) (rt.Value, error) {
	switch x := value.(type) {
	case nil:
		return rt.NilValue, nil
	case bool:
		return rt.BoolValue(x), nil
	case string:
		return rt.StringValue(x), nil
	case int:
		return rt.IntValue(int64(x)), nil
	case int64:
		return rt.IntValue(x), nil
	case uint32:
		return rt.IntValue(int64(x)), nil
	case float32:
		return rt.FloatValue(float64(x)), nil
	case float64:
		return rt.FloatValue(x), nil
	default:
		return rt.NilValue, fmt.Errorf("unsupported AI result %T", value)
	}
}

type Actor struct {
	vm    *VM
	value rt.Value
}

func (v *VM) NewActor(id uint8, host Host) (*Actor, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.host = host
	defer func() { v.host = nil }()
	resolve := v.catalog.AsTable().Get(rt.StringValue("resolve"))
	spec, err := v.call(resolve, rt.IntValue(int64(id)))
	if err != nil {
		return nil, err
	}
	value, err := v.call(spec.AsTable().Get(rt.StringValue("new")), v.context)
	if err != nil {
		return nil, err
	}
	return &Actor{vm: v, value: value}, nil
}
func (a *Actor) Event(host Host, event string, args ...any) error {
	v := a.vm
	v.mu.Lock()
	defer v.mu.Unlock()
	v.host = host
	defer func() { v.host = nil }()
	values := []rt.Value{a.value, v.context, rt.StringValue(event)}
	for _, x := range args {
		val, err := luaValue(x)
		if err != nil {
			return err
		}
		values = append(values, val)
	}
	_, err := v.call(a.value.AsTable().Get(rt.StringValue("event")), values...)
	return err
}

func (p *Program) AcquiresProximity(id uint8) bool { return p.acquire[id] }
