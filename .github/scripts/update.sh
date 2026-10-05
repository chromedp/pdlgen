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
#   YEAR      the last year of the copyright line of LICENSE [the current year, UTC]
#   GIT_AUTHOR_NAME, GIT_AUTHOR_EMAIL, GIT_COMMITTER_NAME, GIT_COMMITTER_EMAIL
#             identity used for the commit and tag [Kenneth Shaw <kenshaw@gmail.com>]
#
# Requires: go, git, and apidiff (golang.org/x/exp/cmd/apidiff) in PATH.
set -euo pipefail

cdproto=$(realpath "${CDPROTO:-cdproto}")
gen=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
pkg=github.com/chromedp/cdproto
# the last version of the module that needs Go 1.25 or earlier (2026-02-13)
jsonmod=github.com/go-json-experiment/json@44df1a37e875e488af5305814494092b9eb3f38d
branch=main

export GIT_AUTHOR_NAME=${GIT_AUTHOR_NAME:-Kenneth Shaw}
export GIT_AUTHOR_EMAIL=${GIT_AUTHOR_EMAIL:-kenshaw@gmail.com}
export GIT_COMMITTER_NAME=${GIT_COMMITTER_NAME:-$GIT_AUTHOR_NAME}
export GIT_COMMITTER_EMAIL=${GIT_COMMITTER_EMAIL:-$GIT_AUTHOR_EMAIL}

tmp=$(mktemp -d)
trap 'git -C "$cdproto" worktree prune; rm -rf "$tmp"' EXIT

# generate
(cd "$gen" && go run . --out "$cdproto" ${GEN_ARGS:-})

# the generated code builds with Go 1.25. Before Go 1.27, the package cdp/jsonv2
# reads JSON with a module (see docs/decisions/2026-10-06-the-subpackage-cdp-jsonv2-hides-the-json-package.md).
# The module needs a fixed version, because a later version needs a later Go.
# go get changes the go line, so the script sets it again.
(cd "$cdproto" && go mod edit -go=1.25 && go get "$jsonmod" && go mod edit -go=1.25 && go mod tidy)

# verify the generated code with the JSON package of the standard library, and
# with the module. Go 1.27 turns the jsonv2 experiment on, and the module does
# not build with it, so the second run turns it off.
(cd "$cdproto" && go build ./... && go vet ./... && go test ./...)
(cd "$cdproto" && GOEXPERIMENT=nojsonv2 go vet -tags cdproto_jsoncompat ./... && GOEXPERIMENT=nojsonv2 go test -tags cdproto_jsoncompat ./...)

if [ -z "$(git -C "$cdproto" status --porcelain)" ]; then
  echo "no changes"
  exit 0
fi

# A new Chromium or V8 version alone does not make a release. When version.go is
# the only file that changed, the generated API is the same, so the script
# restores the file and stops. The next release that changes the code will
# have the new versions. The check runs before the years of the license change.
if [ -z "$(git -C "$cdproto" status --porcelain | grep -v -E '^.. version\.go$' || true)" ]; then
  echo "no changes except the Chromium or V8 version: $(git -C "$cdproto" diff -U0 -- version.go | grep -E '^[-+][[:space:]]+(chromium|v8)Version' | tr -s '\t ' ' ' | tr '\n' ';')"
  git -C "$cdproto" checkout -- version.go
  exit 0
fi

# set the years of the copyright line of the license to 2016 and the current
# year. This runs after the check above, so that a new year alone does not make
# a release. YEAR is for tests.
year=${YEAR:-$(date -u +%Y)}
if [ -f "$cdproto/LICENSE" ]; then
  sed -i -E "s/^(Copyright \(c\) 2016)(-[0-9]{4})?( )/\1-$year\3/" "$cdproto/LICENSE"
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
