# Inline enums are named types, and enums decode without validation

Status: Amends 2026-10-03-the-only-fixup-removes-name-stuttering.md.

The maintainer said on 2026-10-03 that removing the enums was a mistake. The
goal is type safe, idiomatic Go, and a plain `string` field lets a caller
supply a value that the protocol will not accept. The question went to six
models, with the options laid out and without a preferred answer, and all six
chose the same design.

## What was removed

The first rewrite of the generator removed every rewrite but one. That also
removed the extraction of inline enums, which are declared directly on a
property, on a command or event parameter, or on a command return value. The
fields became plain strings, for example the `type` of `Input.dispatchKeyEvent`.

## The rules

1. Every inline enum becomes a named string type with a constant for each value.
   This is a second rewrite, and it adds nothing that the protocol does not
   define.
2. The name of the type is the name of the declaring type, command or event,
   followed by the name of the property, as in `DispatchKeyEventType` and
   `StartScreencastFormat`. The constants are the type name followed by the
   value, as in `DispatchKeyEventTypeKeyDown`. There is no list of names to
   keep. A long name is kept long.
3. Two places never share a type, even when their values are the same, because
   the protocol changes the values of each place on its own.
4. If the name of a new type is the name of a type that the domain already has,
   the generator stops with an error. It does not merge them.
5. An array of an inline enum is an array of the named type.
6. A field that is optional stays a plain field with the same `omitzero` tag as
   other fields, and not a pointer.
7. No enum type has its own `UnmarshalJSON`. A value that the generated code does
   not know decodes into the string as it is. A newer browser can send a value
   that an older `cdproto` lacks, and a decode error on a harmless new value
   fails the whole event or response. This also changes the enum types that
   the protocol names.
8. Encoding is the string itself. A caller can write
   `input.DispatchKeyEventType("newValue")` for a value that the browser knows
   and `cdproto` does not.

## What the models said

All six chose named types with lenient decoding. They agreed on naming by the
parent and the property, on no sharing of types, and on no validation when a
value is decoded or encoded. Two of them also asked for a pointer in an optional
field, so that `nil` means the field is absent. That is not done, because every
optional field in the generated code uses `omitzero` and none uses a pointer.
The typed API has one exception: an optional boolean in the parameters
of a command is a `*bool`. See `docs/API.md`.
One asked for an `IsKnown` method and `Deprecated:` comments, which are not done.
They are possible later.

## What it costs

A field that was a `string` is now a named type, so code that assigns a
`string` variable to it stops compiling. A constant such as `"keyDown"` still
works. On the protocol of Chromium 157 the change adds 80 types. `apidiff`
counts about 270 incompatible changes, mostly field types and the removed
`UnmarshalJSON` methods, and about 490 compatible additions. The longest
constant that this rule makes is 77 characters, `SetInstrumentationBreakpointInstrumentationBeforeScriptWithSourceMapExecution`.
The longest in the package is 89 characters, and it belongs to an enum that the
protocol already names. `chromedp` adopted the change in its port to `cdproto`
v0.157.3.
