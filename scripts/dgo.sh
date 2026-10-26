#!/usr/bin/env bash
# dgo.sh <go args...>: run the Go toolchain in a Linux container on this repo.
# Windows Smart App Control blocks freshly built unsigned .exe files on this machine.
# Usage from the repo root: scripts/dgo.sh run ./cmd/dxfcheck .work/orig/s2.dwg .work/oda/s2.dxf
set -euo pipefail
repo="$(cd "$(dirname "$0")/.." && pwd -W 2>/dev/null || pwd)"
MSYS_NO_PATHCONV=1 exec docker run --rm -i \
  -v "$repo:/src" -v dwgconv-gocache:/root/.cache -v dwgconv-gomod:/go/pkg/mod \
  -w /src golang:1.27 go "$@"
