# Deterministic teardown + rebuild (synthetic-only).
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true
$compose = Join-Path $PSScriptRoot 'compose.dev.yaml'
$env = Join-Path $PSScriptRoot '.env.example'
docker compose -f $compose --env-file $env down -v
if ($LASTEXITCODE -ne 0) { throw "Compose down failed" }
docker compose -f $compose --env-file $env up -d --wait
if ($LASTEXITCODE -ne 0) { throw "Compose up failed" }
Write-Host 'OSHE_LOCAL_STACK_REBUILD=COMPLETE'
