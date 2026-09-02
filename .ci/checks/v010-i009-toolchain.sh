#!/usr/bin/env sh
set -eu

if [ "$#" -gt 1 ]; then
  echo "Usage: $0 [lock-path]" >&2
  exit 2
fi
if [ "$#" -eq 1 ]; then
  lock_path=$1
else
  lock_path="$(dirname "$0")/../../toolchain.lock.yaml"
fi
if [ ! -f "$lock_path" ]; then
  echo "Toolchain lock was not found: $lock_path" >&2
  exit 1
fi

python - "$lock_path" <<'PY'
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
  postgresql: 17.11
  postgis: {selected_version: 3.6.4, optional_when_spatial_features_enabled: true}
  meilisearch: 1.51.0
  valkey: 9.1.1
  seaweedfs: 4.29
  nats_jetstream: 2.14.5
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
PY
