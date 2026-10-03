# The `Update` workflow regenerates and tags `cdproto` every day

Status: Decided.

The maintainer decided on 2026-10-03 that a workflow in this repository regenerates
`cdproto` once a day, and that it commits and tags the result as the repository owner.

## What it does

`.github/workflows/update.yml` runs at 03:17 UTC and can be started by hand. It
runs `.github/scripts/update.sh`, which generates the code, builds it, and
stops when nothing changed. Otherwise it chooses the tag by the rule in
`2026-10-03-cdproto-is-tagged-v0-chromium-major-patch.md`, adds an entry to
`CHANGELOG.md`, commits, tags and pushes. `docs/RELEASES.md` has the details.

## The choices

- The commit is made as the repository owner. The author name and address are
  the values in `update.yml`, which are the values in the git configuration of
  this repository.
- The tag is annotated. It holds the Chromium version, the V8 version and the
  count of API changes.
- `apidiff` counts the changes. The workflow installs it from
  `golang.org/x/exp/cmd/apidiff`.
- The versions are available in the code as the functions `ChromiumVersion` and
  `V8Version`. They are functions and not constants because `apidiff` reports a
  changed constant value as an incompatible change, and every release changes
  the value.
- The old workflow, `cron.yml`, was deleted. It ran on every push, used an old Go
  version, and never committed anything. `.travis.yml` was deleted too, because
  it has not worked for years.

## What it needs

The secret `ACCESS_TOKEN` must be able to push to `chromedp/cdproto`. Whether it
can is an open question in `docs/PLAN.md`.
