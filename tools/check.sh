#!/usr/bin/env bash
# Cached checks for the affected packages; defaults to the whole module.
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ ${1:-} == --help ]]; then
  echo 'Usage: tools/check.sh [Go package ...]'
  echo 'Example: tools/check.sh ./internal/proto ./server/internal/world'
  echo 'Runs vet and cached unit tests. Integration/N2 remain separate checks.'
  exit 0
fi
source tools/go-env.sh
if [[ $# == 0 ]]; then set -- ./...; fi
go vet "$@"
go test -vet=off "$@"
