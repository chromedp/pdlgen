# The command uses ox, and names use ox/strcase

Status: Decided.

The maintainer decided on 2026-10-03 that `pdlgen` moves to
`github.com/xo/ox`, and that the `snaker` package is replaced by `ox/strcase`.
The structure follows `github.com/kenshaw/iv`.

## What was done

`main.go` is now a few lines that call `ox.RunContext`. The command is in the
package `gencmd`, in `gencmd/gencmd.go`. Its `Args` struct holds one field for
each flag, with an `ox` tag that gives the help text, the name and a short
name. `gencmd.New` sets the defaults. The flags that had one dash now have two,
as `--chromium`, `--v8`, `--out` and `--ttl`. `-o` is a short form of `--out`.
The workflows, the update script and the documents use the new form.

`gen/gotpl`, `gen/genutil` and `pdl` import `github.com/xo/ox/strcase` for
`ForceCamelIdentifier`, `ForceLowerCamelIdentifier` and `IsInitialism`. The
generated output is the same as before.

## Notes

`ox` reads a `time.Duration` field as a number, so `--ttl` is a string such as
`24h` and `Exec` parses it. A default in an `ox` tag cannot hold a comma, so
the list of files to keep is set in `gencmd.New`.

`ox` is pinned to the version that `iv` uses.
