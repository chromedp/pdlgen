# The `old` branch became `main`, and the old `main` became `wip`

Status: Decided.

The maintainer decided on 2026-10-03 to move the unfinished work on `main` to
a branch named `wip`. The `old` branch became the new `main`.

## What was there

The `main` branch and the `master` branch both pointed at a commit from
2020-07-09, "always emit Page.setDownloadBehavior for now". It was unfinished
work. The `old` branch held the work that produced `cdproto` for years. It was
51 commits ahead of `main`, and it contained every commit of `main`.

## What was done

`wip` was created at the old `main` commit and pushed. Then `old` was pushed to
`main`. Because `old` already contained `main`, the push was a fast forward and
no history was rewritten. The local branch `old` was renamed to `main`.

## What remains

The branches `old` and `master` still exist on the remote. `master` points at
the 2020 commit. Deleting them is an open question in `docs/PLAN.md`.

The name `wip` was chosen because it is the usual name. A branch `wip2` already
existed, from 2019.
