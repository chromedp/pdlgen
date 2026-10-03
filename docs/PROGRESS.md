# Progress

This file records where the work stands, so that a session that ends or
crashes can resume. Update it when a piece of work starts or ends. Work that is
known and not done goes in [`BACKLOG.md`](BACKLOG.md), and a decision goes in
[`decisions/`](decisions/README.md).

## Where the work stands

On 2026-10-03 the work of the day is staged and not committed. It holds:

- The move of the working branch to `main`, and of the old `main` to `wip`.
  This is done on the remote.
- The new templates, the one rewrite that remains, the version file, the
  `Test` and `Update` workflows, the update script and the generator test.
- `util.Get` now reads the HTTP status, retries a network error, a 429 status
  and a 5xx status up to four times, and never writes a failed response to the
  cache.
- These documents.

The output of the new generator is byte for byte the output of the old one with
the rewrites removed. It was compared on the cached Chromium 157.0.8084.3 and
V8 15.7.23 protocol.

The update script was run against a local copy of `cdproto`. It gave `v0.157.0`
for the first release, `v0.157.1` for a second release of the same Chromium
major version, and `v0.158.0` for a new one. Nothing was pushed to GitHub.

## Waiting

- The maintainer must review the staged change and say when to commit and push.
- The open questions at the end of [`PLAN.md`](PLAN.md) wait for the maintainer.
