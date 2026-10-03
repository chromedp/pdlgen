# The notes of a release come from apidiff

Status: Decided.

The maintainer decided on 2026-10-03 that the commit message of a `cdproto`
release must say what changed, and not only the versions.

## What was there

The update script made the message `Updating to <chromium>_<v8> definitions`
with the line `Release: <version>`, and a tag annotation of three lines. A
reader had to run `apidiff` to learn what a release did.

## What it is now

`cmd/relnotes` reads the output of `apidiff -m` for the previous tag and the new
code, and writes the message of the commit, the annotation of the tag and the
entry of `CHANGELOG.md`. The subject stays the same, so the history reads the
same. The body adds the previous versions, the packages added and removed, a
table of counts for each package, and lists of the removed, changed and added
names. `docs/RELEASES.md` describes it.

## Why apidiff

It is the tool that already counts the changes for the tag, it needs no copy of
the previous protocol file, and it describes the Go API, which is what a user of
`cdproto` sees. A comparison of the two protocol files also shows changes that do
not reach the Go code.

## What it costs

A release that changes many names has a long message. A list is cut after twelve
names for each package, and the changelog has only the table.
