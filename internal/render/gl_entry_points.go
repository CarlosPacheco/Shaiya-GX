package render

/*
#include <stdio.h>
#include <stdlib.h>

#ifdef _WIN32
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#else
#define APIENTRY
#endif

static void unsupportedQuery(const char *name) {
	fprintf(stderr, "OpenGL entry point %s was unavailable at startup\n", name);
	abort();
}

static void APIENTRY unavailableGetnTexImage(unsigned int target, int level, unsigned int format,
	unsigned int type, int size, void *pixels) {
	unsupportedQuery("glGetnTexImage");
}

static void APIENTRY unavailableGetnCompressedTexImage(unsigned int target, int level, int size, void *pixels) {
	unsupportedQuery("glGetnCompressedTexImage");
}

static void APIENTRY unavailableGetnUniformdv(unsigned int program, int location, int size, double *params) {
	unsupportedQuery("glGetnUniformdv");
}

enum { GX_TEXTURE_QUERY, GX_COMPRESSED_TEXTURE_QUERY, GX_DOUBLE_UNIFORM_QUERY };

static void *unavailableReadback(int query) {
	switch (query) {
	case GX_TEXTURE_QUERY: return (void *)unavailableGetnTexImage;
	case GX_COMPRESSED_TEXTURE_QUERY: return (void *)unavailableGetnCompressedTexImage;
	case GX_DOUBLE_UNIFORM_QUERY: return (void *)unavailableGetnUniformdv;
	default: return NULL;
	}
}
*/
import "C"

import (
	"log"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

func loadGLEntryPoints(resolve func(string) unsafe.Pointer) error {
	return gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
		if addr := resolve(name); validGLAddress(addr) {
			return addr
		}
		// Some desktop drivers omit these robustness queries. The generated
		// binding requires them even though GX never calls them. Prefer the
		// same-signature ARB export and trap any future use if both are absent.
		var query C.int
		switch name {
		case "glGetnTexImage":
			query = C.GX_TEXTURE_QUERY
		case "glGetnCompressedTexImage":
			query = C.GX_COMPRESSED_TEXTURE_QUERY
		case "glGetnUniformdv":
			query = C.GX_DOUBLE_UNIFORM_QUERY
		default:
			return nil
		}
		alias := name + "ARB"
		if addr := resolve(alias); validGLAddress(addr) {
			log.Printf("OpenGL entry point %s: using %s", name, alias)
			return addr
		}
		log.Printf("OpenGL entry point %s unavailable; GX does not use it", name)
		return C.unavailableReadback(query)
	})
}

func validGLAddress(addr unsafe.Pointer) bool {
	// WGL implementations can return 1, 2, 3 or -1 for missing functions.
	return uintptr(addr) > 3 && uintptr(addr) != ^uintptr(0)
}
