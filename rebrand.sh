#!/usr/bin/env bash
#
# rebrand.sh — rename this project from "gxt" to "ichi".
#
# Handles, mechanically:
#   - Go module path  github.com/atterpac/gxt -> github.com/atterpac/ichi
#   - all import paths referencing that module
#   - binary directory/file  cmd/gxt/gxt.go -> cmd/ichi/ichi.go
#   - config/cache dir name  "gxt" -> "ichi"
#   - temp-file prefixes, window title, command descriptions
#   - logo variable name  gxtLogo -> ichiLogo
#   - doc references in *.md
#
# It does NOT redraw the ASCII-art logo (it spells "GXT" in block glyphs).
# Replace that art by hand afterward — the script prints a reminder.
#
# Usage:
#   ./rebrand.sh            # apply changes
#   ./rebrand.sh --dry-run  # show what would change, touch nothing

set -euo pipefail

OLD="gxt"
NEW="ichi"
OLD_MOD="github.com/atterpac/${OLD}"
NEW_MOD="github.com/atterpac/${NEW}"

DRY_RUN=0
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=1

cd "$(dirname "$0")"

# Sanity: must be the project root.
if [[ ! -f go.mod ]] || ! grep -q "^module ${OLD_MOD}$" go.mod; then
  echo "error: run from the ${OLD} project root (go.mod with module ${OLD_MOD} not found)" >&2
  exit 1
fi

# Files to rewrite content in. Exclude VCS, build output, vendored deps, and this script.
mapfile -t FILES < <(git ls-files '*.go' '*.mod' '*.sum' '*.md' '*.json' '*.yaml' '*.yml' '*.toml' \
  | grep -v '^rebrand\.sh$' || true)

run() {
  if [[ $DRY_RUN -eq 1 ]]; then
    echo "+ $*"
  else
    "$@"
  fi
}

echo "==> Rewriting references in ${#FILES[@]} tracked files"
for f in "${FILES[@]}"; do
  [[ -f "$f" ]] || continue
  # Quick skip if the file mentions neither the module nor the bare token.
  grep -qiE "${OLD}" "$f" || continue

  if [[ $DRY_RUN -eq 1 ]]; then
    matches=$(grep -nE "${OLD_MOD}|gxtLogo|\"${OLD}\"|${OLD}-conflict|${OLD}-commit-msg|${OLD}_status_debug|${OLD}_debug_patch|gxt - Terminal|SetTitle\(\"${OLD}\"|Quit ${OLD}|cmd/${OLD}" "$f" || true)
    [[ -n "$matches" ]] && { echo "--- $f"; echo "$matches"; }
    continue
  fi

  # Order matters: module path first (most specific), then identifiers, then bare tokens.
  sed -i \
    -e "s#${OLD_MOD}#${NEW_MOD}#g" \
    -e "s#gxtLogo#${NEW}Logo#g" \
    -e "s#cmd/${OLD}#cmd/${NEW}#g" \
    -e "s#\"${OLD}\"#\"${NEW}\"#g" \
    -e "s#${OLD}-conflict#${NEW}-conflict#g" \
    -e "s#${OLD}-commit-msg#${NEW}-commit-msg#g" \
    -e "s#${OLD}_status_debug#${NEW}_status_debug#g" \
    -e "s#${OLD}_debug_patch#${NEW}_debug_patch#g" \
    -e "s#gxt - Terminal#${NEW} - Terminal#g" \
    -e "s#Quit ${OLD}#Quit ${NEW}#g" \
    "$f"
done

echo "==> Moving binary package cmd/${OLD} -> cmd/${NEW}"
if [[ -d "cmd/${OLD}" ]]; then
  run git mv "cmd/${OLD}" "cmd/${NEW}"
  if [[ -f "cmd/${NEW}/${OLD}.go" ]]; then
    run git mv "cmd/${NEW}/${OLD}.go" "cmd/${NEW}/${NEW}.go"
  fi
fi

if [[ $DRY_RUN -eq 1 ]]; then
  echo "==> dry run complete; no files changed"
  exit 0
fi

echo "==> Tidying Go module"
go mod tidy
echo "==> Building"
go build ./...

cat <<EOF

==> Done. Project rebranded ${OLD} -> ${NEW}.

MANUAL STEP REMAINING:
  The ASCII-art logo in cmd/${NEW}/${NEW}.go (const ${NEW}Logo) still spells
  "GXT" in block glyphs. Regenerate it for "ichi", e.g.:
      figlet -f banner ichi      # or your preferred figlet font

Also consider:
  - Rename the project directory itself (.../${OLD} -> .../${NEW}) if desired.
  - Update the remote repo name on GitHub and your git remote URL.
EOF
