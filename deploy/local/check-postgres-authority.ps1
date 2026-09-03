# PostgreSQL-authority degradation check (static).
# Verifies the local stack encodes PostgreSQL as AUTHORITATIVE and Meilisearch/Valkey
# as REBUILDABLE non-authoritative projections/cache, per ADR-0006.
$ErrorActionPreference = 'Stop'
$compose = Join-Path $PSScriptRoot 'compose.dev.yaml'
$text = Get-Content -LiteralPath $compose -Raw
if ($text -notmatch 'PostgreSQL is the AUTHORITATIVE transactional store') {
    throw 'AUTHORITY_VIOLATION: PostgreSQL authoritative boundary not declared'
}
if ($text -notmatch 'Meilisearch holds REBUILDABLE search projections only') {
    throw 'AUTHORITY_VIOLATION: Meilisearch rebuildable-projection boundary not declared'
}
if ($text -notmatch 'Valkey is a REBUILDABLE, non-authoritative cache') {
    throw 'AUTHORITY_VIOLATION: Valkey non-authoritative boundary not declared'
}
if ($text -notmatch 'NATS JetStream is messaging used only AFTER the transactional outbox') {
    throw 'AUTHORITY_VIOLATION: NATS after-outbox boundary not declared'
}
Write-Host 'OSHE_POSTGRES_AUTHORITY=PASS (PostgreSQL authoritative; Meilisearch/Valkey rebuildable; NATS after-outbox)'
