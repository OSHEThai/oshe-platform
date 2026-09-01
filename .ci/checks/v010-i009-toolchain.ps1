[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [string]$LockPath
)

$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($LockPath)) {
    $LockPath = Join-Path $PSScriptRoot '..\..\toolchain.lock.yaml'
}
if (-not (Test-Path -LiteralPath $LockPath -PathType Leaf)) {
    throw "Toolchain lock was not found: $LockPath"
}
$LockPath = (Resolve-Path -LiteralPath $LockPath).Path
$content = Get-Content -Raw -LiteralPath $LockPath

if ($content -match '(?im)(?:^|[:=,{\[])\s*["'']?latest["'']?\s*(?:[,}\]]|#|$)') {
    throw 'Floating latest alias is prohibited'
}

function Add-ObservedValue {
    param(
        [hashtable]$Table,
        [string]$Path,
        [string]$Value
    )
    if ($Table.ContainsKey($Path)) {
        $Table[$Path] = @($Table[$Path]) + $Value
    }
    else {
        $Table[$Path] = @($Value)
    }
}

function Normalize-Value {
    param([string]$Value)
    $normalized = $Value.Trim()
    if (($normalized.Length -ge 2) -and
        (($normalized.StartsWith('"') -and $normalized.EndsWith('"')) -or
         ($normalized.StartsWith("'") -and $normalized.EndsWith("'")))) {
        $normalized = $normalized.Substring(1, $normalized.Length - 2)
    }
    return $normalized
}

$observed = @{}
$section = $null
foreach ($rawLine in (Get-Content -LiteralPath $LockPath)) {
    $line = $rawLine -replace '\s+#.*$', ''
    if ([string]::IsNullOrWhiteSpace($line)) { continue }

    if ($line -match '^(?<indent> *)(?<key>[A-Za-z0-9_-]+):(?:\s*(?<rest>.*))?$') {
        $indent = $matches['indent'].Length
        $key = $matches['key']
        $rest = if ($null -eq $matches['rest']) { '' } else { $matches['rest'].Trim() }

        if ($indent -eq 0) {
            $section = $key
            continue
        }
        if (($indent -ne 2) -or [string]::IsNullOrWhiteSpace($section) -or ($rest -eq '')) {
            continue
        }

        if ($rest.StartsWith('{') -and $rest.EndsWith('}')) {
            $body = $rest.Substring(1, $rest.Length - 2)
            foreach ($part in ($body -split ',')) {
                if ($part -notmatch '^\s*(?<innerKey>[A-Za-z0-9_-]+)\s*:\s*(?<innerValue>.*?)\s*$') {
                    throw "Malformed inline mapping under $section.$key"
                }
                $innerPath = "$section.$key.$($matches['innerKey'])"
                Add-ObservedValue -Table $observed -Path $innerPath -Value (Normalize-Value $matches['innerValue'])
            }
        }
        else {
            Add-ObservedValue -Table $observed -Path "$section.$key" -Value (Normalize-Value $rest)
        }
    }
}

$required = [ordered]@{
    'host_tools.go.selected_version' = '1.26.5'
    'host_tools.node.selected_version' = '24.20.0'
    'host_tools.pnpm.selected_version' = '11.24.0'
    'host_tools.python.selected_version' = '3.14.7'
    'host_tools.docker_engine.selected_version' = '29.7.2'
    'host_tools.docker_compose.selected_version' = '5.4.0'
    'backend_dependencies.chi' = '5.3.2'
    'backend_dependencies.pgx' = '5.10.0'
    'backend_dependencies.goose' = '3.27.3'
    'backend_dependencies.opentelemetry_go' = '1.46.0'
    'frontend_dependencies.react' = '19.2.8'
    'frontend_dependencies.typescript' = '7.0.2'
    'frontend_dependencies.vite' = '8.2.2'
    'frontend_dependencies.tailwind_css' = '4.3.3'
    'frontend_dependencies.motion' = '13.1.1'
    'frontend_dependencies.tanstack_query' = '5.102.8'
    'frontend_dependencies.react_hook_form' = '7.87.0'
    'frontend_dependencies.zod' = '4.5.4'
    'frontend_dependencies.i18next' = '26.4.1'
    'frontend_dependencies.react_i18next' = '17.0.13'
    'frontend_dependencies.vite_plugin_pwa' = '1.3.0'
    'frontend_dependencies.vitest' = '4.1.11'
    'local_services.postgresql' = '17.11'
    'local_services.postgis.selected_version' = '3.6.4'
    'local_services.meilisearch' = '1.51.0'
    'local_services.valkey' = '9.1.1'
    'local_services.seaweedfs' = '4.29'
    'local_services.nats_jetstream' = '2.14.5'
    'identity_verification.status' = 'PENDING_NO_NETWORK'
}

foreach ($pair in $required.GetEnumerator()) {
    if (-not $observed.ContainsKey($pair.Key)) {
        throw "Missing required toolchain key: $($pair.Key)"
    }
    $values = @($observed[$pair.Key])
    if ($values.Count -ne 1) {
        throw "Duplicate required toolchain key: $($pair.Key)"
    }
    if ($values[0] -cne [string]$pair.Value) {
        throw "Mismatched required toolchain value: $($pair.Key) expected '$($pair.Value)' observed '$($values[0])'"
    }
}

Write-Output 'V010_I009_TOOLCHAIN_STATIC_CHECK=PASS'
