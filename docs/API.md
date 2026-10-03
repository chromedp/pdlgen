# A typed API for the protocol

Status: Proposed. Nobody has chosen this design. The generator on the branch
`typed-api` writes the `cdproto` side of it. The generated code builds, passes
`go vet` and has a test. The `chromedp` side below is a sketch, and none of it
compiles. The names and the shapes can change. This document records the design
so that the maintainer can decide. It also shows how `chromedp` can use the
result. See
`docs/decisions/2026-10-03-proposed-generics-and-iterators-api.md`.

## The problem

The code that `pdlgen` wrote before the branch `typed-api` has four
weaknesses.

1. A command returns its results as a list of values. `Navigate(...).Do(ctx)`
   returns four values and an error. When the protocol adds a fifth value,
   every caller stops compiling. A struct of results does not have this
   problem, because a new field breaks nobody.
2. The connection is hidden in the `context.Context`. A call with the wrong
   context fails at run time with `ErrInvalidContext`. The compiler cannot
   find it.
3. `chromedp` returns values through pointers. `chromedp.Text(sel, &text)` fills
   `text` when `chromedp.Run` runs it. The reader must know the order of the
   steps to know when the variable is valid.
4. An event listener is a function that takes `any`. The caller writes a type
   switch, and a typo in a case is silent.

Go 1.27 has generics and iterators. They can fix all four and keep the code
type safe. Generics have one limit that shapes the design: a method cannot have
its own type parameters. A call that is generic over its parameters and its
results must be a function, not a method.

## The model

The model has four parts.

1. A command is a value. It names the method and carries the types of its
   parameters and results.
2. A session is an explicit value. It owns one connection to the browser.
3. A function runs a command on a session and returns a result struct.
4. An event is a value. A function turns it into an iterator.

### What the generator writes in this model

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

A command that takes no parameters uses `cdp.Empty` for `P`, and a command
that returns nothing uses `cdp.Empty` for `R`. A result struct exists for each
command that returns something, even for a command that returns one value, so
that adding a second value later is not a break.

### The core package

The `cdp` package holds the core in about one hundred lines, with the
comments. The sketch shortens it and leaves out the connection, which stays in
`chromedp`:

```go
package cdp

// Empty is the parameters or the result of a command that has none.
type Empty struct{}

// Command is a protocol command with parameters P and result R.
type Command[P, R any] struct{ Method string }

// Event is a protocol event with the payload E.
type Event[E any] struct{ Method string }

// Session is a connection to a browser target.
type Session interface {
	Call(ctx context.Context, method string, params, result any) error
	Subscribe(method string) (events <-chan jsontext.Value, cancel func())
}

// Call runs the command on the session.
func Call[P, R any](ctx context.Context, s Session, c Command[P, R], p P) (R, error)

// Events returns the events of one kind. The session starts to buffer them
// when Events returns, and not when the caller starts to range.
func Events[E any](ctx context.Context, s Session, e Event[E]) iter.Seq2[E, error]
```

A protocol error from the browser is a `*cdproto.Error`, which the session
returns. A caller reads it with `errors.As`.

## How chromedp uses it

`chromedp` keeps the job it has today. It starts a browser, manages targets and
frames, and offers high level actions. The examples show the new shape. They do
not compile.

### A session and one command

```go
s, err := chromedp.Open(ctx, chromedp.Headless())
if err != nil {
	return err
}
defer s.Close()

res, err := cdp.Call(ctx, s.Page(), page.Navigate, page.NavigateParams{
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

An action is a function from a session to a value and an error. It replaces the
out pointer:

```go
// Action is a step that produces a T.
type Action[T any] func(ctx context.Context, s *chromedp.Session) (T, error)

title, err := chromedp.Run(ctx, s, dom.Text("title"))
// title is a string. There is no variable to declare before the call.
```

`dom.Text` here is a `chromedp` helper, and it returns `Action[string]`.

### Steps that return different values

`Run` takes one action, so a sequence of actions with different result types is
an ordinary function:

```go
type Page struct {
	Title string
	Links []string
	PNG   []byte
}

func Fetch(ctx context.Context, s *chromedp.Session, url string) (Page, error) {
	var p Page
	var err error
	if _, err = chromedp.Run(ctx, s, chromedp.Navigate(url)); err != nil {
		return p, err
	}
	if p.Title, err = chromedp.Run(ctx, s, dom.Text("title")); err != nil {
		return p, err
	}
	if p.Links, err = chromedp.Run(ctx, s, dom.Attributes("a", "href")); err != nil {
		return p, err
	}
	p.PNG, err = chromedp.Run(ctx, s, chromedp.Screenshot("body"))
	return p, err
}
```

This is plain Go. There is no `Tasks` list and no variable that is empty until
a later step fills it.

### Wait for an event, without a race

The listener must exist before the page can fire the event. `Events` buffers
from the moment it returns, so the order is safe:

```go
loaded := cdp.Events(ctx, s.Page(), page.LoadEventFired)

