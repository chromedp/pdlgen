# The generated code uses go-json-experiment instead of easyjson

Status: Amended by 2026-10-03-the-generated-code-uses-encoding-json-v2.md.

On 2025-02-22 the generator changed from `easyjson` to
`github.com/go-json-experiment/json`, in the commit `ea7a5fa` named "Change
easyjson -> jsonv2". The same day it added `omitzero` to the tags, in `8a900a7`.
The generated code reads and writes JSON with the `jsonv2` package and the
`jsontext` types, and it no longer needs a code generator for the JSON methods.

This record comes from the history of the repository. It does not hold the
reasons, because the commit messages do not give them. If the maintainer remembers the
reasons, add them here.

## What it means today

The generator does not run `easyjson`. The generated `go.mod` requires
`github.com/go-json-experiment/json`. `UnmarshalMessage` takes `jsonv2.Options`.

## Amendment

The generated code now uses the standard library. See
`2026-10-03-the-generated-code-uses-encoding-json-v2.md`. The text above says
what was true from 2025-02-22 until then.
