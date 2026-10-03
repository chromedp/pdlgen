#!/usr/bin/env bash
#
# Regenerates the cdproto package from the latest Chromium and V8 protocol
# definitions. When the generated code changes, the script commits the result
# to cdproto, tags it with the next version, and pushes the commit and the tag.
#
# Versioning (see docs/RELEASES.md): cdproto stays at major version 0, and the
# minor version is the Chromium major version of the protocol definitions:
#
#   - the first release for a Chromium major version is v0.<major>.0
#   - every later release for the same major version adds one to the patch
#
# The generated API can change in an incompatible way with any update of the
# protocol definitions. The changes to the public API since the last tag, found
# with apidiff, go in the annotation of the tag and in CHANGELOG.md.
#
# Environment:
#   CDPROTO   path to the cdproto checkout (full history and tags) [cdproto]
#   GEN_ARGS  extra arguments passed to pdlgen (--chromium, --v8, ...)
#   PUSH      push the commit and tag when 1 [1]
#   GIT_AUTHOR_NAME, GIT_AUTHOR_EMAIL, GIT_COMMITTER_NAME, GIT_COMMITTER_EMAIL
#             identity used for the commit and tag [Kenneth Shaw <kenshaw@gmail.com>]
#
# Requires: go, git, and apidiff (golang.org/x/exp/cmd/apidiff) in PATH.
set -euo pipefail

cdproto=$(realpath "${CDPROTO:-cdproto}")
gen=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
pkg=github.com/chromedp/cdproto
branch=main

export GIT_AUTHOR_NAME=${GIT_AUTHOR_NAME:-Kenneth Shaw}
export GIT_AUTHOR_EMAIL=${GIT_AUTHOR_EMAIL:-kenshaw@gmail.com}
export GIT_COMMITTER_NAME=${GIT_COMMITTER_NAME:-$GIT_AUTHOR_NAME}
export GIT_COMMITTER_EMAIL=${GIT_COMMITTER_EMAIL:-$GIT_AUTHOR_EMAIL}

tmp=$(mktemp -d)
trap 'git -C "$cdproto" worktree prune; rm -rf "$tmp"' EXIT

# generate
(cd "$gen" && go run . --out "$cdproto" ${GEN_ARGS:-})

# the generated code uses encoding/json/v2, which needs Go 1.27, and it needs no
# module other than the standard library
(cd "$cdproto" && go mod edit -go=1.27 && go mod tidy)

# verify the generated code
(cd "$cdproto" && go build ./... && go vet ./...)

if [ -z "$(git -C "$cdproto" status --porcelain)" ]; then
  echo "no changes"
  exit 0
fi

chromium=$(sed -n 's/^[[:space:]]*chromiumVersion = "\(.*\)"$/\1/p' "$cdproto/version.go")
v8=$(sed -n 's/^[[:space:]]*v8Version[[:space:]]*= "\(.*\)"$/\1/p' "$cdproto/version.go")
[ -n "$chromium" ] && [ -n "$v8" ] || { echo "unable to determine versions" >&2; exit 1; }

# determine the next version
major=${chromium%%.*}
case $major in
  ''|*[!0-9]*) echo "invalid chromium version: $chromium" >&2; exit 1 ;;
esac
last=$(git -C "$cdproto" tag --list 'v0.*' --sort=-version:refname | head -n1)
changes="first tagged release"
if [ -n "$last" ]; then
  # compare the api of the last tag with the generated code
  git -C "$cdproto" worktree add --detach "$tmp/base" "$last" >/dev/null
  cp "$cdproto/go.sum" "$tmp/base/go.sum"
  (cd "$tmp/base" && apidiff -m -w "$tmp/old.apidiff" "$pkg")
  (cd "$cdproto" && apidiff -m -w "$tmp/new.apidiff" "$pkg")
  apidiff -m "$tmp/old.apidiff" "$tmp/new.apidiff" >"$tmp/all.txt"
  apidiff -m -incompatible "$tmp/old.apidiff" "$tmp/new.apidiff" >"$tmp/incompatible.txt"
  total=$(grep -c '^- ' "$tmp/all.txt" || true)
  incompatible=$(grep -c '^- ' "$tmp/incompatible.txt" || true)
  changes="$incompatible incompatible, $((total - incompatible)) compatible"
  [ "$total" -gt 0 ] || changes="none"
