# The typed API of the protocol

Status: Decided. The maintainer approved this design on 2026-10-04. The
generator writes the `cdproto` side of it, and `cdproto` v0.157.3 is the first
release that has it. `chromedp` v0.17.0 uses it. This document describes what
the generator writes and why. See
`docs/decisions/2026-10-03-proposed-generics-and-iterators-api.md` for the
decision and its history.

## The problem

The code that `pdlgen` wrote before the typed API, up to `cdproto` v0.157.2,
had four weaknesses.

1. A command returned its results as a list of values. `Navigate(...).Do(ctx)`
   returned four values and an error. When the protocol added a fifth value,
   every caller stopped compiling. A struct of results does not have this
   problem, because a new field breaks nobody.
2. The connection was hidden in the `context.Context`. A call with the wrong
   context failed at run time with `ErrInvalidContext`. The compiler did not
   find it.
3. `chromedp` returned values through pointers. `chromedp.Text(sel, &text)`
   filled `text` when `chromedp.Run` ran it. The reader had to know the order of
   the steps to know when the variable was valid.
4. An event listener was a function that took `any`. The caller wrote a type
   switch, and a typo in a case was silent.

Go 1.27 has generics and iterators. They fix all four and keep the code type
safe. Generics have one limit that shapes the design: a method cannot have its
own type parameters. A call that is generic over its parameters and its results
must be a function, not a method.

## The model

The model has four parts.

1. A command is a value. It names the method and carries the types of its
   parameters and results.
2. A session is an explicit value. It owns one connection to the browser.
3. A function runs a command on a session and returns a result struct.
4. An event is a value. A function turns it into an iterator.

## What the generator writes

For the command `Page.navigate` and the event `Page.loadEventFired` in the
package `page`, the generator writes this, with the field comments left out:

```go
// NavigateParams are the parameters of the command Page.navigate.
type NavigateParams struct {
	URL            string         `json:"url"`
	Referrer       string         `json:"referrer,omitempty,omitzero"`
	TransitionType TransitionType `json:"transitionType,omitempty,omitzero"`
	FrameID        cdp.FrameID    `json:"frameId,omitempty,omitzero"`
	ReferrerPolicy ReferrerPolicy `json:"referrerPolicy,omitempty,omitzero"`
}

// NavigateResult is the result of the command Page.navigate.
type NavigateResult struct {
	FrameID    cdp.FrameID  `json:"frameId,omitempty,omitzero"`
	LoaderID   cdp.LoaderID `json:"loaderId,omitempty,omitzero"`
	ErrorText  string       `json:"errorText,omitempty,omitzero"`
	IsDownload bool         `json:"isDownload"`
}

// Navigate navigates current page to the given URL.
var Navigate = cdp.Command[NavigateParams, NavigateResult]{Method: CommandNavigate}

// LoadEventFired [no description].
var LoadEventFired = cdp.Event[EventLoadEventFired]{Method: "Page.loadEventFired"}
```

The constant `CommandNavigate` holds the name `Page.navigate`. The generator
also writes the struct `EventLoadEventFired` for the payload of the event.

A command that takes no parameters uses `cdp.Empty` for `P`, and a command that
returns nothing uses `cdp.Empty` for `R`. A result struct exists for each
command that returns something. This includes a command that returns one value,
so that adding a second value later is not a break.

### The core package

The `cdp` package holds the core. This is a short form of it, with the comments
and the bodies left out. The connection is not here, because it stays in
`chromedp`:

```go
package cdp

// Empty is the parameters or the result of a command that has none.
type Empty struct{}

// Command is a protocol command with the parameters P and the result R.
type Command[P, R any] struct{ Method string }

// Event is a protocol event with the payload E.
type Event[E any] struct{ Method string }

// Session is a connection to a browser target.
type Session interface {
	Call(ctx context.Context, method string, params, result any) error
	Subscribe(method string) (events <-chan jsonv2.Value, cancel func())
}

// Call runs the command on the session.
func Call[P, R any](ctx context.Context, s Session, c Command[P, R], params P) (R, error)

// Events returns the events of one kind. The session starts to buffer them
// when Events returns, and not when the caller starts to range.
func Events[E any](ctx context.Context, s Session, e Event[E]) iter.Seq2[E, error]
```

`Session` is an interface with two methods, so the connection stays in
`chromedp`. A protocol error from the browser is a `*cdproto.Error`, which the
session returns. A caller reads it with `errors.As`.

### The rules of the generated code

- A result struct is named `FooResult`, and a parameter struct is named
  `FooParams`. A command with no parameters or no results uses `cdp.Empty`, and
  has no struct for them.
- The generator does not write a `Do` method, a function for each command or
  the `With...` methods. A generated package cannot hold both forms, because
  the value `page.Navigate` and an old function `page.Navigate` have the same
  name. There is no executor in the context.
- A required parameter has no `omitempty` or `omitzero` in its tag. An optional
  parameter has `omitempty,omitzero`, so a zero value is left out of the
  message.
- An optional boolean in the parameters of a command is a `*bool`. A plain
  `bool` cannot say "leave it out". The protocol has 137 optional booleans in
  the parameters of commands, and some of them have the default true, so a zero
  `false` changes what the browser does. A `nil` pointer leaves the parameter
  out. In Go 1.26 and later, `new(true)` and `new(false)` make the pointer. The
  generated code itself builds with Go 1.25. A
  boolean in a result or in another struct is a plain `bool`.
