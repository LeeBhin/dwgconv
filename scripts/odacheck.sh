#!/usr/bin/env bash
# odacheck.sh <dir-a> <dir-b>: convert every DWG in both dirs to DXF with ODA (audit on)
# and report differing DXF lines per file, ignoring save timestamps ($TDUPDATE/$TDUUPDATE).
set -euo pipefail
ODA="${ODA:-/c/Users/bin/AppData/Local/ODA-extract/ODAFileConverter.exe}"
a="$1"; b="$2"
work="$(mktemp -d)"
mkdir -p "$work/a" "$work/b"
# DXF as code|value pairs, dropping the save timestamps
norm() { tr -d '\r' < "$1" | awk 'NR%2==1{c=$0; next} skip{skip=0; next} /^\$TDU?UPDATE$/{skip=1; next} {print c"|"$0}'; }
"$ODA" "$(cygpath -w "$a")" "$(cygpath -w "$work/a")" ACAD2018 DXF 0 1 "*.DWG"
"$ODA" "$(cygpath -w "$b")" "$(cygpath -w "$work/b")" ACAD2018 DXF 0 1 "*.DWG"
for f in "$work/a"/*.dxf; do
  n="$(basename "$f")"
  if [ ! -f "$work/b/$n" ]; then echo "MISSING  $n"; continue; fi
  d=$(diff <(norm "$f") <(norm "$work/b/$n") | grep -c '^[<>]' || true)
  echo "diff=$d  $n"
done
echo "dxf in $work"
