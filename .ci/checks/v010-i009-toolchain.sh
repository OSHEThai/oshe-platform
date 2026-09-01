#!/usr/bin/env sh
set -eu
lock_path="$(dirname "$0")/../../toolchain.lock.yaml"
for value in 1.26.5 24.20.0 11.24.0 3.14.7 29.7.2 5.4.0 17.11 1.51.0 9.1.1 4.29 2.14.5 PENDING_NO_NETWORK; do
  grep -F "$value" "$lock_path" >/dev/null
done
if grep -E ':[[:space:]]*latest[[:space:]]*($|#)' "$lock_path" >/dev/null; then
  echo 'Floating latest alias is prohibited' >&2
  exit 1
fi
echo 'V010_I009_TOOLCHAIN_STATIC_CHECK=PASS'
