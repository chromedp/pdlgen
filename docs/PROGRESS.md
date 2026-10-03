# Progress

This file records where the work stands, so that a session that ends or
crashes can resume. Update it when a piece of work starts or ends. Work that is
known and not done goes in [`BACKLOG.md`](BACKLOG.md), and a decision goes in
[`decisions/`](decisions/README.md).

## Where the work stands

On 2026-10-03 the new generator was pushed to `main`. It has these parts:

- Standard `text/template` templates, and `encoding/json/v2` in the generated
  code.
- Two rewrites: name stuttering and the extraction of inline enums.
- The command is `pdlgen`, built on `ox`, and the module is
  `github.com/chromedp/pdlgen`.
- The `Test` and `Update` workflows, and `.github/scripts/update.sh`, which tags
  `cdproto` as `v0.<Chromium major>.<patch>`.
- Documents, decisions and agent skills.

On the branch `typed-api` the generator writes the typed API of `docs/API.md`.
A command is a value of the type `cdp.Command`, an event is a value of the type
`cdp.Event`, and `cdp.Call` and `cdp.Events` run them. The branch is not
merged, because the maintainer has not decided. A test builds a generated
package and runs the API against it.

`cdproto` has the tag `v0.157.0`, made before the enum extraction. The script
was run against a local copy of `cdproto` for the next release. Nothing about
that run is pushed.

`chromedp` is being ported to the tagged `cdproto`, in the repository of
`chromedp`.

## Waiting

- The secret `ACCESS_TOKEN` must exist before the `Update` workflow can push.
- The open questions at the end of [`PLAN.md`](PLAN.md) wait for the maintainer.
  One of them is the decision to merge the branch `typed-api`.