fi
latest=$(git -C "$cdproto" tag --list "v0.$major.*" --sort=-version:refname | head -n1)
if [ -z "$latest" ]; then
  next="v0.$major.0"
else
  next="v0.$major.$(( ${latest##*.} + 1 ))"
fi
echo "version: ${last:-none} -> $next (chromium $chromium, v8 $v8, api changes: $changes)"

# the versions and the packages of the last release, for the notes
prev_chromium= prev_v8=
added= removed=
if [ -n "$last" ]; then
  prev_chromium=$(sed -n 's/^[[:space:]]*chromiumVersion = "\(.*\)"$/\1/p' "$tmp/base/version.go")
  prev_v8=$(sed -n 's/^[[:space:]]*v8Version[[:space:]]*= "\(.*\)"$/\1/p' "$tmp/base/version.go")
  packages() { (cd "$1" && for d in */; do ls "$d" | grep -q '\.go$' && echo "${d%/}"; done); }
  packages "$tmp/base" | sort >"$tmp/packages.old"
  packages "$cdproto" | sort >"$tmp/packages.new"
  added=$(comm -13 "$tmp/packages.old" "$tmp/packages.new" | tr '\n' ' ')
  removed=$(comm -23 "$tmp/packages.old" "$tmp/packages.new" | tr '\n' ' ')
fi

# write the notes of the release: the message of the commit, the annotation of
# the tag and the entry of the changelog
relnotes() {
  (cd "$gen" && go run ./cmd/relnotes --mode "$1" --diff "${tmp}/all.txt" --version "$next" --previous "$last" \
    --chromium "$chromium" --v8 "$v8" --prev-chromium "$prev_chromium" --prev-v8 "$prev_v8" \
    --date "$(date -u +%Y-%m-%d)" --added-packages "$added" --removed-packages "$removed")
}
[ -n "$last" ] || : >"$tmp/all.txt"
relnotes commit >"$tmp/commit.txt"
relnotes tag >"$tmp/tag.txt"
relnotes changelog >"$tmp/entry.txt"

# update the changelog
if [ -f "$cdproto/CHANGELOG.md" ]; then
  sed -n '1,/^## /{/^## /!p}' "$cdproto/CHANGELOG.md" | sed -e :a -e '/^\n*$/{$d;N;ba' -e '}' >"$tmp/changelog.new"
else
  cat >"$tmp/changelog.new" <<'HEADER'
# Changelog

Each release of `cdproto` is generated from the Chromium and V8 protocol
definitions listed below. The minor version is the Chromium major version. As
the protocol definitions deprecate and remove commands, events, types, and
fields, any release can contain incompatible changes to the generated API.
HEADER
fi
{
  echo
  cat "$tmp/entry.txt"
  if [ -f "$cdproto/CHANGELOG.md" ]; then
    echo
    sed -n '/^## /,$p' "$cdproto/CHANGELOG.md"
  fi
} >>"$tmp/changelog.new"
mv "$tmp/changelog.new" "$cdproto/CHANGELOG.md"

# commit and tag
git -C "$cdproto" add -A
git -C "$cdproto" commit -q -F "$tmp/commit.txt"
git -C "$cdproto" tag -a "$next" -F "$tmp/tag.txt"

if [ "${PUSH:-1}" = 1 ]; then
  git -C "$cdproto" push origin "HEAD:$branch"
  git -C "$cdproto" push origin "$next"
fi
echo "$next" >"${GITHUB_OUTPUT_VERSION:-/dev/null}"
