#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="${ARTICLEFLOW_ROOT:-$(pwd)}"

if [ ! -f "${PROJECT_ROOT}/go.work" ]; then
  echo "scripts/env.sh must be sourced from the articleflow project root or with ARTICLEFLOW_ROOT set" >&2
  return 1 2>/dev/null || exit 1
fi

export GOPATH="${PROJECT_ROOT}/.go"
export GOMODCACHE="${PROJECT_ROOT}/.go/pkg/mod"
export GOCACHE="${PROJECT_ROOT}/.cache/go-build"
export GOBIN="${PROJECT_ROOT}/.bin"
export PATH="${GOBIN}:${PATH}"

mkdir -p "${GOPATH}" "${GOMODCACHE}" "${GOCACHE}" "${GOBIN}"
