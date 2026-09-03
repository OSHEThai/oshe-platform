# Environment report (static, synthetic-only). Emits one JSON object per service
# with allowlisted fields only: project, service, image. Never secrets.
$ErrorActionPreference = 'Stop'

$compose = Join-Path $PSScriptRoot 'compose.dev.yaml'
$text = Get-Content -LiteralPath $compose -Raw

$project = 'unknown'
if ($text -match '(?m)^name:\s*(\S+)') { $project = $Matches[1] }

$blocks = [regex]::Split($text, '(?m)(?=^  [a-z0-9_-]+:\s*$)')
foreach ($block in $blocks) {
    if ($block -notmatch '(?m)^  ([a-z0-9_-]+):\s*$') { continue }
    $svc = $Matches[1]
    if ($svc -eq 'volumes') { continue }
    if ($block -notmatch '(?m)^\s*image:\s*(\S+)') { continue }
    $image = $Matches[1]
    [ordered]@{ project = $project; service = $svc; image = $image } | ConvertTo-Json -Compress
}
