# Backlog

This document lists work that is known and not done. Each item names the
decision or the measurement that found it.

A decision is not a backlog item. It goes in
[`decisions/`](decisions/README.md). A question for the maintainer goes at the end of
[`PLAN.md`](PLAN.md), under Open questions. When an item here is done,
delete it, and record in `decisions/` anything that was decided on the way.

## Generator

### Add a golden test of the generated output

`gen/gen_test.go` checks names. A small protocol file in `testdata`, with the
files that it must generate, catches a change in spacing or in a comment. The
comparison in `docs/GENERATOR.md` finds such a change now, but only when a
person runs it.

### Add a linter configuration

The `xo` projects run `golangci-lint` with a configuration that enables most
linters and lists the exceptions with a reason. This repository has none. See
open question 2 in `PLAN.md`.
