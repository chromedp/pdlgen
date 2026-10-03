# Generate a typed API that uses generics and iterators

Status: Proposed.

On 2026-10-03 the maintainer asked whether Go 1.27 generics and iterators allow a better
API for the protocol than the `chromedp.Action` interface, which is old and
not friendly. The maintainer asked for a document that records the idea, and for example
code that shows how `chromedp` can use it. He did not choose it.

Gemini Pro answered the question. Qwen Max did not answer in time. The answer
and the research are in `docs/API.md`.

## The proposal

A command becomes a value, `cdp.Command[P, R]`. A session becomes an explicit
value. A free function, `cdp.Call`, runs a command and returns a result struct
instead of a list of values. An event becomes a value, `cdp.Event[E]`, and
`cdp.Events` turns it into an `iter.Seq2[E, error]`. In `chromedp`, an action
becomes `Action[T]`, which returns a value and removes the out pointer.

## Why

A list of return values breaks every caller when the protocol adds a value. A
connection hidden in the context fails at run time. An out pointer hides when a
variable is valid. An event handler that takes `any` hides a wrong case.

## What it costs

Every name changes, so this is a break for every user of `cdproto`. The
generated code grows. The call is longer than the chained form, because Go has
no generic methods and so the call must be a function.

## What happens next

The generator side is built on the branch `typed-api`, as an experiment for
review, and `docs/API.md` describes what it generates. It is not merged. Nothing
more happens until the maintainer decides. If the maintainer accepts it, this file is amended to
`Decided`, and the branch is merged.
