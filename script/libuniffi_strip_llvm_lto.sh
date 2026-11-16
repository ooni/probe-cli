#!/bin/bash
set -euo pipefail

# Code adapted from:
# https://github.com/Tencent/WeKnora/pull/5417/changes/d76e90f3d5fa01ae550c26881206b8be3d39e473

# Newer rustc stable embeds LLVM bitcode sections (.llvmbc, .llvmcmd, ...) in
# the staticlib. Go 1.27.4 refuses to link an archive that carries them, they
# are only useful for cross-language LTO, which this cgo link does not use,
# and it reports one section per run. Strip every .llvm* section before cgo
# consumes the archive, so the build does not depend on which rustc `stable`
# resolves to that day.
strip_llvm_lto_sections() {
  local lib="$1"
  command -v readelf >/dev/null 2>&1 && command -v objcopy >/dev/null 2>&1 || return 0
  local sections section
  sections=$(readelf -S "$lib" 2>/dev/null | grep -oE '\.llvm[a-zA-Z0-9_]+' | sort -u || true)
  [ -n "$sections" ] || return 0
  while IFS= read -r section; do
    [ -n "$section" ] || continue
    if ! objcopy --remove-section="$section" "$lib" 2>/dev/null; then
      echo "warning: could not strip $section from $lib; the Go linker may reject it" >&2
    fi
  done <<< "$sections"
}

while IFS= read -r -d '' file; do
  printf '\n==> Processing: %s\n' "$file"
  set -x
  strip_llvm_lto_sections "$file"
  { set +x; } 2>/dev/null
done < <(find internal/userauth/lib/ -type f -name 'libuniffi_ooniprobe.a' -print0)
