# pdlgen

`pdlgen` generates Go code for the commands, events and types of the
[Chrome DevTools Protocol][devtools-protocol]. It is a core component of the
[`chromedp`][chromedp] project. The generator follows the needs of `chromedp`.
Its aim is to produce
[type safe, fast, efficient, idiomatic Go code][cdproto] that any Go program can
use to drive Chrome.

Every issue and every pull request for the `cdproto` package belongs in this
repository. None belongs in the `cdproto` repository, because `cdproto` is
generated output.

A workflow in this repository regenerates `cdproto` every day. When the
generated code changes, the workflow commits it and tags a release. See
[Releases](#releases).

## Installing

Install `pdlgen` in the usual Go way:

```sh
go install github.com/chromedp/pdlgen@latest
```

## Using

`pdlgen` generates the [`github.com/chromedp/cdproto`][cdproto-godoc]
package and a `github.com/chromedp/cdproto/<domain>` package for each domain. A
domain is a group of related commands, events and types, such as `Page`. The
`--out` option names the directory of a checkout of the [`cdproto`][cdproto]
repository:

```sh
pdlgen --chromium=157.0.8084.3 --v8=15.7.23 --out=../cdproto
```

The generator retrieves the protocol files from the [Chromium source
tree][chromium-src] and caches them. Then it replaces the contents of the
output directory. It keeps `LICENSE`, `README.md`, `CHANGELOG.md`, `*.pdl`,
`go.mod`, `go.sum` and any name that starts with a dot.

### Command-line options

```sh
pdlgen --help
```

The options are:

- `--chromium` is the Chromium version of `browser_protocol.pdl`. Any Git tag or
  branch of the Chromium source tree works. The default is the latest version.
- `--v8` is the V8 version of `js_protocol.pdl`. The default is the version in the
  `DEPS` file of the Chromium version. `--latest` uses the latest V8 version.
- `--pdl` reads one combined protocol file and retrieves nothing.
- `--out` or `-o` is the output directory.
- `--cache` is the cache directory. The default is `pdlgen` in the user
  cache directory.
- `--ttl` is how long a cached file is used. The default is 24 hours, and `0`
  forces a new retrieval.
- `--go-pkg` is the Go package name of the output. The default is
  `github.com/chromedp/cdproto`.
- `--go-wl` is the comma-separated list of files that the cleaning step keeps.
  The default is `LICENSE,README.md,CHANGELOG.md,*.pdl,go.mod,go.sum`.
- `--no-clean` keeps the files that the generator did not write.
- `--debug` writes the files without `goimports` and `gofmt`.
- `--version` or `-v` prints the version of `pdlgen`, and `--help` or `-h`
  prints the options.

## How it works

The generator reads the protocol, skips what the protocol deprecates, and fills
standard Go [`text/template`][text-template] templates. Two rewrites change the
types. One removes name stuttering, so `css.CSSStyle` becomes `css.Style`. The
other gives each inline enum a named type. An inline enum is an enum that a
property or a parameter declares in place. For example, the `type` parameter of
`Input.dispatchKeyEvent` has the type `input.DispatchKeyEventType`.

Types that two domains need move to the `cdp` package, to prevent import
cycles. The `cdp` package also holds the typed API. Each command is a value of
the type `cdp.Command` with a parameter struct and a result struct. Each event
is a value of the type `cdp.Event`. [`docs/API.md`](docs/API.md)
describes that API, and [`docs/GENERATOR.md`](docs/GENERATOR.md) describes every
step.

## Releases

`cdproto` is tagged `v0.<Chromium major>.<patch>`, so the first release for
Chromium 157 is `v0.157.0`, and the next is `v0.157.1`. The `Update` workflow
makes the tags. `v0.157.3` is the first release with the typed API. Each tag and
`CHANGELOG.md` in `cdproto` record the Chromium and V8 versions and the count of
API changes. The module stays at major version 0. Any release can contain an
incompatible change, because the protocol removes and renames things.

A new Chromium or V8 version alone makes no release. If the generated API is
the same, the workflow makes no commit and no tag. For this reason, the
Chromium version in `version.go` can be older than the newest Chromium build.
The package reports its versions with `cdproto.ChromiumVersion()` and
`cdproto.V8Version()`. [`docs/RELEASES.md`](docs/RELEASES.md) has the rules.

## Supported Go versions

The generated `cdproto` package builds with Go 1.25 and later. `pdlgen`, which
generates it, uses the current version of Go. Go 1.27 has `encoding/json/v2` in
the standard library, but Go 1.25 and Go 1.26 have it only with
`GOEXPERIMENT=jsonv2`. To work without that setting, the subpackage
`cdp/jsonv2` is the only generated package that imports a JSON package. It has
the names of the standard library, such as `jsonv2.Value` and
`jsonv2.Unmarshal`. On Go 1.27 or with `GOEXPERIMENT=jsonv2`, it uses the
standard packages. In all other cases it uses the module
`github.com/go-json-experiment/json`. The build tag `cdproto_jsoncompat` selects
the module on every version of Go, for tests. Use `GOEXPERIMENT=nojsonv2` with
the tag on Go 1.27. See
[the decision](docs/decisions/2026-10-06-the-subpackage-cdp-jsonv2-hides-the-json-package.md).

## Documents

| Document | Contents |
| --- | --- |
| [`AGENTS.md`](AGENTS.md) | The rules for coding agents, which apply to a person too |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | How to change this repository |
| [`docs/GENERATOR.md`](docs/GENERATOR.md) | The pipeline, the templates and how to change them |
| [`docs/RELEASES.md`](docs/RELEASES.md) | The workflows, the tags and the changelog |
| [`docs/API.md`](docs/API.md) | The typed API that uses generics and iterators |
| [`docs/PLAN.md`](docs/PLAN.md) | Purpose, architecture, testing and open questions |
| [`docs/BACKLOG.md`](docs/BACKLOG.md) | Work that is known and not done |
| [`docs/PROGRESS.md`](docs/PROGRESS.md) | Where the work stands |
| [`docs/decisions/`](docs/decisions/README.md) | Every decision, one file each, named by date |

[devtools-protocol]: https://chromedevtools.github.io/devtools-protocol/
[chromedp]: https://github.com/chromedp
[cdproto]: https://github.com/chromedp/cdproto
[chromium-src]: https://chromium.googlesource.com/chromium/src.git
[cdproto-godoc]: https://godoc.org/github.com/chromedp/cdproto
[text-template]: https://pkg.go.dev/text/template
