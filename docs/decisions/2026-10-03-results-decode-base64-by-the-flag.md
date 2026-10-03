# A result decodes a base64 value by its flag

Status: Decided.

The maintainer decided on 2026-10-03 that the typed API must decode a base64
value automatically, as the old `Do` method did. The typed result of
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

## What it costs

The result has an `UnmarshalJSON` method of its own, and the five result types
are the only ones that have one. A new rule needs no list of names, because the
generator finds the pair. A protocol change that adds a sixth pair is covered.
