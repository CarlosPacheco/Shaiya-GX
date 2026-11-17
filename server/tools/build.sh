#!/bin/sh
# Build the terminal console and login/world services without CGO.
set -e
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"
if [ -x "$HOME/.local/opt/go-1.27.4/bin/go" ]; then
    PATH="$HOME/.local/opt/go-1.27.4/bin:$PATH"
    export PATH
fi
export GOTOOLCHAIN=local
CGO_ENABLED=0 go build -o build/ ./cmd/gx_login ./cmd/gx_world ./cmd/gx_console
