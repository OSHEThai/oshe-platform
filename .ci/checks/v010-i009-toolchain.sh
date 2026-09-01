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

if awk '
function trim(value) {
  sub(/^[[:space:]]+/, "", value)
  sub(/[[:space:]]+$/, "", value)
  return value
}
function normalize(value, first, last, double_quote, single_quote) {
  value = trim(value)
  double_quote = sprintf("%c", 34)
  single_quote = sprintf("%c", 39)
  first = substr(value, 1, 1)
  last = substr(value, length(value), 1)
  if ((first == double_quote && last == double_quote) ||
      (first == single_quote && last == single_quote)) {
    value = substr(value, 2, length(value) - 2)
  }
  return value
}
function is_mutable_alias(value) {
  value = tolower(normalize(value))
  return value == "latest" || value == "stable" || value == "edge" ||
         value == "rolling" || value == "canary" || value == "main" ||
         value == "master" || value == "dev" || value == "nightly"
}
function inspect_scalar(value, first, last, body, count, i, item, separator, pairs) {
  value = trim(value)
  first = substr(value, 1, 1)
  last = substr(value, length(value), 1)
  if (first == "{" && last == "}") {
    body = substr(value, 2, length(value) - 2)
    count = split(body, pairs, ",")
    for (i = 1; i <= count; i++) {
      item = trim(pairs[i])
      separator = index(item, ":")
      if (separator > 0) {
        inspect_scalar(substr(item, separator + 1))
      }
    }
    return
  }
  if (first == "[" && last == "]") {
    body = substr(value, 2, length(value) - 2)
    count = split(body, pairs, ",")
    for (i = 1; i <= count; i++) {
      inspect_scalar(pairs[i])
    }
    return
  }
  if (is_mutable_alias(value)) {
    found = normalize(value)
  }
}
{
  line = $0
  sub(/[[:space:]]+#.*/, "", line)
  if (line ~ /^[[:space:]]*$/) {
    next
  }
  if (line ~ /^[[:space:]]*(\-?[[:space:]]*)?[A-Za-z0-9_-]+:/) {
    match(line, /[A-Za-z0-9_-]+:/)
    inspect_scalar(substr(line, RSTART + RLENGTH))
    next
  }
  if (line ~ /^[[:space:]]*-[[:space:]]*.+$/) {
    sub(/^[[:space:]]*-[[:space:]]*/, "", line)
    inspect_scalar(line)
  }
}
END {
  if (found != "") {
    print "Mutable scalar alias is prohibited: " found > "/dev/stderr"
    exit 0
  }
  exit 1
}
' "$lock_path"; then
  exit 1
fi

if ! parsed="$(awk '
function trim(value) {
  sub(/^[[:space:]]+/, "", value)
  sub(/[[:space:]]+$/, "", value)
  return value
}
function normalize(value, first, last, double_quote, single_quote) {
  value = trim(value)
  double_quote = sprintf("%c", 34)
  single_quote = sprintf("%c", 39)
  first = substr(value, 1, 1)
  last = substr(value, length(value), 1)
  if ((first == double_quote && last == double_quote) ||
      (first == single_quote && last == single_quote)) {
    value = substr(value, 2, length(value) - 2)
  }
  return value
}
function emit(path, value) {
  print path "|" normalize(value)
}
{
  line = $0
  sub(/[[:space:]]+#.*/, "", line)
  if (line ~ /^[[:space:]]*$/) {
    next
  }
  if (line ~ /^[A-Za-z0-9_-]+:/) {
    split(line, top, ":")
    section = top[1]
    next
  }
  if (line ~ /^  [A-Za-z0-9_-]+:/) {
    line = substr(line, 3)
    separator = index(line, ":")
    if (separator == 0) {
      print "Malformed structural YAML" > "/dev/stderr"
      exit 2
    }
    key = trim(substr(line, 1, separator - 1))
    rest = trim(substr(line, separator + 1))
    if (rest == "") {
      next
    }
    if (rest ~ /^\{.*\}$/) {
      body = substr(rest, 2, length(rest) - 2)
      pair_count = split(body, pairs, ",")
      for (i = 1; i <= pair_count; i++) {
        pair = trim(pairs[i])
        inner_separator = index(pair, ":")
        if (inner_separator == 0) {
          print "Malformed inline mapping under " section "." key > "/dev/stderr"
          exit 2
        }
        inner_key = trim(substr(pair, 1, inner_separator - 1))
        inner_value = trim(substr(pair, inner_separator + 1))
        emit(section "." key "." inner_key, inner_value)
      }
    }
    else {
      emit(section "." key, rest)
    }
  }
}
' "$lock_path")"; then
  echo 'Malformed structural YAML' >&2
  exit 1
fi

while IFS='|' read -r path expected; do
  [ -n "$path" ] || continue
  match_count="$(printf '%s\n' "$parsed" | awk -F '|' -v expected_path="$path" '$1 == expected_path { count += 1 } END { print count + 0 }')"
  if [ "$match_count" -eq 0 ]; then
    echo "Missing required toolchain key: $path" >&2
    exit 1
  fi
  if [ "$match_count" -ne 1 ]; then
    echo "Duplicate required toolchain key: $path" >&2
    exit 1
  fi
  actual="$(printf '%s\n' "$parsed" | awk -F '|' -v expected_path="$path" '$1 == expected_path { print $2; exit }')"
  if [ "$actual" != "$expected" ]; then
    echo "Mismatched required toolchain value: $path expected '$expected' observed '$actual'" >&2
    exit 1
  fi
done <<'REQUIRED_PAIRS'
host_tools.go.selected_version|1.26.5
host_tools.node.selected_version|24.20.0
host_tools.pnpm.selected_version|11.24.0
host_tools.python.selected_version|3.14.7
host_tools.docker_engine.selected_version|29.7.2
host_tools.docker_compose.selected_version|5.4.0
backend_dependencies.chi|5.3.2
backend_dependencies.pgx|5.10.0
backend_dependencies.goose|3.27.3
backend_dependencies.opentelemetry_go|1.46.0
frontend_dependencies.react|19.2.8
frontend_dependencies.typescript|7.0.2
frontend_dependencies.vite|8.2.2
frontend_dependencies.tailwind_css|4.3.3
frontend_dependencies.motion|13.1.1
frontend_dependencies.tanstack_query|5.102.8
frontend_dependencies.react_hook_form|7.87.0
frontend_dependencies.zod|4.5.4
frontend_dependencies.i18next|26.4.1
frontend_dependencies.react_i18next|17.0.13
frontend_dependencies.vite_plugin_pwa|1.3.0
frontend_dependencies.vitest|4.1.11
local_services.postgresql|17.11
local_services.postgis.selected_version|3.6.4
local_services.meilisearch|1.51.0
local_services.valkey|9.1.1
local_services.seaweedfs|4.29
local_services.nats_jetstream|2.14.5
identity_verification.status|PENDING_NO_NETWORK
REQUIRED_PAIRS

echo 'V010_I009_TOOLCHAIN_STATIC_CHECK=PASS'
