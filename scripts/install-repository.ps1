[CmdletBinding()]
param(
    [string]$Destination = 'G:\Documents\ChatGPT\mix-dl',
    [string]$ReleaseDirectory = $PSScriptRoot,
    [string]$Remote = 'git@github.com:pandazor/mix-dl.git',
    [string]$SshSourceRepository = 'G:\Documents\ChatGPT\maison-work\nouveau-jeu',
    [switch]$NoPush
)
$ErrorActionPreference = 'Stop'
function Invoke-Git {
    param([Parameter(ValueFromRemainingArguments=$true)][string[]]$Arguments)
    & git @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Git a echoue (code $LASTEXITCODE). Aucun push force ne sera tente." }
}
$release = (Resolve-Path -LiteralPath $ReleaseDirectory).Path
$target = [System.IO.Path]::GetFullPath($Destination)
if ($target.TrimEnd('\') -eq [System.IO.Path]::GetPathRoot($target).TrimEnd('\')) {
    throw 'Choisissez un sous-dossier, pas la racine du disque.'
}
$bundle = Join-Path $release 'mix-dl-0.2.0.bundle'
$checksumFile = Join-Path $release 'transfert-SHA256SUMS.txt'
if (-not (Test-Path -LiteralPath $bundle) -or -not (Test-Path -LiteralPath $checksumFile)) {
    throw 'Bundle et sommes de controle manquants. Extraire tout le kit de transfert dans un meme dossier.'
}
foreach ($line in Get-Content -LiteralPath $checksumFile) {
    if ($line -notmatch '^([a-fA-F0-9]{64})  ([a-zA-Z0-9_.-]+)$') { throw 'Format des empreintes invalide.' }
    $expected = $Matches[1]
    $file = Join-Path $release $Matches[2]
    if ((Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash -ne $expected) { throw "Empreinte incorrecte : $file" }
}
$headLine = Invoke-Git bundle list-heads $bundle refs/heads/main
$wantedHead = ($headLine -split ' ')[0]
if (Test-Path -LiteralPath (Join-Path $target '.git')) {
    $existingHead = Invoke-Git -C $target rev-parse HEAD
    if ($existingHead -ne $wantedHead) { throw 'Ce dossier contient un autre etat Git. Rien ne sera ecrase. Choisissez un dossier vide.' }
    if (Invoke-Git -C $target status --porcelain) { throw 'Ce depot contient des modifications locales. Rien ne sera ecrase.' }
} else {
    if ((Test-Path -LiteralPath $target) -and @(Get-ChildItem -LiteralPath $target -Force).Count -gt 0) {
        throw 'Le dossier de destination n est pas vide. Rien ne sera ecrase.'
    }
    Invoke-Git clone --branch main -- $bundle $target
}
Invoke-Git -C $target remote set-url origin $Remote
if (Test-Path -LiteralPath (Join-Path $SshSourceRepository '.git')) {
    $sshCommand = & git -C $SshSourceRepository config --get core.sshCommand
    if ($LASTEXITCODE -eq 0 -and $sshCommand) { Invoke-Git -C $target config core.sshCommand $sshCommand }
}
$dist = Join-Path $target 'dist'
New-Item -ItemType Directory -Path $dist -Force | Out-Null
foreach ($name in @('dl-manager-0.2.0-windows-x64.zip','dl-manager-0.2.0-linux-x64.tar.gz','dl-manager-0.2.0-SHA256SUMS.txt')) {
    $source = Join-Path $release $name
    if (Test-Path -LiteralPath $source) { Copy-Item -LiteralPath $source -Destination (Join-Path $dist $name) -Force }
}
Write-Host "Projet copie dans $target ; paquets dans $dist."
if (-not $NoPush) {
    Invoke-Git -C $target push -u origin main
    Write-Host "Sources poussees vers $Remote. Les paquets restent dans dist (non versionnes)."
} else { Write-Host 'Push ignore (-NoPush).' }
