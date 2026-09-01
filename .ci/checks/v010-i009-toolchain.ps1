$ErrorActionPreference = 'Stop'
$lockPath = Join-Path $PSScriptRoot '..\..\toolchain.lock.yaml'
$required = @('1.26.5', '24.20.0', '11.24.0', '3.14.7', '29.7.2', '5.4.0', '17.11', '1.51.0', '9.1.1', '4.29', '2.14.5', 'PENDING_NO_NETWORK')
$content = Get-Content -Raw -LiteralPath $lockPath
foreach ($value in $required) {
    if (-not $content.Contains($value)) { throw "Missing expected toolchain value: $value" }
}
if ($content -match '(?m):\s*latest\s*(?:$|#)') { throw 'Floating latest alias is prohibited' }
Write-Output 'V010_I009_TOOLCHAIN_STATIC_CHECK=PASS'