if _, err := cdp.Call(ctx, s.Page(), page.Navigate, page.NavigateParams{URL: url}); err != nil {
	return err
}
for ev, err := range loaded {
	if err != nil {
		return fmt.Errorf("waiting for the load event: %w", err)
	}
	_ = ev
	break
}
```

`chromedp.WaitLoaded` wraps those lines.

### Watch the network

Today the caller writes a type switch inside `ListenTarget`. With a typed event,
the type is known:

```go
for ev, err := range cdp.Events(ctx, s.Page(), network.ResponseReceived) {
	if err != nil {
		return err
	}
	fmt.Println(ev.Response.Status, ev.Response.URL)
}
```

To read two kinds of event, the caller starts two iterators and merges them
with a `chromedp` helper:

```go
for ev, err := range chromedp.Merge(
	chromedp.Map(requests, func(e network.RequestWillBeSent) Seen { return Seen{URL: e.Request.URL} }),
	chromedp.Map(responses, func(e network.ResponseReceived) Seen { return Seen{URL: e.Response.URL, Status: e.Response.Status} }),
) {
	// ev is a Seen.
}
```

### Poll until a condition holds

An iterator can express a retry loop without a callback:

```go
for attempt, err := range chromedp.Poll(ctx, 100*time.Millisecond) {
	if err != nil {
		return err
	}
	n, err := chromedp.Run(ctx, s, dom.Count(".row"))
	if err != nil {
		return err
	}
	if n >= 10 {
		break
	}
	_ = attempt
}
```

### An error from the browser

```go
_, err := cdp.Call(ctx, s.Page(), dom.QuerySelector, dom.QuerySelectorParams{
	NodeID:   root,
	Selector: "(",
})
var perr *cdproto.Error
if errors.As(err, &perr) {
	fmt.Println(perr.Code, perr.Message)
}
```

### Options

A command takes a struct. A zero field is omitted from the message, because
the field has `omitzero`. The generator does not write the `With...` methods. An
optional boolean is a `*bool`, so that a caller can send `false`. For another
type, a field that must be sent when it is zero is an open question below. A
field that holds an enum has the named type of that enum, so a caller cannot
pass an arbitrary string by accident.

### Old code keeps working

`chromedp.Legacy` adapts a value that has the old `Do(context.Context) error`
method. It puts the session where the old code expects to find it:

```go
func Legacy(a OldAction) Action[struct{}] {
	return func(ctx context.Context, s *Session) (struct{}, error) {
		return struct{}{}, a.Do(withSession(ctx, s))
	}
}

_, err := chromedp.Run(ctx, s, chromedp.Legacy(chromedp.Navigate(url)))
```

This is the migration path. A project moves one call at a time.

## What it costs

- Every name changes. `page.Navigate(url).Do(ctx)` becomes
  `cdp.Call(ctx, s, page.Navigate, page.NavigateParams{URL: url})`. The new call
  is longer. The old chain read well, but a free function cannot chain.
- The generator writes a result struct for every command. `cdproto` grows.
- `cdp.Call` uses reflection through the JSON package. A hot path can need
  generated code for each command. That makes the API larger.
- An iterator that nobody reads can fill a buffer. The session must drop events
  for a slow reader, or close it, and say so in its documentation.
- The `With...` methods let the caller write one expression. A struct literal
  needs a variable for a long list of fields. Some people will prefer the old
  form.

## Open questions

The branch answers three earlier questions: where the core lives, whether the
old API stays, and what to do about the name collision. The next section gives
the answers.

1. How does a caller send a zero value that the protocol requires, for a type
   other than a boolean? Today the generator has a few of these in a hand
   written list, which the rewrite rule removed. A pointer field is one answer.
2. Does the protocol need a `Seq2` for every event, or is `Seq` enough when the
   caller cancels with the context?

## What the branch generates

The branch `typed-api` of this repository has the `cdproto` side. A few points
differ from the first sketches above. The sketch of the `chromedp` side does not
change because of them.

- The core is in the `cdp` package, because the generated packages already
  import it. `Session` is an interface with two methods, `Call` and `Subscribe`,
  so the connection stays in `chromedp`. `cdp.Call` and `cdp.Events` are the
  functions that the examples use.
- A result struct is named `FooResult`. A command with no parameters or no
  results uses `cdp.Empty`, and has no struct for them.
- The old `Do` method, the old function for each command and the `With...`
  methods are gone. A generated package cannot hold both, because the value
  `page.Navigate` and the old function `page.Navigate` have the same name. The
  old executor in the context is gone as well.
- An optional boolean in the parameters of a command is a `*bool`. A plain
  `bool` cannot say "leave it out". The protocol has about ten optional
  booleans whose default is true, so a zero `false` changes what the browser
  does. In Go 1.26 and later, `new(true)` and `new(false)` make the pointer.
- A binary value is a `[]byte`. The standard library encodes it as base64
  without a tag, so the base64 step of the old `Do` method is gone.
- `cdp.Events` buffers from the moment it returns. A caller that never ranges
  over the iterator holds the subscription, and the documentation of the
  function says so.
- `gencmd/gencmd_test.go` generates a small protocol and runs
  `gencmd/testdata/generated_test.go.txt` against it with the go command. The
  test covers a command with an enum, a pointer boolean and a binary result, a
  command with nothing, an error, the order of events and the end of an
  iterator.

## What changes in this repository

The work of the branch is these changes, and each is a commit:

- `gen/gotpl/extra.tmpl` writes the `cdp` package of this document.
- `gen/gotpl/domain.tmpl` writes the parameter and result structs, the command
  value and the event value, and does not write `Do` or the `With...` methods.
- `gen/gotpl/util.go` writes `[]byte` and `*bool` where the rules above say.
- `docs/GENERATOR.md` describes the new files.

Nothing is merged to `main` until the maintainer decides.
