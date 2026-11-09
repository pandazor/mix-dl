param([switch]$Debug)
$ErrorActionPreference = 'Stop'
$projectDir = $PSScriptRoot
$toolchainDir = Join-Path $projectDir '..\toolchain'
$savedGoEnvironment = @{}
foreach ($name in 'GOOS', 'GOARCH', 'CGO_ENABLED', 'GOPATH', 'GOCACHE') {
    $savedGoEnvironment[$name] = [Environment]::GetEnvironmentVariable($name)
}
Push-Location $projectDir
try {
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:GOPATH = Join-Path $toolchainDir 'gopath'
    $env:GOCACHE = Join-Path $toolchainDir 'gocache'
    $env:CGO_ENABLED = '0'
    $goExecutable = Join-Path $toolchainDir 'go\bin\go.exe'
    if (-not (Test-Path -LiteralPath $goExecutable)) { $goExecutable = (Get-Command go -ErrorAction Stop).Source }
    $tsCompiler = Join-Path $toolchainDir 'typescript\package\bin\tsc'
    if (Test-Path -LiteralPath $tsCompiler) {
        & node $tsCompiler -p frontend\tsconfig.json
    } else {
        & npx --no-install tsc -p frontend\tsconfig.json
    }
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    New-Item -ItemType Directory -Force -Path bin | Out-Null
    & $goExecutable mod tidy
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    $linkFlags = '-s -w -H windowsgui'
    if ($Debug) { $linkFlags = '-s -w' }
    & $goExecutable build -trimpath -ldflags $linkFlags -o bin\dl-manager.exe .
    exit $LASTEXITCODE
} finally {
    foreach ($name in $savedGoEnvironment.Keys) {
        $value = $savedGoEnvironment[$name]
        if ($null -eq $value) { $value = [NullString]::Value }
        [Environment]::SetEnvironmentVariable($name, $value)
    }
    Pop-Location
}
