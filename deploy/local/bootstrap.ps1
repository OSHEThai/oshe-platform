# Deterministic local stack bootstrap (synthetic-only; no production data).
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true
$compose = Join-Path $PSScriptRoot 'compose.dev.yaml'
$env = Join-Path $PSScriptRoot '.env.example'
docker compose -f $compose --env-file $env up -d --wait
if ($LASTEXITCODE -ne 0) { throw "Compose failed" }
& (Join-Path $PSScriptRoot 'seed-synthetic.ps1')
Write-Host 'OSHE_LOCAL_STACK_BOOTSTRAP=COMPLETE'