- An optional number or integer is a plain value, and a zero is left out of the
  message. A short table in `gen/gotpl/util.go` lists the fields where zero is a
  value of its own, such as `input.TouchPoint.ID`, the margins of
  `page.PrintToPDFParams` and the `Depth` of `dom.GetDocumentParams`. Those
  fields are a `*float64` or a `*int64`, in the same way as an optional
  boolean. A `nil` pointer leaves the field out, and `new(float64)` sends 0.
  `input.TouchPoint{ID: new(float64)}` encodes as `{"x":0,"y":0,"id":0}`. See
  `docs/decisions/2026-10-04-an-optional-number-can-be-a-pointer.md` for the
  rule and the list.
- A field that holds an enum has the named type of that enum, so a caller
  cannot pass an arbitrary string by accident.
- A binary value is a `[]byte`. The standard library encodes it as base64
  without a tag. A struct with a text value and a `base64Encoded` flag next to
  it, such as the result of `Network.getResponseBody`, holds the value as a
  `[]byte` and decodes it by the flag in its `UnmarshalJSON`. A text value whose
  description starts with "Base64-encoded" is a `[]byte` as well. Neither case
  needs code in the caller.
- `network.CookiePartitionKey` decodes the plain string that older versions of
  Chrome send, as well as the object that newer versions send. The string is
  the top level site. The type encodes as an object.
- The `cdp` package has no `ErrInvalidContext`. The old API hid the connection in
  the context, and a wrong context failed with that error. The typed API passes
  the session as an argument, so nothing in the generated code used the
  constant. `chromedp` has its own `ErrInvalidContext`.
- `cdp.Events` buffers from the moment it returns. A caller that never ranges
  over the iterator holds the subscription, and the documentation of the
  function says so.
- `gencmd/gencmd_test.go` generates a small protocol and runs
  `gencmd/testdata/generated_test.go.txt` against it with the go command. The
  test covers a command with an enum, a pointer boolean and a binary result, a
  command with nothing, an error, the order of events and the end of an
  iterator.

## How chromedp uses it

`chromedp` keeps the job that it had. It starts a browser, manages targets and
frames, and offers high level actions. `chromedp` v0.17.0 has the code below.
The `Target` of `chromedp` is a `cdp.Session`.

### One command

```go
res, err := chromedp.Call(ctx, page.Navigate, page.NavigateParams{
	URL: "https://go.dev",
})
if err != nil {
	return fmt.Errorf("navigating: %w", err)
}
if res.ErrorText != "" {
	return fmt.Errorf("navigating: %s", res.ErrorText)
}
```

The fields of `page.NavigateParams` are the parameters, and the fields of
`page.NavigateResult` are the results. A reader does not count return values.

### An action that returns a value

An action is a function from a target to a value and an error. It replaces the
out pointer:

```go
title, err := chromedp.Run(ctx, chromedp.Title())
// title is a string. There is no variable to declare before the call.
```

### Steps that return different values

`Run` takes one action, so a sequence of actions with different result types is
an ordinary function:

```go
type Snapshot struct {
	Title string
	PNG   []byte
}

func Fetch(ctx context.Context, url string) (Snapshot, error) {
	var s Snapshot
	var err error
	if err = chromedp.Do(ctx, chromedp.Navigate(url)); err != nil {
		return s, err
	}
	if s.Title, err = chromedp.Run(ctx, chromedp.Title()); err != nil {
		return s, err
	}
	s.PNG, err = chromedp.Run(ctx, chromedp.Screenshot("body"))
	return s, err
}
```

This is plain Go. There is no `Tasks` list and no variable that is empty until a
later step fills it.

### Wait for an event, without a race

The listener must exist before the page can fire the event. `Events` buffers
from the moment it returns, so the order is safe:

```go
loaded := chromedp.Events(ctx, page.LoadEventFired)

if _, err := chromedp.Call(ctx, page.Navigate, page.NavigateParams{URL: url}); err != nil {
	return err
}
for _, err := range loaded {
	if err != nil {
		return fmt.Errorf("waiting for the load event: %w", err)
	}
	break
}
```

### Watch the network

The old API needed a type switch inside `ListenTarget`. With a typed event, the
type is known:

```go
for ev, err := range chromedp.Events(ctx, network.ResponseReceived) {
	if err != nil {
		return err
	}
	fmt.Println(ev.Response.Status, ev.Response.URL)
}
```

The caller enables the domain first, with `network.Enable`.

### An error from the browser

```go
_, err := chromedp.Call(ctx, dom.QuerySelector, dom.QuerySelectorParams{
	NodeID:   root,
	Selector: "(",
})
var perr *cdproto.Error
if errors.As(err, &perr) {
	fmt.Println(perr.Code, perr.Message)
}
```

### Old actions

`chromedp.Legacy` adapts a value that has the `Do(context.Context) error`
method of the previous version. `chromedp.Call` and `chromedp.CallBrowser` work
inside it, because the old action receives the same context.

## What it costs

- Every name changed. `page.Navigate(url).Do(ctx)` became
  `cdp.Call(ctx, s, page.Navigate, page.NavigateParams{URL: url})`. The new
  call is longer. The old chain read well, but a free function cannot chain.
- The generator writes a result struct for every command, so `cdproto` is
  larger.
- `cdp.Call` uses reflection through the JSON package. A hot path can need
  generated code for each command. That makes the API larger.
- An iterator that nobody reads can fill a buffer. The session must drop events
  for a slow reader, or close it, and say so in its documentation.
- The `With...` methods let the caller write one expression. A struct literal
  needs a variable for a long list of fields. Some people prefer the old form.

## What changed in this repository

The work was merged on 2026-10-04. These files hold it:

- `gen/gotpl/extra.tmpl` writes the `cdp` package of this document.
- `gen/gotpl/domain.tmpl` writes the parameter and result structs, the command
  value and the event value, and does not write `Do` or the `With...` methods.
- `gen/gotpl/util.go` writes `[]byte` and `*bool` where the rules above say.
- `docs/GENERATOR.md` describes the files that the generator writes.
