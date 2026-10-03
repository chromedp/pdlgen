# The only fixup removes name stuttering

Status: Decided.

The maintainer decided on 2026-10-03 that the generated code must be regular, and that
the only rewrite the generator can make removes name stuttering.

## What was removed

`fixup/fixup.go` held about 590 lines of rewrites. They added types and
methods that the protocol does not define, such as `Node.Attribute`, the
`FrameState` type, `input.Modifier` and the timestamp types. They changed
parameters to named enum types, and they forced some fields to always appear in
the message. The generator also had two special cases for `Page` and `Runtime`
in `main.go`, and a step that renamed a method after generation. All of these
are gone.

## What remains

A type whose name starts with the name of its domain loses that prefix, because
the package name already says it. The type `CSSStyle` in the package `css`
becomes `css.Style`. For the `Accessibility` domain, the prefix `AX` is also
removed. The rule is in `fixup.FixDomains`.

A type moves to the `cdp` package when two domains need it and an import cycle
results. That is in `pdl/dep.go`. It is not a rewrite, because it only
decides where the code goes. The type `Page.SecurityOriginDetails` joined that
list, because the fixup that used to move it is gone.

## What it costs

The generated API changes. `chromedp` uses several of the removed helpers, so
it must provide them itself. The enum parameters become plain strings, and a
field that is zero is left out of the message unless the protocol requires it.
`cdproto` does not hold a convenience that the protocol does not define.

## Why not keep them

Each rewrite made the code differ from the protocol, and each one needed a
maintainer who knew why it was there. Anything that `chromedp` needs belongs in
`chromedp`, where it can change without a protocol update.
