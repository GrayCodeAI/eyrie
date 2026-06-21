#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

violations="$(
  rg -n 'github\.com/GrayCodeAI/hawk/(internal/|shared/types)' \
    --glob '*.go' \
    . || true
)"

if [[ -n "${violations}" ]]; then
  echo "forbidden Hawk imports found:"
  echo "${violations}"
  echo
  echo "support repos must use hawk-core-contracts or local contracts, not hawk/internal or removed hawk/shared/types"
  exit 1
fi

echo "ecosystem boundary guard passed"
