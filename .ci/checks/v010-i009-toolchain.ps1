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

$python = (Get-Command python -ErrorAction Stop).Source
$pythonCode = @'
import re
import sys
from pathlib import Path

import yaml
from yaml.nodes import MappingNode


BASELINE_YAML = r'''schema_version: 1.0.0
lock_id: V010-I009-TOOLCHAIN-LOCK-001
lifecycle_status: DRAFT_SELECTED_BASELINE_PENDING_IDENTITY_VERIFICATION
authority:
  decision_ref: HDEC-V010-I009-H010-003
  source_adr: Plan/10 Engineering and GitHub/06 ADR RFC API Event Schema and Database Migration Governance/05 Architecture Decision Records/ADR-0006 - Go Search First and Locale Configurable Platform Stack.md
  no_provider_network_envelope: HDEC-NO-SPEND-DISPATCH-013
host_tools:
  go: {selected_version: 1.26.5, observed_local_version: 1.26.5}
  node: {selected_version: 24.20.0, observed_local_version: 24.20.0}
  pnpm: {selected_version: 11.24.0, observed_local_version: 11.24.0}
  python: {selected_version: 3.14.7, observed_local_version: 3.14.7}
  docker_engine: {selected_version: 29.7.2, observed_local_version: UNVERIFIED_NO_NETWORK}
  docker_compose: {selected_version: 5.4.0, observed_local_version: UNVERIFIED_NO_NETWORK}
backend_dependencies:
  chi: 5.3.2
  pgx: 5.10.0
  goose: 3.27.3
  opentelemetry_go: 1.46.0
frontend_dependencies:
  react: 19.2.8
  typescript: 7.0.2
  vite: 8.2.2
  tailwind_css: 4.3.3
  motion: 13.1.1
  tanstack_query: 5.102.8
  react_hook_form: 7.87.0
  zod: 4.5.4
  i18next: 26.4.1
  react_i18next: 17.0.13
  vite_plugin_pwa: 1.3.0
  vitest: 4.1.11
local_services:
  postgresql:
    selected_version: 17.11
    image_ref: postgres:17.11-alpine@sha256:18cfe3ef5e6815560c98237d6216d1e5119702fb0f3894c8785dd58b8bbe5d73
    platform: linux/amd64
    platform_manifest_digest: sha256:7456ef82e5f5bc43d997f4781bbd7c0d6389bff397564649a356e206ba473aee
    index_digest: sha256:18cfe3ef5e6815560c98237d6216d1e5119702fb0f3894c8785dd58b8bbe5d73
  postgis:
    selected_version: 3.6.4
    optional_when_spatial_features_enabled: true
    image_ref: postgis/postgis:17-3.6-alpine@sha256:a8ffa9afeea4ad6eada171fa2afdb57cd3eb90f92ce20156aa2cb8411d70e0cd
    platform: linux/amd64
    platform_manifest_digest: sha256:7ce143dbc804dc08a8f1dcf9067724f9b6e4ded48711e9d884487967acb442b3
    index_digest: sha256:a8ffa9afeea4ad6eada171fa2afdb57cd3eb90f92ce20156aa2cb8411d70e0cd
  meilisearch:
    selected_version: 1.51.0
    image_ref: getmeili/meilisearch:v1.51.0@sha256:a9eb29ee09ab4943db3b4c68620bd6f3382e6b2b0ac4431c0e607b48dbcd4c14
    platform: linux/amd64
    platform_manifest_digest: sha256:a215118cfab8591dfb9e9b82e9c45f751d324bb71be2581ce2af344f50e5a52c
    index_digest: sha256:a9eb29ee09ab4943db3b4c68620bd6f3382e6b2b0ac4431c0e607b48dbcd4c14
  valkey:
    selected_version: 9.1.1
    image_ref: valkey/valkey:9.1.1-alpine@sha256:15568b9cb7eb67f4aed4de018c23f13d344e0e6437b31fe8fb8823dc81ebb3a9
    platform: linux/amd64
    platform_manifest_digest: sha256:17539e366539969d44eb7a217237cd3ccd1a7755cdb71c7ea69be06cab2e6b9c
    index_digest: sha256:15568b9cb7eb67f4aed4de018c23f13d344e0e6437b31fe8fb8823dc81ebb3a9
  seaweedfs:
    selected_version: 4.29
    image_ref: chrislusf/seaweedfs:4.29@sha256:d47c7ee99fcb951351d7194915f4e3a5ea604a8e8871183d713907dec4fb9bf5
    platform: linux/amd64
    platform_manifest_digest: sha256:f16591b02e7a1d79dca57801405eec2c784711436edf65c0aa6394ef52800a3e
    index_digest: sha256:d47c7ee99fcb951351d7194915f4e3a5ea604a8e8871183d713907dec4fb9bf5
  nats_jetstream:
    selected_version: 2.14.5
    image_ref: nats:2.14.5-alpine@sha256:d4ac35882ac65aff236cd65b9d3fa4d24332c681e1a85f94eedccd3cdd65b1da
    platform: linux/amd64
    platform_manifest_digest: sha256:bacc5a40233588bd10201a2576903d9bfcc3bf84261ed324e522df2666e5eefd
    index_digest: sha256:d4ac35882ac65aff236cd65b9d3fa4d24332c681e1a85f94eedccd3cdd65b1da
identity_verification:
  status: PENDING_NO_NETWORK
  required_before_clean_install_or_container_claim:
  - official acquisition provenance
  - package checksum_or_integrity
  - OCI_image_digest
  - action_commit_sha
  - Windows_Linux_resolution_evidence
  prohibited: [latest_alias, unpinned_image, unverified_package_lock]
'''

