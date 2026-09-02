# Static contract checker for the I010 local stack.
# Rejects: tag-only/unpinned image references, services without a healthcheck,
# production-like settings, and direct-projection authority claims.
[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [string]$ComposePath,
    [Parameter(Position = 1)]
    [string]$EnvPath
)

$ErrorActionPreference = 'Stop'
if (-not $ComposePath) { $ComposePath = Join-Path $PSScriptRoot '..\..\deploy\local\compose.dev.yaml' }
if (-not $EnvPath) { $EnvPath = Join-Path $PSScriptRoot '..\..\deploy\local\.env.example' }

if (-not (Test-Path -LiteralPath $ComposePath -PathType Leaf)) {
    throw "Compose manifest not found: $ComposePath"
}

$compose = Get-Content -LiteralPath $ComposePath -Raw
$env = if (Test-Path -LiteralPath $EnvPath -PathType Leaf) { Get-Content -LiteralPath $EnvPath -Raw } else { '' }
$services = @('postgres', 'postgis', 'meilisearch', 'valkey', 'seaweedfs', 'nats')

# 1. Every image reference must be digest-pinned (contain @sha256:).
$imageLines = [regex]::Matches($compose, '(?m)^\s*image:\s*(.+)$') | ForEach-Object { $_.Groups[1].Value.Trim() }
if ($imageLines.Count -eq 0) {
    throw 'CONTRACT_VIOLATION: no image references found'
}
foreach ($img in $imageLines) {
    if ($img -notmatch '@sha256:') {
        throw "CONTRACT_VIOLATION: unpinned/tag-only image reference: $img"
    }
}

# 2. Every service must declare a healthcheck (block-bounded).
foreach ($svc in $services) {
    if ($compose -notmatch "(?m)^  ${svc}:(?:\n(?:    .*|      .*| {5,}.*))*?\n    healthcheck:") {
        throw "CONTRACT_VIOLATION: service missing healthcheck: $svc"
    }
}

# 3. No production-like settings (real cloud endpoints or production hosts).
if ($compose -match 'amazonaws\.com|\.cloud\.|production_endpoint|prod\.oshe|PROD_') {
    throw 'CONTRACT_VIOLATION: production-like setting detected'
}
if ($env -match 'amazonaws\.com|\.cloud\.|production_endpoint|prod\.oshe') {
    throw 'CONTRACT_VIOLATION: production-like value in environment example'
}

# 4. No direct-projection authority claims (search/cache must remain non-authoritative).
foreach ($line in ($compose -split "`n")) {
    $low = $line.ToLowerInvariant()
    if (($low -match 'meilisearch' -or $low -match 'valkey') -and $low -match 'authoritative' -and $low -notmatch 'non-authoritative') {
        throw "CONTRACT_VIOLATION: direct projection authority claim: $line"
    }
}

# 5. Exact Dockerfile parity with the required digest refs.
$dockerfile = Get-Content -LiteralPath (Join-Path $PSScriptRoot '..\..\.devcontainer\Dockerfile') -Raw
$go_matches = [regex]::Matches($dockerfile, "(?m)^FROM golang:1\.26\.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go$")
if ($go_matches.Count -ne 1) { throw 'CONTRACT_VIOLATION: Dockerfile missing exact golang digest' }

$node_matches = [regex]::Matches($dockerfile, "(?m)^FROM node:24\.20\.0-alpine@sha256:e67514e5d0f6c46656005e1b693b2ec9d52e80b641307de684d4a015ba7a4eaf AS node$")
if ($node_matches.Count -ne 1) { throw 'CONTRACT_VIOLATION: Dockerfile missing exact node digest' }

$py_matches = [regex]::Matches($dockerfile, "(?m)^FROM python:3\.14\.7-alpine@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc$")
if ($py_matches.Count -ne 1) { throw 'CONTRACT_VIOLATION: Dockerfile missing exact python digest' }

$all_froms = [regex]::Matches($dockerfile, "(?m)^FROM .*$")
if ($all_froms.Count -ne 3) { throw 'CONTRACT_VIOLATION: Dockerfile contains alternate or duplicate FROM instructions' }

# 6. devcontainer JSON remoteUser exactly vscode
$devcontainerPath = Join-Path $PSScriptRoot '..\..\.devcontainer\devcontainer.json'
$dev_cfg = Get-Content -LiteralPath $devcontainerPath -Raw | ConvertFrom-Json
if ($dev_cfg.remoteUser -cne 'vscode') { throw 'CONTRACT_VIOLATION: devcontainer remoteUser must be exactly vscode' }

# 7. Fail-closed bootstrap and teardown
$bootstrap = Get-Content -LiteralPath (Join-Path $PSScriptRoot '..\..\deploy\local\bootstrap.ps1') -Raw
if ($bootstrap -notmatch '(?m)^docker compose.*$\r?\n^if \(\$LASTEXITCODE -ne 0\) \{ throw .* \}$') {
    throw 'CONTRACT_VIOLATION: bootstrap missing immediately following structural guard'
}
$teardown = Get-Content -LiteralPath (Join-Path $PSScriptRoot '..\..\deploy\local\teardown-rebuild.ps1') -Raw
if ($teardown -notmatch '(?m)^docker compose -f .*? down -v$\r?\n^if \(\$LASTEXITCODE -ne 0\) \{ throw .* \}$') {
    throw 'CONTRACT_VIOLATION: teardown down missing immediately following structural guard'
}
if ($teardown -notmatch '(?m)^docker compose -f .*? up -d --wait$\r?\n^if \(\$LASTEXITCODE -ne 0\) \{ throw .* \}$') {
    throw 'CONTRACT_VIOLATION: teardown up missing immediately following structural guard'
}

Write-Host 'V010_I010_LOCAL_STACK_CONTRACT=PASS'
