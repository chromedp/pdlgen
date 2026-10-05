# Progress

This file records where the work stands, so that a session that ends or
crashes can resume. Update it when a piece of work starts or ends. Work that is
known and not done goes in [`BACKLOG.md`](BACKLOG.md), and a decision goes in
[`decisions/`](decisions/README.md).

## Where the work stands

On 2026-10-03 the new generator was pushed to `main`. It has these parts:

- Standard `text/template` templates, and `encoding/json/v2` in the generated
  code. On 2026-10-06 the package `cdp` took over the JSON names, so that
  `cdproto` builds with Go 1.25.
- Two rewrites: name stuttering and the extraction of inline enums.
- The command is `pdlgen`, built on `ox`, and the module is
  `github.com/chromedp/pdlgen`.
- The `Test` and `Update` workflows, and `.github/scripts/update.sh`, which tags
  `cdproto` as `v0.<Chromium major>.<patch>`.
- Documents, decisions and agent skills.

On 2026-10-04 the typed API was merged to `main`. The generator writes the typed
API of `docs/API.md`. A command is a value of the type `cdp.Command`, an event is
a value of the type `cdp.Event`, and `cdp.Call` and `cdp.Events` run them. A test
builds a generated package and runs the API against it. The same day the
maintainer approved the generics and iterators API, the pipe transport, the
visible window option and the port of the examples.

On 2026-10-04 three fixes followed, each in its own commit on `main`, not yet
released. An optional number that zero can mean something for is now a pointer,
from a table of 71 fields. See
`decisions/2026-10-04-an-optional-number-can-be-a-pointer.md`. This fixes issue
30 of `cdproto`. `network.CookiePartitionKey` decodes the plain string of older
Chrome versions and the object. `cdp.ErrInvalidContext` is gone, and the package
comment of `cdproto.go` says what the package is. These changes break the
build of a caller that uses a listed field, so `chromedp` must adapt before it
takes the next `cdproto` release.

`cdproto` v0.157.3 is the first release with the typed API, and `chromedp`
v0.17.0 uses it. `cdproto` v0.157.0, v0.157.1 and v0.157.2 have the old API.

## Waiting

- The secret `ACCESS_TOKEN` must exist before the `Update` workflow can push.
- The open questions at the end of [`PLAN.md`](PLAN.md) wait for the maintainer.
