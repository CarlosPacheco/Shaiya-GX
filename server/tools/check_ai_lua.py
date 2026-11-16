#!/usr/bin/env python3
"""Run the standalone AI Lua tests using a Lua 5.4+ shared library.

No server, database, game assets, Go dependency or installed runtime is started.
A Lua CLI can alternatively execute server/ai/tests/run.lua directly.
"""
import argparse
import ctypes
import ctypes.util
from pathlib import Path


def run(library: str) -> None:
    lua = ctypes.CDLL(library)
    pointer = ctypes.c_void_p
    lua.luaL_newstate.restype = pointer
    signatures = {
        "luaL_loadbufferx": ([pointer, ctypes.c_char_p, ctypes.c_size_t, ctypes.c_char_p, ctypes.c_char_p], ctypes.c_int),
        "lua_pcallk": ([pointer, ctypes.c_int, ctypes.c_int, ctypes.c_int, ctypes.c_ssize_t, pointer], ctypes.c_int),
        "lua_tolstring": ([pointer, ctypes.c_int, pointer], ctypes.c_char_p),
        "lua_close": ([pointer], None),
    }
    for name, (args, result) in signatures.items():
        fn = getattr(lua, name)
        fn.argtypes, fn.restype = args, result
    state = lua.luaL_newstate()
    if not state:
        raise RuntimeError("could not create Lua state")
    try:
        if hasattr(lua, "luaL_openlibs"):
            lua.luaL_openlibs.argtypes = [pointer]
            lua.luaL_openlibs.restype = None
            lua.luaL_openlibs(state)
        else:
            # Lua 5.5 makes openlibs a macro over selected libraries.
            lua.luaL_openselectedlibs.argtypes = [pointer, ctypes.c_int, ctypes.c_int]
            lua.luaL_openselectedlibs.restype = None
            lua.luaL_openselectedlibs(state, -1, 0)
        root = Path(__file__).resolve().parents[1] / "ai"
        # Lua source uses a quoted path, not shell evaluation.
        escaped = str(root).replace("\\", "\\\\").replace('"', '\\"')
        source = f'package.path = "{escaped}/?.lua;" .. package.path\ndofile("{escaped}/tests/run.lua")'.encode()
        result = lua.luaL_loadbufferx(state, source, len(source), b"@check_ai_lua", b"t")
        if result == 0:
            result = lua.lua_pcallk(state, 0, 0, 0, 0, None)
        if result:
            message = lua.lua_tolstring(state, -1, None)
            raise RuntimeError(message.decode(errors="replace") if message else f"Lua error {result}")
    finally:
        lua.lua_close(state)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--library", help="Lua 5.4+ shared library path/name")
    args = parser.parse_args()
    library = args.library or ctypes.util.find_library("lua5.4")
    if not library:
        parser.error("Lua 5.4 shared library not found; supply --library or use a Lua CLI")
    run(library)


if __name__ == "__main__":
    main()
