//go:build integration

package render

import (
	"os"
	"runtime"
	"testing"
	"unsafe"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.4/glfw"
)

func TestOpenGLStartupWithoutUnusedReadbackQueries(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires a local OpenGL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Width: 32, Height: 32, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	defer func() {
		if err := loadGLEntryPoints(glfw.GetProcAddress); err != nil {
			t.Error(err)
		}
	}()

	resolve := func(name string) unsafe.Pointer {
		switch name {
		case "glGetnTexImage", "glGetnTexImageARB":
			return nil
		}
		return glfw.GetProcAddress(name)
	}
	if err := gl.InitWithProcAddrFunc(resolve); err == nil || err.Error() != "glGetnTexImage" {
		t.Fatalf("unmodified loader returned %v, want glGetnTexImage", err)
	}
	if err := loadGLEntryPoints(func(name string) unsafe.Pointer {
		switch name {
		case "glGetnCompressedTexImage", "glGetnCompressedTexImageARB", "glGetnUniformdv", "glGetnUniformdvARB":
			return nil
		}
		return resolve(name)
	}); err != nil {
		t.Fatal(err)
	}
	gl.ClearColor(1, 0, 1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	var pixel [4]byte
	gl.ReadPixels(0, 0, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
	if pixel != [4]byte{255, 0, 255, 255} || gl.GetError() != gl.NO_ERROR {
		t.Fatalf("framebuffer readback failed: %v", pixel)
	}
	if err := loadGLEntryPoints(func(name string) unsafe.Pointer {
		if name == "glCreateTextures" {
			return nil
		}
		return resolve(name)
	}); err == nil || err.Error() != "glCreateTextures" {
		t.Fatalf("loader accepted a missing required function: %v", err)
	}
}
