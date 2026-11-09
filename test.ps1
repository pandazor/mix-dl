$ErrorActionPreference = 'Stop'
$env:GOPATH = Join-Path $PSScriptRoot '..\toolchain\gopath'
$env:GOCACHE = Join-Path $PSScriptRoot '..\toolchain\gocache'
$env:CGO_ENABLED = '0'
$goExecutable = Join-Path $PSScriptRoot '..\toolchain\go\bin\go.exe'
if (-not (Test-Path -LiteralPath $goExecutable)) { $goExecutable = (Get-Command go -ErrorAction Stop).Source }
Push-Location $PSScriptRoot
try { & $goExecutable test -count=1 ./...; exit $LASTEXITCODE } finally { Pop-Location }
