package ui

import (
	"reflect"
	"unsafe"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

var buttonType = reflect.TypeOf(Button{})

// ButtonHit reports whether the absolute point lies on any Button held by the
// window struct (found recursively through struct, array and slice fields).
// Controls are positioned every frame by Update/Draw, so hit boxes are current.
// Native frames give controls priority over the whole-frame drag (0x526f30).
func ButtonHit(owner any, mx, my int) bool {
	v := reflect.ValueOf(owner)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return false
	}
	probe := platform.Input{MouseX: mx, MouseY: my}
	var walk func(reflect.Value) bool
	walk = func(f reflect.Value) bool {
		switch f.Kind() {
		case reflect.Struct:
			if f.Type() == buttonType {
				if !f.CanAddr() {
					return false
				}
				b := (*Button)(unsafe.Pointer(f.UnsafeAddr()))
				// Update/Draw place controls every frame; one never placed (or
				// hidden by the current mode) has no hit box yet.
				if b.OriginX == 0 && b.OriginY == 0 {
					return false
				}
				return b.HitW > 0 && b.hit(&probe)
			}
			for i := 0; i < f.NumField(); i++ {
				if walk(f.Field(i)) {
					return true
				}
			}
		case reflect.Array, reflect.Slice:
			if k := f.Type().Elem().Kind(); k == reflect.Struct || k == reflect.Array {
				for i := 0; i < f.Len(); i++ {
					if walk(f.Index(i)) {
						return true
					}
				}
			}
		}
		return false
	}
	return walk(v.Elem())
}
