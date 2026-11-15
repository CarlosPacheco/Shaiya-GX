#!/usr/bin/env bash
# Shared Linux Go environment; source from repository scripts.
if [[ -x "$HOME/.local/opt/go-1.27.3/bin/go" ]]; then
  export PATH="$HOME/.local/opt/go-1.27.3/bin:$PATH"
fi
export GOTOOLCHAIN=local
if ! command -v go >/dev/null 2>&1; then
  echo 'Go 1.27.3 is required; install it or add it to PATH.' >&2
  return 1
fi
# Preserve Go's build/test caches and default CPU parallelism.
