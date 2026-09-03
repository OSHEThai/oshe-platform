# Deterministic teardown + rebuild (synthetic-only). Fails closed.
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

$compose = Join-Path $PSScriptRoot 'compose.dev.yaml'
$env = Join-Path $PSScriptRoot '.env.example'

# Positive local-disposable-target identification before any destructive command.
# Exactly one canonical top-level name; case-insensitive unsafe-marker rejection.
$composeText = Get-Content -LiteralPath $compose -Raw
$nameMatches = [regex]::Matches($composeText, '(?m)^name:\s*(\S+)\s*$')
if ($nameMatches.Count -ne 1 -or $nameMatches[0].Groups[1].Value -ne 'oshe-local') {
    throw 'RESET_REFUSED: local disposable target (oshe-local) not positively identified'
}
$envText = Get-Content -LiteralPath $env -Raw
$lower = ($composeText + "`n" + $envText).ToLowerInvariant()
if ($lower -match 'amazonaws\.com|\.cloud\.|prod_|production_endpoint|prod\.oshe') {
    throw 'RESET_REFUSED: production-like marker detected'
}

docker compose -f $compose --env-file $env down -v
if ($LASTEXITCODE -ne 0) { throw "Compose down failed" }
docker compose -f $compose --env-file $env up -d --wait
if ($LASTEXITCODE -ne 0) { throw "Compose up failed" }
Write-Host 'OSHE_LOCAL_STACK_REBUILD=COMPLETE'
