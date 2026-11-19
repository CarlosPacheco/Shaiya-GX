# Source this file: Ghidra headless environment for the Shaiya EP5.4 client.
export JAVA_HOME="$HOME/.cache/phoenix-reverse/ghidra/jdk-21.0.12.1+1"
export GHIDRA_HOME="$HOME/.cache/phoenix-reverse/ghidra/ghidra_12.1.3_PUBLIC"
export GHIDRA_PROJECT_DIR="${GHIDRA_PROJECT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/work/ghidra}"
export GHIDRA_PROJECT=GameEP54
export SHAIYA_CLIENT_EXE="${SHAIYA_CLIENT_EXE:-$HOME/Desktop/Client EP5/Game.exe}"
export GHIDRA_SCRIPTS="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# ghidra_run SCRIPT [args...]: run a script read-only against the imported program.
ghidra_run() {
  local script="$1"; shift
  nice -n 15 "$GHIDRA_HOME/support/analyzeHeadless" "$GHIDRA_PROJECT_DIR" "$GHIDRA_PROJECT" \
    -process Game.exe -readOnly -noanalysis \
    -scriptPath "$GHIDRA_SCRIPTS" -postScript "$script" "$@" 2>&1 | awk '/^INFO  [A-Za-z0-9_]+\.java> /{sub(/^INFO  [A-Za-z0-9_]+\.java> /,""); sub(/ \(GhidraScript\) *$/,""); print; m=1; next} /^(INFO|WARN|ERROR) /{m=0; if ($1=="ERROR") print; next} m{sub(/ \(GhidraScript\) *$/,""); print}'
}
# ghidra_run_server SCRIPT [args...]: same, against Eternity's ps_game.exe (EP4.5 server with PDB names).
export SHAIYA_SERVER_PROJECT_DIR="${SHAIYA_SERVER_PROJECT_DIR:-$HOME/Projects/Eternity/work/ghidra-12.1.3}"
ghidra_run_server() {
  local script="$1"; shift
  nice -n 15 "$GHIDRA_HOME/support/analyzeHeadless" "$SHAIYA_SERVER_PROJECT_DIR" Eternity \
    -process ps_game.exe -readOnly -noanalysis \
    -scriptPath "$GHIDRA_SCRIPTS" -postScript "$script" "$@" 2>&1 | awk '/^INFO  [A-Za-z0-9_]+\.java> /{sub(/^INFO  [A-Za-z0-9_]+\.java> /,""); sub(/ \(GhidraScript\) *$/,""); print; m=1; next} /^(INFO|WARN|ERROR) /{m=0; if ($1=="ERROR") print; next} m{sub(/ \(GhidraScript\) *$/,""); print}'
}
