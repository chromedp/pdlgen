# Plan

This document holds the purpose of `pdlgen`, its architecture, what exists
and how it is tested. It ends with the open questions.

## Purpose

`pdlgen` turns the Chrome DevTools Protocol definitions into the Go
package `github.com/chromedp/cdproto`. It aims to produce type safe, fast and
regular Go that any program can use to drive Chrome, and not only `chromedp`.
The development of the generator follows the needs of `chromedp`.

The protocol changes every few days. The generator must keep up without a
person, and it must tell the people who use `cdproto` what changed.

## Architecture

The input is two protocol files, one from Chromium and one from V8, and a HAR
definition that this repository holds. The output is one Go package with a
sub-package for each domain. `docs/GENERATOR.md` describes each step.

A daily workflow runs the generator, builds the result, and when the code
changes, commits it to `cdproto` and tags a release. A new Chromium or V8
version alone is not a change. `docs/RELEASES.md` describes the tags.

## What exists

- The generator and its templates, which are standard `text/template` files.
- Two rewrites. One removes name stuttering, and one gives each inline enum a
  named type with a constant for each value. Enums decode without validation.
- A generated `version.go` that reports the Chromium and V8 versions.
- The `Test` workflow, which tests the generator and builds the output with
  Go 1.25, Go 1.26 and the stable version.
- The `Update` workflow, which regenerates, tags and pushes `cdproto` every day.
  It makes no release when only the Chromium or V8 version changed.
- The package `cdp/jsonv2`, which is the only generated package that imports a
  JSON package. It lets `cdproto` build with Go 1.25 and later. The generator
  itself uses the current version of Go.
- The typed API that uses generics and iterators, which the maintainer approved
  on 2026-10-04. `docs/API.md` describes it. The generated code has no `Do`
  method and no `With` method. `cdproto` v0.157.3 is the first release that has
  it, and `chromedp` v0.17.0 and later use it.
- A table of optional numbers that are pointers, so that a caller can send an
  explicit zero. See `decisions/2026-10-04-an-optional-number-can-be-a-pointer.md`.
- A decoder for the two forms of `network.CookiePartitionKey`.
- Documents and decisions for coding agents.

## Testing

`gen/gen_test.go` generates code from a small protocol in the test and makes
sure that every file is valid Go and holds the expected names. `fixup/` tests
the enum extraction, and `relnotes/` tests the notes of a release. `gencmd/`
runs the whole command on a small protocol file. It also builds the output and
runs a test of the typed API against it. `util/` tests retrieval against a local
server, and `diff/` tests the protocol diff. `docs/docs_test.go` checks the
documents. No test uses the network.

The `generate` job of the `Test` workflow is the end to end test. It retrieves
the latest protocol, generates `cdproto`, and builds, vets and tests it with
each version of Go in its matrix and with both JSON layers. Its weakness is
that it depends on the Chromium source site.

A change to a template is compared against the old output. See
`docs/GENERATOR.md`, under Changing a template.

## Open questions

1. Does the maintainer want `origin/master` deleted? It is the old default
   branch, and it still points at the 2020 commit. The default branch is
   `main`.
2. Does the maintainer want a `golangci-lint` configuration here? The `xo`
   projects have one.
3. Does the maintainer want an `IsKnown` method, or `Deprecated:` comments, on
   the generated enum types? Two models suggested them. See
   `decisions/2026-10-03-inline-enums-are-named-types.md`.
