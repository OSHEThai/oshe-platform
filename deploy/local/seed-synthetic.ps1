# Synthetic-only seed. No production or customer data is ever loaded.
$ErrorActionPreference = 'Stop'
Write-Host 'Synthetic seed boundary: OSHE_LOCAL_SYNTHETIC_SEED=READY (no production/customer data).'
# The deterministic synthetic seed data set is applied by the application bootstrap;
# this script records the synthetic-only boundary for the local stack.
