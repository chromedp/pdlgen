# The templates are standard Go templates

Status: Decided.

The maintainer decided on 2026-10-03 to rewrite the code generation templates.
They now use the `text/template` package of the standard library, in place of
`github.com/valyala/quicktemplate`.

## Why

`quicktemplate` needs a compiler, `qtc`, that turns each template into Go. A
person who changes a template must install `qtc` and run `go generate`, and the
compiled files hold thousands of lines that nobody reads. Standard templates
need no compiler and no dependency. The binary embeds them, so there is one
thing to build.

## What was done

The four `.qtpl` files and their compiled `.qtpl.go` files were replaced with
four `.tmpl` files and `gen/gotpl/gotpl.go`. The new output is byte for byte
the same as the old output, when the rewrites are removed. It was compared on
the cached Chromium 157.0.8084.3 and V8 15.7.23 protocol.

## What it costs

A mistake in a standard template is found when the template runs and not when
it is built. The test in `gen/gen_test.go` runs every template, so a mistake
fails the test.