ALIASES = frozenset({'latest', 'stable', 'edge', 'rolling', 'canary', 'main', 'master', 'dev', 'nightly'})
CONTAINER_TAG_ALIAS = re.compile(r'^[^\s/]+(?:/[^\s/]+)+:(?:latest|stable|edge|rolling|canary|main|master|dev|nightly)$', re.IGNORECASE)
BARE_CONTAINER_REFERENCE = re.compile(r'^[^\s/@]+(?:/[^\s/@]+)+$')
VERSION_RANGE = re.compile(r'^(?:>=|\^|~)\d+\.\d+\.\d+$|^\d+\.x$|^\*$|^\d+\.\*$|^\d+\.\d+\.(?:\*|x)$', re.IGNORECASE)
SINGLE_COMPONENT_OCI_REFERENCE = re.compile(r'^[a-z0-9][a-z0-9._-]*$', re.IGNORECASE)
TAGGED_SINGLE_COMPONENT_OCI_REFERENCE = re.compile(r'^[a-z0-9][a-z0-9._-]*:[^\s/@]+$', re.IGNORECASE)
OCI_REFERENCE_FIELD_NAMES = frozenset({
    'image', 'image_ref', 'oci_image', 'oci_reference', 'container_image', 'container_reference',
})
FIXED_VALUE_ALLOWLIST = {
    'local_services.seaweedfs': frozenset({'4.29'}),
}
DIGEST_PINNED_OCI_PATHS = frozenset({
    'local_services.postgresql.image_ref',
    'local_services.postgis.image_ref',
    'local_services.meilisearch.image_ref',
    'local_services.valkey.image_ref',
    'local_services.seaweedfs.image_ref',
    'local_services.nats_jetstream.image_ref',
})
PLATFORM_PATHS = frozenset({
    'local_services.postgresql.platform',
    'local_services.postgis.platform',
    'local_services.meilisearch.platform',
    'local_services.valkey.platform',
    'local_services.seaweedfs.platform',
    'local_services.nats_jetstream.platform',
})
UNVERIFIED = 'UNVERIFIED_NO_NETWORK'
UNVERIFIED_PATHS = frozenset({
    'host_tools.docker_engine.observed_local_version',
    'host_tools.docker_compose.observed_local_version',
})
PENDING = 'PENDING_NO_NETWORK'
PENDING_PATH = 'identity_verification.status'


class ContractError(Exception):
    pass


class StrictLoader(yaml.SafeLoader):
    pass


def construct_mapping(loader, node, deep=False):
    if not isinstance(node, MappingNode):
        raise ContractError('YAML_PARSE_ERROR: mapping node expected')
    mapping = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        try:
            duplicate = key in mapping
        except TypeError as exc:
            raise ContractError('YAML_PARSE_ERROR: unhashable mapping key') from exc
        if duplicate:
            raise ContractError(f'DUPLICATE_MAPPING_KEY: {key!r}')
        mapping[key] = loader.construct_object(value_node, deep=deep)
    return mapping


StrictLoader.add_constructor(yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, construct_mapping)


def load_yaml(text, label):
    try:
        return yaml.load(text, Loader=StrictLoader)
    except ContractError:
        raise
    except yaml.YAMLError as exc:
        detail = getattr(exc, 'problem', None) or str(exc)
        raise ContractError(f'YAML_PARSE_ERROR: {label}: {detail}') from exc


def type_name(value):
    if value is None:
        return 'null'
    if isinstance(value, bool):
        return 'bool'
    if isinstance(value, int):
        return 'int'
    if isinstance(value, float):
        return 'float'
    if isinstance(value, str):
        return 'string'
    if isinstance(value, list):
        return 'list'
    if isinstance(value, dict):
        return 'mapping'
    return type(value).__name__


