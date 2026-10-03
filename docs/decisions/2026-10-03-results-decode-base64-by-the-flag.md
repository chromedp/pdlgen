# A struct decodes a base64 value automatically

Status: Decided.

The maintainer decided on 2026-10-03 that the typed API must decode a base64
value automatically, as the old `Do` method did, wherever the protocol says that
a value is base64. The typed result of
`Network.getResponseBody` first had a `Body string` and a `Base64encoded bool`,
so every caller had to decode the body by hand.

## The rule

A binary value in the protocol is a `[]byte`, and the standard library JSON
package decodes it from base64 with no code. Five commands have a different
shape: a text value and a flag `base64Encoded` next to it. They are
`Fetch.getResponseBody`, `Network.getResponseBody`, `Network.getRequestPostData`,
`Page.getResourceContent` and `IO.read`. The generator finds the pair by the name
`base64Encoded` and the string value before or after it. The value of the result
is a `[]byte`, and the generator writes an `UnmarshalJSON` for the result. It
decodes the value from base64 when the flag is true, and keeps the text as bytes
when the flag is false or missing. A value that is not base64 gives an error. The
flag stays in the result.

## Other base64 text

The rule is not limited to results. It applies to every struct, so a type or an
event that has a text value and a `base64Encoded` flag decodes the same way. Today
only the five results above have the pair. A text value whose description starts
with "Base64-encoded", and that has no flag, is a `[]byte` too, so the JSON
package decodes it. `Storage.PrivateVerificationToken.token` is the only one.

Three text values that mention base64 are left as strings, because the decoding
has a condition that the generator cannot read from a field name:
`Network.WebSocketFrame.payloadData` is base64 only when the opcode is not 1,
`HAR.Content.text` is base64 when `encoding` says so, and
`Network.SignedExchangeHeader.headerIntegrity` is a hash text.

## What it costs

The result has an `UnmarshalJSON` method of its own, and the five result types
are the only ones that have one. A new rule needs no list of names, because the
generator finds the pair. A protocol change that adds a sixth pair is covered.
