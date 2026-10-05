# A new Chromium version alone makes no release

Status: Amends 2026-10-03-the-update-workflow-tags-cdproto-daily.md.

The maintainer decided on 2026-10-06 that the `Update` workflow makes no commit
and no tag when the only change is the Chromium version or the V8 version.

## The problem

Chromium releases a new build many times in a week. Most builds do not change
the protocol. The generated code was the same, but `version.go` held the new
version number, so the script saw a change. It made a commit and a patch tag
that held no change of the API. One example is the commit
`4476000d93bc88eb2d079349be5145767f8cfba1` of `cdproto`, which changed
Chromium 157.0.8085.1 to 157.0.8085.2 and nothing else. The tag v0.157.5
followed.

## The decision

After the generator writes the code, `.github/scripts/update.sh` looks at the
files that changed. When `version.go` is the only one, the script prints the
old and the new version, restores `version.go`, and stops. There is no commit
and no tag.

## The effect

- The constants in `version.go` of the latest release can name an older build
  of Chromium than the newest build. They name the build of the last release
  that changed the code.
- The next release that changes the code writes the current versions. Its
  notes say which versions it replaces, as before.
- A new file or a deleted file counts as a change, and so does a change in any
  other file, such as `go.mod`.