def key_sort(key):
    return type_name(key), repr(key)


def child_path(path, key):
    return f'{path}.{key}' if path else str(key)


def is_oci_reference_path(path):
    field_name = re.sub(r'\[\d+\]$', '', path.rsplit('.', 1)[-1])
    return field_name in OCI_REFERENCE_FIELD_NAMES


def inspect_scalars(value, path=''):
    if isinstance(value, dict):
        for key in sorted(value, key=key_sort):
            inspect_scalars(value[key], child_path(path, key))
        return
    if isinstance(value, list):
        for index, item in enumerate(value):
            inspect_scalars(item, f'{path}[{index}]')
        return
    if not isinstance(value, str):
        return
    if path in PLATFORM_PATHS:
        return
    if path in DIGEST_PINNED_OCI_PATHS and '@sha256:' not in value:
        raise ContractError(f'MUTABLE_TAGGED_IMAGE: {path}: {value}')
    if value in FIXED_VALUE_ALLOWLIST.get(path, frozenset()):
        return
    folded = value.casefold()
    if folded in ALIASES:
        raise ContractError(f'MUTABLE_SCALAR_ALIAS: {path}: {value}')
    if CONTAINER_TAG_ALIAS.fullmatch(value):
        raise ContractError(f'MUTABLE_CONTAINER_TAG_ALIAS: {path}: {value}')
    if is_oci_reference_path(path) and TAGGED_SINGLE_COMPONENT_OCI_REFERENCE.fullmatch(value):
        raise ContractError(f'UNPINNED_TAGGED_SINGLE_COMPONENT_OCI_REFERENCE: {path}: {value}')
    if is_oci_reference_path(path) and SINGLE_COMPONENT_OCI_REFERENCE.fullmatch(value):
        raise ContractError(f'UNPINNED_SINGLE_COMPONENT_OCI_REFERENCE: {path}: {value}')
    if '://' not in value and BARE_CONTAINER_REFERENCE.fullmatch(value):
        raise ContractError(f'GENERIC_BARE_REFERENCE: {path}: {value}')
    if VERSION_RANGE.fullmatch(value):
        raise ContractError(f'GENERIC_VERSION_RANGE: {path}: {value}')
    if value == UNVERIFIED and path not in UNVERIFIED_PATHS:
        raise ContractError(f'MISPLACED_UNVERIFIED_NO_NETWORK: {path}')
    if value == PENDING and path != PENDING_PATH:
        raise ContractError(f'MISPLACED_PENDING_NO_NETWORK: {path}')


def compare(expected, actual, path='$'):
    if type(expected) is not type(actual):
        raise ContractError(
            f'BASELINE_MISMATCH: {path}: expected type {type_name(expected)}, got {type_name(actual)}'
        )
    if isinstance(expected, dict):
        for key in sorted(expected, key=key_sort):
            if key not in actual:
                raise ContractError(f'BASELINE_MISMATCH: {child_path(path, key)}: missing key')
            compare(expected[key], actual[key], child_path(path, key))
        for key in sorted(actual, key=key_sort):
            if key not in expected:
                raise ContractError(f'BASELINE_MISMATCH: {child_path(path, key)}: unexpected key')
        return
    if isinstance(expected, list):
        if len(expected) != len(actual):
            raise ContractError(
                f'BASELINE_MISMATCH: {path}: expected list length {len(expected)}, got {len(actual)}'
            )
        for index, (expected_item, actual_item) in enumerate(zip(expected, actual)):
            compare(expected_item, actual_item, f'{path}[{index}]')
        return
    if expected != actual:
        raise ContractError(f'BASELINE_MISMATCH: {path}: expected {expected!r}, got {actual!r}')


def main():
    if len(sys.argv) != 2:
        raise ContractError('USAGE: expected exactly one lock path')
    lock_path = Path(sys.argv[1])
    try:
        actual_text = lock_path.read_text(encoding='utf-8')
    except OSError as exc:
        raise ContractError(f'LOCK_READ_ERROR: {lock_path}: {exc}') from exc
    expected = load_yaml(BASELINE_YAML, 'known baseline')
    actual = load_yaml(actual_text, str(lock_path))
    inspect_scalars(actual)
    compare(expected, actual)
    print('V010_I009_TOOLCHAIN_STATIC_CHECK=PASS')


try:
    main()
except ContractError as exc:
    print(str(exc), file=sys.stderr)
    sys.exit(1)
except Exception as exc:
    print(f'CHECKER_ERROR: {exc}', file=sys.stderr)
    sys.exit(1)
'@

& $python -c $pythonCode $LockPath
$code = $LASTEXITCODE
exit $code
