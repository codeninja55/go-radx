#!/usr/bin/env bash
#
# Pin-drift gate: fail if any reference tool or interop image has floated back to an unpinned
# reference. The interop and conformance gates are only reproducible if every external input resolves
# to fixed bytes; this check guards that invariant so a `:latest` image tag, an `@latest` install, or
# an unpinned `pip`/`apt-get` install of a reference tool cannot slip back in unnoticed.
#
# It scans the files that actually pull external tools and images:
#   - .github/workflows/ci.yml      (apt/pip/go installs)
#   - mise.toml                     (the [tools] pins)
#   - the testcontainers helpers    (interop image references)
# and flags:
#   - any `:latest` (or other floating tag) on a container image reference;
#   - any `@latest` on a `go install` / VCS install;
#   - any `pip install` / `apt-get install` of a known reference tool without a version pin.
#
# The pinned set of record is tools/versions; this script enforces the *shape* (a pin is present),
# not the exact value — bumping a pin is a deliberate, reviewed change recorded there.
#
# Exit 0 when every reference is pinned; exit 1 (printing each offender) on any drift.

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

CI_YML=".github/workflows/ci.yml"
MISE_TOML="mise.toml"
# Testcontainers helpers that reference interop images.
HELPER_GLOBS=(
  "dimse/integration/orthanc/orthanc.go"
  "dicomweb/integration/orthanc/orthanc.go"
  "dimse/integration/dcm4chee/dcm4chee.go"
)

violations=0

# report records one drift finding: a file, a line number, and why it failed.
report() {
  echo "DRIFT: $1:$2: $3" >&2
  violations=$((violations + 1))
}

# is_comment is true for a shell/YAML/Go comment line — one whose first non-blank character starts a
# comment. Such lines describe the pinning policy in prose (e.g. "@latest would let it float") and
# must not be flagged as live unpinned references.
is_comment() {
  case "$(echo "$1" | sed 's/^[[:space:]]*//')" in
    \#* | //*) return 0 ;;
    *) return 1 ;;
  esac
}

# scan_floating_image flags container image references that end in a floating tag with no digest.
# A pinned reference carries an `@sha256:` digest; `image:tag` without a digest, or the bare `:latest`
# tag, is drift. The grep matches a `name:tag` token in a quoted string; lines that already carry
# `@sha256:` are exempt, and comment lines (which quote an image in prose, e.g. a re-resolution hint)
# are skipped.
scan_floating_image() {
  local file="$1"
  [ -f "$file" ] || return 0
  while IFS=: read -r lineno line; do
    [ -z "${lineno:-}" ] && continue
    is_comment "$line" && continue
    # Skip lines that pin by digest (the desired state).
    case "$line" in
      *"@sha256:"*) continue ;;
    esac
    report "$file" "$lineno" "image reference without an @sha256: digest pin -> $(echo "$line" | sed 's/^[[:space:]]*//')"
  done < <(grep -nE '"[A-Za-z0-9._/-]+/[A-Za-z0-9._-]+:[A-Za-z0-9._-]+"' "$file" 2>/dev/null \
            | grep -E 'orthancteam/|dcm4che/' || true)
}

# scan_latest_tag flags an explicit `:latest` image tag anywhere in a file (defence in depth: catches
# a `:latest` even on a registry/name shape the image grep above does not match). Comment lines that
# merely mention `:latest` in prose are skipped.
scan_latest_tag() {
  local file="$1"
  [ -f "$file" ] || return 0
  while IFS=: read -r lineno line; do
    [ -z "${lineno:-}" ] && continue
    is_comment "$line" && continue
    report "$file" "$lineno" "':latest' floating tag -> $(echo "$line" | sed 's/^[[:space:]]*//')"
  done < <(grep -nE ':latest("|/|[[:space:]]|$)' "$file" 2>/dev/null || true)
}

# scan_at_latest flags `@latest` on a `go install` or any VCS-style install. It requires `@latest` to
# be attached to a module/package path (a `/`-bearing token), so a comment that merely mentions the
# word `@latest` in prose is not flagged — only a live `…/pkg@latest` install is.
scan_at_latest() {
  local file="$1"
  [ -f "$file" ] || return 0
  while IFS=: read -r lineno line; do
    [ -z "${lineno:-}" ] && continue
    is_comment "$line" && continue
    report "$file" "$lineno" "'@latest' install (pin to a released version) -> $(echo "$line" | sed 's/^[[:space:]]*//')"
  done < <(grep -nE '[A-Za-z0-9._/-]+/[A-Za-z0-9._-]+@latest([[:space:]"'\'']|$)' "$file" 2>/dev/null || true)
}

