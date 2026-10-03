# An optional number can be a pointer, from a table

Status: Decided.

This decision answers open question 6 of `docs/PLAN.md` and the open question at
the end of `docs/API.md`. It also fixes issue 30 of `cdproto`. A caller cannot
send `id: 0` in a `TouchPoint`, and Chrome rejects a request where some touch
points have an id and some have none.

## The problem

An optional field that is not a boolean has a plain Go type, and its tag has
`omitempty,omitzero`. The encoder leaves the field out when it is zero. For most
fields that is right, because the browser default is zero too. For some fields
zero is a value of its own, and an absent field means something else. The
protocol says what in the description of the field: a default that is not zero,
no change, no limit, or a state that is cleared.

## The scan

We read every optional number and integer in the parameters of a command and in
the types that a parameter can hold. We kept a field when its description shows
that zero and absent differ. The first test was the description of the field. A
field where the description says "default: 0" stayed a plain value. So did a
field where zero has no use, such as a width or a scale.

The scan found 71 fields in 25 commands and 12 types. The list is below. The
old rule of the generator (`omitempty,omitzero` for every optional number) hides
a real value in each of them.

## The rule

A table in `gen/gotpl/util.go`, named `pointerNumbers`, lists the fields. It is
in the style of `base64Rules`. The key is the protocol name of a command, for its
parameters, or of a type. The value is the names of the fields. A listed field is
a `*float64` or a `*int64`. Its tag keeps `omitempty,omitzero`. A `nil` pointer
leaves the field out, and a pointer to zero sends zero. In Go 1.26 and later,
`new(float64)` and `new(int64(5))` make the pointer. This is the rule that
`*bool` already follows for an optional boolean in the parameters of a command.

The table applies to the parameters of a command and to a type. It does not apply
to an event or to the result of a command, because a caller reads those. A type
that a result also holds, such as `browser.Bounds`, is a pointer there too. A
caller must dereference it, and must check for `nil` first.

A name in the table that the protocol no longer has is ignored, so a protocol
change cannot break the daily run. A new field that needs a pointer is a change to
the table.

## Why not every optional number

A pointer for every optional number changes about 270 fields, and about 120 of
them are in the parameters of a command. `Scale: 2` becomes `Scale: new(2.0)` for
a field where zero never matters. Every caller of those fields breaks for no
gain. A plain value stays easier to read and to write. The table pays that
cost only where a plain value loses information.

The table has a cost too. A person must read the description of a new optional
field and decide. The table is short, and a test proves that a listed field is a
pointer and that a field that is not listed is plain.

## The fields

A command is named with its parameters. A type is named with its fields.

- `Accessibility.getFullAXTree`: `depth`. Absent means the full tree.
- `Animation.seekAnimations`: `currentTime`. Zero is the start. The command takes
  `currentTime` or `currentTimes`.
- `Audits.getEncodedResponse`: `quality`. Absent means 1, and 0 is a quality.
- `Browser.Bounds`: `left`, `top`. Absent means no change.
- `CSS.forcePositionTryOption`: `index`. 0 is the base position, and absent
  clears the forced state.
- `DOM.describeNode`, `DOM.getDocument`, `DOM.requestChildNodes` and
  `DOMDebugger.getEventListeners`: `depth`. Absent means 1, and 0 is a depth.
- `Debugger.Location` and `Debugger.setBreakpointByUrl`: `columnNumber`. Column 0
  is a column, and absent lets the browser choose.
- `Emulation.SafeAreaInsets`: `top`, `topMax`, `left`, `leftMax`, `bottom`,
  `bottomMax`, `right`, `rightMax`. Absent means no override, and 0 overrides.
- `Emulation.setGeolocationOverride`: `latitude`, `longitude`, `accuracy`,
  `altitude`, `altitudeAccuracy`, `heading`, `speed`. A place on the equator has
  latitude 0, and absent means no value.
- `Emulation.setVirtualTimePolicy`: `budget`. Zero is a budget, and absent means
  no budget.
- `Emulation.updateScreen`: `left`, `top`, `rotation`. Absent means no change.
- `HAR.Content`: `compression`. Absent means that the information is not
  available, and 0 is a value.
- `HAR.PageTimings`: `onContentLoad`, `onLoad`. The HAR format writes -1 for a time that does not
  apply, and 0 is a time.
- `HAR.Timings`: `blocked`, `dns`, `connect`, `ssl`. The HAR format writes -1 for a time that
  does not apply, and 0 is a time.
- `HeadlessExperimental.ScreenshotParams`, `Page.captureScreenshot` and
  `Page.startScreencast`: `quality`. Absent means the browser default, and 0 is a
  quality.
- `IO.read`: `offset`. Absent means to continue after the last read, and 0 is the
  start.
- `IndexedDB.Key`: `number`, `date`. 0 is a key.
- `Input.TouchPoint`: `id`, `radiusX`, `radiusY`, `force`. Absent means a default
  of 1.0 for the last three. 0 is an id and is the case of issue 30.
- `Input.imeSetComposition`: `replacementStart`, `replacementEnd`. 0 is an offset.
- `Input.synthesizeScrollGesture`: `repeatDelayMs`. Absent means 250.
- `Input.synthesizeTapGesture`: `duration`. Absent means 50.
- `LayerTree.replaySnapshot`: `toStep`. Absent means to replay to the end.
- `Network.enable`: `maxTotalBufferSize`, `maxResourceBufferSize`,
  `maxPostDataSize`. Absent means the browser default, and 0 is a size.
- `Network.configureDurableMessages`: `maxTotalBufferSize`,
  `maxResourceBufferSize`.
- `Overlay.DisplayCutoutConfig`: `cx`, `cy`. 0 is a coordinate.
- `Page.printToPDF`: `marginTop`, `marginBottom`, `marginLeft`, `marginRight`.
  Absent means 1 cm, and 0 is a margin.
- `Runtime.SerializationOptions`: `maxDepth`. Absent means full depth.
- `Storage.overrideQuotaForOrigin`: `quotaSize`. Absent clears the override.
- `Target.createTarget`: `left`, `top`. Absent means the browser places the
  window.
- `WebAuthn.Credential` and `WebAuthn.setCredentialProperties`:
  `activeCmtgKeyIndex`. 0 is an index.
- `WebAuthn.setCredentialProperties`: `signCount`. Absent means no change.

## Fields that the scan left plain

- A node id, a backend node id, an execution context id and a certificate id. 0 is
  not an id in the protocol.
- A field whose description says that the default is 0, such as `modifiers`,
  `clickCount`, `deltaX`, `deltaY`, `tiltX`, `tiltY`, `twist` and
  `rotationAngle`.
- A size where 0 has no use: `width`, `height`, `scale`, `paperWidth` and
  `paperHeight`.
- A timestamp where 0 is the epoch. `Network.setCookie` with `expires` 0 is a
  cookie that is already expired, and the caller can delete the cookie instead.
- A field of an event or of a result, because a caller reads it.

## What it costs

A caller of a listed field writes `new(float64)` or `new(int64(5))` in place of a
number. A caller that reads one of these fields from a result must check for
`nil`. The change breaks the build of every caller that uses a listed field, once.
`chromedp` is affected. The maintainer must tell it, and the next release of
`cdproto` carries the change in its notes.
