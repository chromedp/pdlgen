# The subpackage cdp/jsonv2 hides the JSON package, so that cdproto builds with Go 1.25

Status: Amends 2026-10-03-the-generated-code-uses-encoding-json-v2.md.

The maintainer decided on 2026-10-06 that `cdproto` must build with the line
`go 1.25` in its `go.mod`, for a program on Go 1.25 or Go 1.26 that does not set
`GOEXPERIMENT`. The decision of 2026-10-03 needed Go 1.27. The standard packages
`encoding/json/v2` and `encoding/json/jsontext` exist in Go 1.25 and Go 1.26
only with `GOEXPERIMENT=jsonv2`.

## What changed

The new subpackage `cdp/jsonv2` is the only package that imports a JSON package.
Its name is `jsonv2`, and its names mirror the standard library: `Value`,
`Options`, `Decoder`, `Encoder`, `SyntacticError`, `Marshal`, `Unmarshal`,
`MarshalEncode`, `UnmarshalDecode`, `JoinOptions`, `DefaultOptionsV2` and
`AllowInvalidUTF8`. Every other generated file uses these names, for example
`jsonv2.Value` and `jsonv2.Unmarshal`. A first design put the names in the
package `cdp`, but `cdp.Value` was ambiguous. The subpackage imports no other
package of the module, so the package `cdp`, every domain package and the root
package can import it without a cycle.

Two generated files define the names:

- `cdp/jsonv2/json_std.go` builds with Go 1.27 or later, and with
  `GOEXPERIMENT=jsonv2`. It uses the standard packages. Its types are aliases,
  so the public API is the same as before for a program on Go 1.27.
- `cdp/jsonv2/json_compat.go` builds in all other cases. It uses the module
  `github.com/go-json-experiment/json`.

The build tag `cdproto_jsoncompat` selects the second file on every version of
Go, so that a test can run it with a new toolchain. Go 1.27 turns the `jsonv2`
experiment on, and the pinned module does not build with it. A test on Go 1.27
must set `GOEXPERIMENT=nojsonv2` with the tag.

## The pinned version

The update script pins the module to the commit 44df1a37e875 of 2026-02-13,
which is the version `v0.0.0-20260213210345-44df1a37e875`. It is the last
commit with `go 1.25` in its `go.mod`. The next commit, 4e358349a803, needs
Go 1.26. The script sets the `go` line of `cdproto` again after `go get`.

## Other changes

The `go.mod` of `pdlgen` says `go 1.25`. This needs older releases of
`github.com/xo/ox` and `golang.org/x/tools`, `mod`, `net` and `sync`, because
the newest releases need Go 1.26 or Go 1.27. The generated tests use a helper
for a pointer, because `new(expr)` needs Go 1.26.

## What it costs

A program on Go 1.25 or Go 1.26 gets the module as a dependency, and its types
are not the standard types. A program that passes a `jsonv2.Value` to code that
takes a standard `jsontext.Value` must be on Go 1.27 or set `GOEXPERIMENT`. The
update script must pin the module by hand.
