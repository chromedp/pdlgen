# The generated code uses encoding/json/v2 from the standard library

Status: Decided.

This decision supersedes
`2025-02-22-the-generated-code-uses-go-json-experiment.md`. The maintainer
decided on 2026-10-03 that the generated code must not use
`github.com/go-json-experiment/json`. It must use `encoding/json/v2` and
`encoding/json/jsontext`, which are in the standard library of Go 1.27.

## Why

The experiment module became the standard library packages. A package from the
standard library needs no dependency and no version choice, and it is the one
that every Go program already has.

## What changed

The import map in `gen/gogen.go` names `encoding/json/v2`, with the alias
`jsonv2`, and `encoding/json/jsontext`. Nothing else in the templates changed,
because the names and the struct tags are the same. The generated code builds
and passes `go vet`, and after `go mod tidy` the `go.mod` of `cdproto` requires
no module at all. The timestamp types, which used `sysutil`, are gone with the
fixups.

## What it costs

`cdproto` needs Go 1.27 or later. The update script sets the `go` line of the
`go.mod` of `cdproto` to 1.27 and runs `go mod tidy`. A program that builds with
an older Go stops compiling. The maintainer decided on 2026-10-03 that the
minimum Go version of `chromedp` is also 1.27.
