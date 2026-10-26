#!/usr/bin/env bash
# rt.sh <sN> [skip-types]: ODA DXF -> our DWG -> ODA read; prints ODA's error log.
set -euo pipefail
cd "$(dirname "$0")/.."
n="$1"
rm -rf .work/rt/$n .work/rt/${n}_out && mkdir -p .work/rt/$n .work/rt/${n}_out
MSYS_NO_PATHCONV=1 docker run --rm -e DWGCONV_SKIP="${2:-}" -v "$(pwd -W):/src" -v dwgconv-gocache:/root/.cache -v dwgconv-gomod:/go/pkg/mod -w /src golang:1.27 \
  go run ./cmd/dwgconv .work/oda/$n.dxf .work/rt/$n/$n.dwg
"/c/Users/bin/AppData/Local/ODA-extract/ODAFileConverter.exe" "$(cygpath -w .work/rt/$n)" "$(cygpath -w .work/rt/${n}_out)" ACAD2018 DXF 0 1 "*.DWG"
if [ -f .work/rt/${n}_out/$n.dxf.err ]; then cat .work/rt/${n}_out/$n.dxf.err; else echo "ODA read OK: .work/rt/${n}_out/$n.dxf"; fi
