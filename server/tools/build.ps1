# Build the terminal console and login/world services without CGO.
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)
$previousCGO = $env:CGO_ENABLED
try {
    $env:CGO_ENABLED = "0"
    go build -o build\ ./cmd/gx_login ./cmd/gx_world ./cmd/gx_console
} finally {
    if ($null -eq $previousCGO) { Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue }
    else { $env:CGO_ENABLED = $previousCGO }
}
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
