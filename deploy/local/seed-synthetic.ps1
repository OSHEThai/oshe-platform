# Synthetic-only deterministic seed. No production or customer data is ever loaded.
# Determinism: fixed fictional rows and identifiers only; no wall-clock time,
# randomness, GUIDs, or host/user-derived values.
$ErrorActionPreference = 'Stop'

$syntheticSeed = @(
    [pscustomobject]@{ tenant_id = 'synth-0001'; name = 'Synthetic Tenant One'; locale = 'th-TH' },
    [pscustomobject]@{ tenant_id = 'synth-0002'; name = 'Synthetic Tenant Two'; locale = 'en-US' }
)

foreach ($row in $syntheticSeed) {
    Write-Host ("OSHE_SYNTHETIC_SEED_ROW=" + ($row | ConvertTo-Json -Compress))
}
Write-Host 'OSHE_LOCAL_SYNTHETIC_SEED=READY'
