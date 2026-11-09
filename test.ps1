$ErrorActionPreference = 'Stop'
$savedGoEnvironment = @{}
foreach ($name in 'GOOS', 'GOARCH', 'CGO_ENABLED', 'GOPATH', 'GOCACHE') {
    $savedGoEnvironment[$name] = [Environment]::GetEnvironmentVariable($name)
}
Push-Location $PSScriptRoot
try {
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:GOPATH = Join-Path $PSScriptRoot '..\toolchain\gopath'
    $env:GOCACHE = Join-Path $PSScriptRoot '..\toolchain\gocache'
    $env:CGO_ENABLED = '0'
    $goExecutable = Join-Path $PSScriptRoot '..\toolchain\go\bin\go.exe'
    if (-not (Test-Path -LiteralPath $goExecutable)) { $goExecutable = (Get-Command go -ErrorAction Stop).Source }
    & $goExecutable test -count=1 ./...
    exit $LASTEXITCODE
} finally {
    foreach ($name in $savedGoEnvironment.Keys) {
        $value = $savedGoEnvironment[$name]
        if ($null -eq $value) { $value = [NullString]::Value }
        [Environment]::SetEnvironmentVariable($name, $value)
    }
    Pop-Location
}
