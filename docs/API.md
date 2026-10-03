# A typed API for the protocol

Status: Proposed. Nobody has chosen this design, and none of the code below
compiles. The names and the shapes can change. This document records the idea
so that the maintainer can decide, and it shows how `chromedp` can use the result. See
`docs/decisions/2026-10-03-proposed-generics-and-iterators-api.md`.

## The problem

The code that `pdlgen` writes today has four weaknesses.

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

For the command `Page.navigate` in the package `page`:

```go
// NavigateParams are the parameters of Page.navigate.
type NavigateParams struct {
	URL            string         `json:"url"`
	Referrer       string         `json:"referrer,omitzero"`
	TransitionType TransitionType `json:"transitionType,omitzero"`
	FrameID        cdp.FrameID    `json:"frameId,omitzero"`
}

// NavigateResult is the result of Page.navigate.
type NavigateResult struct {
	FrameID    cdp.FrameID  `json:"frameId"`
	LoaderID   cdp.LoaderID `json:"loaderId,omitzero"`
	ErrorText  string       `json:"errorText,omitzero"`
	IsDownload bool         `json:"isDownload,omitzero"`
}

// Navigate is the command Page.navigate.
var Navigate = cdp.Command[NavigateParams, NavigateResult]{Method: "Page.navigate"}

// LoadEventFired is the event Page.loadEventFired.
var LoadEventFired = cdp.Event[LoadEventFiredEvent]{Method: "Page.loadEventFired"}
```

A command that takes no parameters uses `cdp.Empty` for `P`, and a command
that returns nothing uses `cdp.Empty` for `R`. A result struct is always
generated, even for a command that returns one value, so that adding a second
value later is not a break.

### The core package

The `cdp` package holds about forty lines of code. The sketch leaves out the
connection:

```go
package cdp

// Empty is the parameters or the result of a command that has none.
type Empty struct{}

// Command is a protocol command with parameters P and result R.
type Command[P, R any] struct{ Method string }

// Event is a protocol event with the payload E.
type Event[E any] struct{ Method string }

// Session is one connection to a browser target.
type Session struct { /* connection, pending calls, listeners */ }

// Call runs the command on the session.
func Call[P, R any](ctx context.Context, s *Session, c Command[P, R], p P) (R, error) {
	var res R
	err := s.call(ctx, c.Method, p, &res)
	return res, err
}

// Events returns the events of one kind. The session starts to buffer them
// when Events returns, and not when the caller starts to range.
func Events[E any](ctx context.Context, s *Session, e Event[E]) iter.Seq2[E, error] {
	ch := s.listen(e.Method)
	return func(yield func(E, error) bool) {
		defer s.unlisten(e.Method, ch)
		for {
			select {
			case <-ctx.Done():
				var zero E
				yield(zero, ctx.Err())
				return
			case v := <-ch:
				if !yield(v.(E), nil) {
					return
				}
			}
		}
	}
}
```

A protocol error from the browser is a `*cdp.Error`. A caller reads it with
`errors.As`.

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
var perr *cdp.Error
if errors.As(err, &perr) {
	fmt.Println(perr.Code, perr.Message)
}
```

### Options

A command takes a struct. A zero field is omitted from the message, because
the field has `omitzero`. The generator no longer writes the `With...` methods.
A field that must be sent even when it is zero is a pointer, or a type from
`cdp` that says so. This is an open question below. A field that holds an enum
has the named type of that enum, so a caller cannot pass an arbitrary string by
accident.

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

1. Where do `Command`, `Event`, `Session` and `Call` live? `cdproto` can hold
   them, because the generated code needs the types. A session needs a
   connection, and `cdproto` has none today. The connection can stay in
   `chromedp`, and `cdproto` can define an interface:

   ```go
   type Session interface {
   	Call(ctx context.Context, method string, params, result any) error
   	Listen(method string) (<-chan jsontext.Value, func())
   }
   ```

   `Call` takes that interface instead of `*Session`.
2. How does a caller send a zero value that the protocol requires? Today the
   generator has a few of these in a hand written list, which the fixup rule
   removed. A pointer field is one answer.
3. Does the old API stay in `cdproto` for a time, so that both can be used
   in one program? That doubles the size of the generated code.
4. How is a name chosen when a command and its parameter type share a name?
   Today the command is `Navigate` and the parameters are `NavigateParams`. The
   variable `page.Navigate` collides with a function of the same name, so
   the old and the new cannot live in one package.
5. Does the protocol need a `Seq2` for every event, or is `Seq` enough when the
   caller cancels with the context?

## What changes in this repository

- `gen/gotpl/domain.tmpl` writes the parameter, the result and the command
  variable, and does not write `Do` or the `With...` methods.
- `gen/gotpl/type.tmpl` writes an event as a payload type and an `Event`
  variable.
- `gen/gotpl/extra.tmpl` writes the `cdp` package of this document.
- `docs/GENERATOR.md` describes the new files.

Nothing in this list starts until the maintainer decides.
