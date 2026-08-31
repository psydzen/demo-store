#!/usr/bin/env bash
# Repository-wide Semgrep run with timing and peak memory.
# Usage: scripts/run_semgrep_repo.sh <out-dir> [--pro]
set -euo pipefail

out="$1"; shift
mkdir -p "$out"

pro=()
label="oss"
if [[ "${1:-}" == "--pro" ]]; then pro=(--pro); label="pro"; fi

for i in 1 2 3; do
  /usr/bin/time -l semgrep "${pro[@]+"${pro[@]}"}" \
      --metrics=off --quiet \
      --config rules/semgrep \
      --exclude internal/analysisfixtures \
      --json --output "$out/semgrep-$label.json" \
      . \
    2> "$out/semgrep-$label.time.$i"
done
