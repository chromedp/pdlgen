# Releases

This document describes how `cdproto` is regenerated, versioned and tagged.
Read it before you change a workflow, the update script or the version rules.

## The version rule

`cdproto` stays at major version 0. A tag has the form `v0.<major>.<patch>`:

- The minor version is the Chromium major version of the protocol, for example
  157.
- The first release for a Chromium major version has patch 0, so it is
  `v0.157.0`.
- Each later release for the same major version adds one to the patch, so the
  next is `v0.157.1`.
- The first release for Chromium 158 is `v0.158.0`.

The generated API can change in an incompatible way in any release, because the
protocol removes and renames things. The tag does not say whether a release is
compatible. The annotation of the tag and `CHANGELOG.md` in `cdproto` say it.
Each lists the Chromium and V8 versions and counts the incompatible and the
compatible changes since the previous tag. See
`docs/decisions/2026-10-03-cdproto-is-tagged-v0-chromium-major-patch.md` for the
choices that were made and the ones that were rejected.

The package itself reports its versions. `cdproto.ChromiumVersion()` returns
the Chromium version and `cdproto.V8Version()` returns the V8 version.

## The workflows

`.github/workflows/test.yml` is the `Test` workflow. It runs on every push to
`main` and on every pull request. It has two jobs:

- `test` makes sure that the code is formatted and that `go.mod` is tidy, then
  runs `go vet` and `go test -race`.
- `generate` generates `cdproto` from the latest protocol and makes sure that
  the result builds and passes `go vet`.

`.github/workflows/update.yml` is the `Update` workflow. It runs once a day at
03:17 UTC and can be started by hand, with an optional Chromium version and V8
version. It checks out this repository and `cdproto`, installs `apidiff`, and
runs `.github/scripts/update.sh`. The checkout of `cdproto` uses the secret
`ACCESS_TOKEN`, which must be able to push to `chromedp/cdproto`.

## What the update script does

`.github/scripts/update.sh` does these steps:

1. Run the generator, and write the result into the `cdproto` checkout.
2. Set the `go` line of the `go.mod` of `cdproto` to 1.27 and run `go mod tidy`.
   The generated code uses `encoding/json/v2` from the standard library, so it
   needs Go 1.27 and no other module.
3. Run `go build` and `go vet` in `cdproto`. If either fails, the script stops
   and nothing is committed.
4. If `git status` shows no change, print `no changes` and stop.
5. Read the Chromium and V8 versions from `version.go`.
6. Compare the public API of the last tag with the new code, using `apidiff`.
7. Choose the next version by the rule above.
8. Add an entry to `CHANGELOG.md`.
9. Commit with the message `Updating to <chromium>_<v8> definitions`, make an
   annotated tag, and push the commit and the tag.

The commit and the tag are made as the repository owner. The script sets the author from
`GIT_AUTHOR_NAME` and `GIT_AUTHOR_EMAIL`, and the workflow sets both.

## Running the script on your machine

Do not push from a test run. Clone `cdproto` into a bare repository and use that
as the remote:

```bash
git clone --bare ../cdproto /tmp/remote.git
git clone /tmp/remote.git /tmp/work
CDPROTO=/tmp/work PUSH=0 .github/scripts/update.sh
```

`PUSH=0` skips the push. `GEN_ARGS` passes more options to the generator, for
example `GEN_ARGS="--chromium 157.0.8084.3 --v8 15.7.23 --ttl 87600h"`. The script
needs `apidiff` on the path. Install it with
`go install golang.org/x/exp/cmd/apidiff@latest`.

To test a version change without waiting for a new protocol, write a changed
copy of a cached combined protocol file and pass it with `--pdl`, together with
new `--chromium` and `--v8` values.

## A bad release

Never move or delete a tag after it is pushed, because the Go module proxy
keeps the first copy that it saw. If a release is bad, add a `retract`
directive for it to the `go.mod` of `cdproto`, then let the next daily run make
a new release.

The Go module proxy learns of a tag when somebody asks for it. To make a new
release available at once, ask for it:

```bash
curl https://proxy.golang.org/github.com/chromedp/cdproto/@v/v0.157.0.info
```