# scan_mise_latest flags a mise [tools] entry pinned to a floating version — `tool = "latest"` or the
# unversioned forms (`"stable"`, `"prefix:…"` aside, a bare `"latest"`). mise.toml [tools] entries
# pin a version string; "latest" there floats just like a `:latest` image tag.
scan_mise_latest() {
  local file="$1"
  [ -f "$file" ] || return 0
  while IFS=: read -r lineno line; do
    [ -z "${lineno:-}" ] && continue
    is_comment "$line" && continue
    report "$file" "$lineno" "mise tool pinned to floating \"latest\" (pin an exact version) -> $(echo "$line" | sed 's/^[[:space:]]*//')"
  done < <(grep -nE '=[[:space:]]*"latest"' "$file" 2>/dev/null || true)
}

# scan_unpinned_pip flags `pip install` of a known reference tool unless it carries an exact `==`
# version pin. Only `tool==X` is accepted; any looser specifier (`tool`, `tool~=X`, `tool>=X`) still
# floats and is flagged. The grep matches a `pip install` line naming the tool; the case filter then
# exempts the one pinned shape. Prose mentioning the tool elsewhere is not on a `pip install` line.
scan_unpinned_pip() {
  local file="$1"
  [ -f "$file" ] || return 0
  local tool
  for tool in pydicom pynetdicom hl7 mkdocs mkdocs-material; do
    while IFS=: read -r lineno line; do
      [ -z "${lineno:-}" ] && continue
      is_comment "$line" && continue
      # Exempt only the exact pin `tool==`. A bare name or a non-exact specifier (~=, >=, <=, >, <)
      # is drift. Match `tool==` only when `==` immediately follows the tool name (not `tool>=2,==3`
      # style trickery, which is rejected as not the exact pinned shape).
      case "$line" in
        *"${tool}=="*) continue ;;
      esac
      report "$file" "$lineno" "pip install of '$tool' without an exact '==' version pin -> $(echo "$line" | sed 's/^[[:space:]]*//')"
    done < <(grep -nE "pip[[:space:]]+install.*[\"' ]?${tool}([\"' =~<>]|$)" "$file" 2>/dev/null || true)
  done
}

# scan_unpinned_apt flags `apt-get install` of a known reference tool unless it carries an exact
# `=version` pin. A bare name or the target-release form (`tool/noble`) still floats and is flagged.
scan_unpinned_apt() {
  local file="$1"
  [ -f "$file" ] || return 0
  local tool
  for tool in dicom3tools; do
    while IFS=: read -r lineno line; do
      [ -z "${lineno:-}" ] && continue
      is_comment "$line" && continue
      # Exempt only the exact pin `tool=version`. The target-release form `tool/suite` is NOT a
      # version pin (it selects a suite, whose contents still move), so it is not exempted.
      case "$line" in
        *"${tool}="*) continue ;;
      esac
      report "$file" "$lineno" "apt-get install of '$tool' without an exact '=version' pin -> $(echo "$line" | sed 's/^[[:space:]]*//')"
    done < <(grep -nE "apt-get[[:space:]]+install.*[\"' ]?${tool}([\"' =/]|$)" "$file" 2>/dev/null || true)
  done
}

for helper in "${HELPER_GLOBS[@]}"; do
  scan_floating_image "$helper"
  scan_latest_tag "$helper"
done

scan_latest_tag "$CI_YML"
scan_at_latest "$CI_YML"
scan_unpinned_pip "$CI_YML"
scan_unpinned_apt "$CI_YML"

scan_latest_tag "$MISE_TOML"
scan_at_latest "$MISE_TOML"
scan_mise_latest "$MISE_TOML"

if [ "$violations" -gt 0 ]; then
  echo "FAIL: $violations unpinned reference(s) found. Pin each (digest preferred, exact version" \
       "acceptable) and record it in tools/versions." >&2
  exit 1
fi

echo "OK: all scanned reference tools and interop images are pinned."
