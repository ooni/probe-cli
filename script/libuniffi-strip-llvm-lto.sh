#!/bin/bash
set -euo pipefail

# Workaround for a Go linker regression affecting Rust static libraries.
#
# Newer stable Rust toolchains may include LLVM bitcode sections such as
# .llvmbc and .llvmcmd in static libraries.
#
# These sections support cross-language link-time optimization (LTO),
# which this cgo build does not use. Go 1.27.4 may reject an archive
# containing them, reporting only one offending section at a time.
#
# Remove all section names beginning with ".llvm" from the Rust archives
# before the Go linker consumes them. This also makes the build less
# dependent on which Rust version the "stable" toolchain resolves to.
#
# Adapted from:
# https://github.com/Tencent/WeKnora/pull/5417/changes/d76e90f3d5fa01ae550c26881206b8be3d39e473

strip_llvm_lto_sections() {
  local lib="$1"

  # Without both tools, leave the archive unchanged. The Go linker may then
  # report the original problem, but this workaround will not block the build.
  command -v readelf >/dev/null 2>&1 &&
    command -v objcopy >/dev/null 2>&1 || return 0

  local sections section

  # List the archive's sections, keep names beginning with ".llvm", and
  # deduplicate them so each section name is removed only once.
  # readelf can return nonzero for some archive contents, so suppress its
  # diagnostics and tolerate grep finding no matches.
  sections=$(
    readelf -S "$lib" 2>/dev/null |
      grep -oE '\.llvm[a-zA-Z0-9_]+' |
      sort -u ||
      true
  )

  # Nothing to change if the archive contains no matching sections.
  [ -n "$sections" ] || return 0

  # Remove each matching section from the archive in place. A failed removal
  # is a warning rather than a fatal error; the Go linker may still reject
  # the archive.
  while IFS= read -r section; do
    [ -n "$section" ] || continue

    if ! objcopy --remove-section="$section" "$lib" 2>/dev/null; then
      echo "warning: could not strip $section from $lib; the Go linker may reject it" >&2
    fi
  done <<< "$sections"
}

# Find matching Rust static libraries under the user-authentication package.
# NUL-delimited paths handle filenames containing spaces or unusual characters.
while IFS= read -r -d '' file; do
  printf '\n==> Processing: %s\n' "$file"

  # Show the commands being run for this file, then turn tracing back off.
  set -x
  strip_llvm_lto_sections "$file"
  { set +x; } 2>/dev/null
done < <(find internal/userauth/lib/linux/ -type f -name 'libuniffi_ooniprobe.a' -print0)
