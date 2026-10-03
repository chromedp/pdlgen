# An object type without properties is a map

Status: Decided.

On 2026-10-03 the port of the `examples` repository found that `network.Headers`
was an empty struct, so a program cannot send or read a header. The protocol
defines `Network.Headers` as an object with no declared properties: "Request /
response headers as keys / values of JSON object". The generator wrote
`type Headers struct{}`.

## The cause

The old generator had a rewrite that changed `Network.Headers` to
`map[string]any`. The decision to keep only the stuttering fix removed it, and the
published `cdproto` v0.157.0 and v0.157.1 have the empty struct. Nothing in the
tests used a header.

## The rule

An object type that the protocol declares without any properties is a
`map[string]any`, in every domain. It is a rule of the generator, and it needs no
list of names. Two types follow it today: `Network.Headers` and
`Tracing.MemoryDumpConfig`. An object with declared properties is a struct, as
before.

## What it costs

`Headers` is a map, so a program writes `network.Headers{"X-Header": "value"}`.
The values are `any`, because the protocol does not say more. A new release of
`cdproto` is needed, and it changes the type of two names.
