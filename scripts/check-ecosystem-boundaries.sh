#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if command -v rg >/dev/null 2>&1; then
  violations="$(rg -n 'github\.com/GrayCodeAI/hawk/(internal/|shared/types)' --glob '*.go' . || true)"
else
  violations="$(grep -rn --include='*.go' -E 'github\.com/GrayCodeAI/hawk/(internal/|shared/types)' . || true)"
fi

if [[ -n "${violations}" ]]; then
  echo "forbidden Hawk imports found:"
  echo "${violations}"
  echo
  echo "support repos must use hawk-core-contracts or local contracts, not hawk/internal or removed hawk/shared/types"
  exit 1
fi

echo "ecosystem boundary guard passed"
