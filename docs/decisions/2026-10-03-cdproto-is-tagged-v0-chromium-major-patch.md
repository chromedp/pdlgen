# `cdproto` is tagged v0.<Chromium major>.<patch>

Status: Decided.

The maintainer decided on 2026-10-03 that `cdproto` gets tags of the form
`v0.<Chromium major>.<patch>`. The first tag for Chromium 157 is `v0.157.0`.

## The problem

`cdproto` had no tags. `chromedp` required a pseudo-version, which is an
arbitrary commit. The only record of the protocol version was in a commit
message. A user cannot pin a meaningful version, and cannot tell whether
an update is safe.

The generated API changes in an incompatible way very often. Of the last 40
definition updates, about 31 removed or changed an exported name. The count is
from the removed lines of the diff, so it includes some changes to a comment or
a type. The updates arrive about 50 times a year, and several arrive for each
Chromium major version.

## The rule

The minor version is the Chromium major version. The patch version counts the
releases for that major version, starting at 0. The module stays at major
version 0, so no `/vN` path is needed. `docs/RELEASES.md` describes it.

## What was rejected

- A minor version that counts every update, such as `v0.1.0` then `v0.2.0`.
  This is what `apidiff` can choose. The maintainer chose to follow the Chromium major
  version instead, so a user can see which browser a release matches.
- Version 1 or later. Version 1 promises compatibility, and the generated API
  cannot keep it unless the generator never removes a deprecated name.
- A Chromium based major version such as `v157.0.0`. Go needs a `/v157` suffix
  on every import path from version 2 on, which is 60 or more packages.
- A release branch for each Chromium milestone. It adds work and the number of
  releases is high.
- Build metadata such as `+chromium.157`. Go ignores it when it chooses a
  version.

## What it costs

A patch release can break the API. Within one Chromium major version, `go get
-u` can pull in an incompatible change without a warning from the version
number. The tag annotation and `CHANGELOG.md` count the incompatible changes,
so a person can read them. Gemini Pro first suggested this scheme. It then said
that the patch number was a mistake once it saw the data. The maintainer chose the scheme
with that cost known.
