param([switch]$Debug)
$ErrorActionPreference = 'Stop'
$projectDir = $PSScriptRoot
$toolchainDir = Join-Path $projectDir '..\toolchain'
$env:GOPATH = Join-Path $toolchainDir 'gopath'
$env:GOCACHE = Join-Path $toolchainDir 'gocache'
$env:CGO_ENABLED = '0'
$goExecutable = Join-Path $toolchainDir 'go\bin\go.exe'
if (-not (Test-Path -LiteralPath $goExecutable)) { $goExecutable = (Get-Command go -ErrorAction Stop).Source }
$tsCompiler = Join-Path $toolchainDir 'typescript\package\bin\tsc'
if (Test-Path -LiteralPath $tsCompiler) { & node $tsCompiler -p (Join-Path $projectDir 'frontend\tsconfig.json') } else { Push-Location $projectDir; try { & npx --no-install tsc -p frontend\tsconfig.json } finally { Pop-Location } }
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
New-Item -ItemType Directory -Force -Path (Join-Path $projectDir 'bin') | Out-Null
Push-Location $projectDir
try {
 & $goExecutable mod tidy
 if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
 $linkFlags = '-s -w -H windowsgui'
 if ($Debug) { $linkFlags = '-s -w' }
 & $goExecutable build -trimpath -ldflags $linkFlags -o bin\dl-manager.exe .
 exit $LASTEXITCODE
} finally { Pop-Location }
