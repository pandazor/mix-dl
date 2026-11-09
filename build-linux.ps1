$ErrorActionPreference = 'Stop'
$projectDir = $PSScriptRoot
$toolchainDir = Join-Path $projectDir '..\toolchain'
$goExecutable = Join-Path $toolchainDir 'go\bin\go.exe'
$savedGoEnvironment = @{}
foreach ($name in 'GOOS', 'GOARCH', 'CGO_ENABLED', 'GOPATH', 'GOCACHE') {
    $savedGoEnvironment[$name] = [Environment]::GetEnvironmentVariable($name)
}
Push-Location $projectDir
try {
    if (Test-Path -LiteralPath $goExecutable) {
        $env:GOPATH = Join-Path $toolchainDir 'gopath'
        $env:GOCACHE = Join-Path $toolchainDir 'gocache'
    } else {
        $goExecutable = (Get-Command go -ErrorAction Stop).Source
    }
    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    $tsCompiler = Join-Path $toolchainDir 'typescript\package\bin\tsc'
    if (Test-Path -LiteralPath $tsCompiler) {
        & node $tsCompiler -p frontend\tsconfig.json
    } else {
        & npx --no-install tsc -p frontend\tsconfig.json
    }
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    New-Item -ItemType Directory -Force -Path bin\linux-x64 | Out-Null
    & $goExecutable build -trimpath -tags purego -ldflags '-s -w' -o bin\linux-x64\dl-manager .
    exit $LASTEXITCODE
} finally {
    foreach ($name in $savedGoEnvironment.Keys) {
        $value = $savedGoEnvironment[$name]
        if ($null -eq $value) { $value = [NullString]::Value }
        [Environment]::SetEnvironmentVariable($name, $value)
    }
    Pop-Location
}
