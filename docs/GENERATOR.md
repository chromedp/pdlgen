# The generator

This document describes how `pdlgen` turns a protocol definition into the
files of `cdproto`. Read it before you change `main.go`, `pdl/`, `fixup/` or
`gen/`.

## The pipeline

`gencmd.Args.run` in `gencmd/gencmd.go` runs these steps in order. `main.go`
only starts it, with `ox`.

1. Find the versions. If `--chromium` is not set, the generator reads the latest
   Chromium version from the Chromium source site. If `--v8` is not set, it
   reads the V8 version from the `DEPS` file of that Chromium version. With
   `--latest` it reads the latest V8 version instead.
2. Retrieve the protocol files. The browser protocol comes from the Chromium
   source tree and the JavaScript protocol comes from V8. Each file is cached
   under the cache directory, which is `<user cache directory>/pdlgen`
   unless `--cache` says otherwise. A cached file is used until it is older than
   `--ttl`, which is 24 hours by default. `--ttl=0` forces a new retrieval.
   `grab.go` retrieves the files that a protocol file includes.
3. Parse and combine. `pdl/parse.go` parses each file. `pdl.Combine` joins the
   two files and the HAR definition in `pdl/har.go`, because no protocol file
   defines the HAR domain. The combined file is written to the cache as
   `combined/<chromium>_<v8>.pdl`, and `diff/` prints how it differs from the
   previous one.
4. Skip what Go does not need. A domain, a type, a command, an event or a
   parameter that the protocol marks as deprecated is skipped. So is one that
   the protocol redirects to another domain. The log names each one.
5. Mark the circular types. A type that two domains need causes an import
   cycle. `pdl/dep.go` lists these types and `pdl.IsCircularDep` reads the
   list. The generator writes them in the `cdp` package.
6. Fix the types. `fixup.FixDomains` gives each inline enum a named type, and it
   removes name stuttering. See
   `docs/decisions/2026-10-03-inline-enums-are-named-types.md` and
   `docs/decisions/2026-10-03-the-only-fixup-removes-name-stuttering.md`.
7. Generate. `gen.NewGoGenerator` fills the templates and returns the files in
   memory.
8. Clean the output directory. The generator removes every file and directory
   that it did not generate, except the names in `--go-wl`. The default list
   keeps `LICENSE`, `README.md`, `CHANGELOG.md`, `*.pdl`, `go.mod` and
   `go.sum`. A name that starts with a dot is always kept.
9. Write the files, run `goimports` on them, and run `gofmt`.

`--debug` writes the files before the last step, which shows what the templates
produce. `--no-clean` skips step 8. `--pdl` reads one combined file and skips the
retrieval.

## The files that it writes

For the package `github.com/chromedp/cdproto` and each domain `<domain>`:

- `cdproto.go` holds the `MethodType`, `Message` and `Error` types and the
  `UnmarshalMessage` function.
- `version.go` holds the Chromium and V8 versions, as `ChromiumVersion` and
  `V8Version`. They are functions on purpose. See
  `docs/decisions/2026-10-03-the-update-workflow-tags-cdproto-daily.md`.
- `cdp/types.go` holds the shared types, the error types, and the core of the
  typed API: `Command`, `Event`, `Empty`, `Session`, `Call` and `Events`.
- `<domain>/<domain>.go` holds the parameter and result structs and the value
  of each command of the domain, and the value of each event.
- `<domain>/types.go` holds its types, and `<domain>/events.go` holds its
  events.

A command `Foo` in the domain `Bar` becomes the struct `FooParams`, the struct
`FooResult`, and the value `Foo` of the type `cdp.Command[FooParams, FooResult]`.
A command without parameters or without results uses `cdp.Empty` in place of the
struct. The constant `CommandFoo` holds the name of the command. An optional
boolean parameter is a `*bool`, so that `nil` leaves it out of the message. A
binary value is a `[]byte`. An event `baz` becomes the struct `EventBaz` and the
value `Baz` of the type `cdp.Event[EventBaz]`. A caller runs a command with
`cdp.Call(ctx, session, bar.Foo, bar.FooParams{...})` and reads an event with
`cdp.Events(ctx, session, bar.Baz)`. `docs/API.md` describes the design.

## The templates

The templates are in `gen/gotpl/`, in four files:

- `file.tmpl` writes the package header and the import block.
- `domain.tmpl` writes the commands of a domain.
- `type.tmpl` writes a type, its enum values and its enum methods.
- `extra.tmpl` writes the shared executor, the method list, the message
  unmarshaler and the version functions.

`gen/gotpl/gotpl.go` embeds the files, builds the function map and holds the
functions that start a template. `gen/gotpl/util.go` holds the functions that
the templates call to compute a name, a type or a comment.

The templates leave some spacing to `gofmt`, but a blank line in a template is
a blank line in the output. `gofmt` keeps one blank line and does not add one.

## Changing a template

A template change is safe when it changes only what you meant to change. Use a
fixed protocol file, so that the protocol does not move while you work.

1. Generate once from the committed code, before your change. Use the Chromium
   and V8 versions of a protocol file that is already cached, and a large
   `--ttl` so that nothing is retrieved:

   ```bash
   mkdir -p /tmp/before && cp ../cdproto/go.mod ../cdproto/go.sum /tmp/before/
   go run . --out /tmp/before --chromium 157.0.8084.3 --v8 15.7.23 --ttl 87600h
   ```

2. Make your change.
3. Generate again into another directory with the same options:

   ```bash
   mkdir -p /tmp/after && cp ../cdproto/go.mod ../cdproto/go.sum /tmp/after/
   go run . --out /tmp/after --chromium 157.0.8084.3 --v8 15.7.23 --ttl 87600h
   ```

4. Compare the two trees with `diff -r /tmp/before /tmp/after`. A refactor must
   print nothing. A change to the output must print exactly the change.
5. Build the new tree and run `go vet` on it:

   ```bash
   (cd /tmp/after && go build ./... && go vet ./...)
   ```

6. Run `go test ./...` in this repository.

To see how a change affects the public API, run `apidiff` on the two trees. The
`Update` workflow does the same thing to count the changes of each release.

## Names

`gen/gotpl/util.go` decides every Go name. A protocol name becomes an exported
name with `strcase.ForceCamelIdentifier`, from `github.com/xo/ox/strcase`. A parameter name that is a Go keyword
gets the suffix `Val`. `gen/genutil/genutil.go` decides how a comment is
worded and wrapped to 80 columns, and it holds the lists of words that keep
their case, such as `DOM` and `UTC`.

## What the generator does not do

It does not add a method, a type or a field that the protocol does not define.
The helpers that `chromedp` needs, such as node helpers and key modifiers, are
in `chromedp`. See `docs/decisions/2026-10-03-the-only-fixup-removes-name-stuttering.md`.
