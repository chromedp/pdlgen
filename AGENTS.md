# pdlgen

`pdlgen` generates the Go package `github.com/chromedp/cdproto`. The
package holds the commands, events and types of the Chrome DevTools Protocol.
`chromedp` is the main consumer of `cdproto`.

`cdproto` is output. Nobody edits it by hand. A workflow in this repository
regenerates it every day, commits the result and tags a release. Every issue
and every pull request for `cdproto` belongs here, and none belongs in the
`cdproto` repository.

The generated code must be regular. A reader who knows how one command is
generated knows how all of them are. The generator makes two rewrites, and no
others. A rewrite changes a name or a type that the protocol defines. The first
rewrite removes name stuttering, so `css.CSSStyle` becomes `css.Style`. The
second gives each inline enum a named type with a constant for each value. An
inline enum is an enum that a property or a parameter declares in place. See
`docs/decisions/2026-10-03-the-only-fixup-removes-name-stuttering.md` and
`docs/decisions/2026-10-03-inline-enums-are-named-types.md`.

## Standing rules

These hold in every `chromedp` repository, for every coding agent.

1. Stage changes for review. Commit and push only when the maintainer says so.
2. Load the `simple-english` skill before you write any text that a person
   reads: a document, a code comment, an error message or a commit message.
   Follow it for that text.
3. Load the `go-pedantry` skill before you write or review Go code. Follow it
   where it does not conflict with a rule in this file. A rule here wins.

`CLAUDE.md` holds one line that imports this file, so that Claude Code and
every other agent read the same rules. Edit this file, not that one.

## Which document to read

Start with `docs/GENERATOR.md`. It describes each step from a protocol
definition to a Go file. `docs/decisions/` holds every decision, one file each,
and `docs/decisions/README.md` is the index. Read the status of a decision
before you read the decision, because a later one can amend or replace it. Do
not decide an open question on your own. The open questions are at the end of
`docs/PLAN.md`. Ask the maintainer.

Then by what you are doing:

| If you are | Read |
| --- | --- |
| changing a template | `docs/GENERATOR.md`, under Changing a template |
| changing how a protocol definition is read | `docs/GENERATOR.md`, under The pipeline |
| adding a rewrite to the generated code | `docs/decisions/2026-10-03-the-only-fixup-removes-name-stuttering.md`, then ask the maintainer |
| changing the output of a generated type | `docs/GENERATOR.md`, then tell the maintainer that `chromedp` is affected |
| changing how `cdproto` is versioned or tagged | `docs/RELEASES.md`, then `docs/decisions/2026-10-03-cdproto-is-tagged-v0-chromium-major-patch.md` |
| changing or using the typed API of the protocol | `docs/API.md`, which describes what the generator writes |
| looking for the purpose, the architecture, the tests or the open questions | `docs/PLAN.md` |
| changing a workflow | `docs/RELEASES.md`, under The workflows |
| asking why something is the way it is | the index in `docs/decisions/README.md` |
| looking for work that is known and not done | `docs/BACKLOG.md` |
| resuming a session that ended or crashed | `docs/PROGRESS.md`, which says where the work stands |
| writing a document, a code comment, an error message or a commit message | the `simple-english` skill. Load it first |
| writing or reviewing Go code | the `go-pedantry` skill. Load it first |
| adding or updating an agent skill | `CONTRIBUTING.md`, under Agent skills |

`CONTRIBUTING.md` is the same thing for a person, and shorter.

A document that is not in that table does not exist. If you cannot find where
something is written down, it is not written down, and it is an open question.

## Hard rules

1. Never edit a file in a `cdproto` checkout by hand, and never commit a
   generated file here. Fix the generator, then regenerate. A hand edit is
   lost on the next daily run.
2. Do not add a rewrite to the generated code. The only two are the name
   stuttering fix and the inline enum extraction, both in `fixup/`. A helper
   method, a convenience type, a renamed field or a changed type belongs in
   `chromedp`, not here. A change that only lets the code compile is not a
   rewrite. Moving a type into the `cdp` package to break an import cycle is not
   a rewrite either, because it only decides where the code goes. The list is
   `pdl/dep.go`.
3. Templates are standard `text/template` files in `gen/gotpl/*.tmpl`, and the
   binary embeds them. Do not add a template engine, and do not add a step that
   turns templates into Go.
4. A change to a template or to the generator must not change the output unless
   the output is what you are changing. Generate with a fixed protocol file
   before and after, and compare the two trees. `docs/GENERATOR.md` gives the
   steps. A refactor must give an identical tree.
5. A test never uses the network. The protocol files come from a string in the
   test or from a file in `testdata`, and never from the Chromium source tree.
6. The generated code must build and pass `go vet`. The `Test` workflow
   generates `cdproto` from the latest protocol and builds it. Do not merge a
   change that fails that job.
7. `cdproto` stays at major version 0. The minor version is the Chromium major
   version, and the patch version counts the releases for it. Only the
   `Update` workflow makes a tag, and it makes none when only the Chromium or
   V8 version changed. Never tag by hand unless the maintainer says
   so. Never move or delete a tag that you pushed. If a release is bad, retract
   it in the `go.mod` of `cdproto` and release a new one. See
   `docs/RELEASES.md`.
8. The `Update` workflow commits as the repository owner. Do not change the
   author. Do not add an author line for a coding agent to a commit that the
   workflow makes.
9. Never write a token, a key or a password in a file, a log or a message. The
   workflow reads `ACCESS_TOKEN` from the secrets of the repository, and a
   file names it and nothing more.
10. Keep the dependencies few. `go.mod` must be tidy, and the `Test` workflow
    fails when it is not. Ask the maintainer before you add a package.
11. The protocol files are cached in a directory on the machine that runs the
    generator. Do not commit a protocol file, except a small one in `testdata`.

12. The generated code must build with `go 1.25`. `pdlgen` itself uses the
    current version of Go, and its `go.mod` follows its dependencies. Do not write
    `new(expr)` or another feature of a later Go version. Only
    `cdp/jsonv2/json_std.go` and `cdp/jsonv2/json_compat.go` import a JSON
    package, and every other generated file uses `jsonv2.Value`,
    `jsonv2.Unmarshal` and the other names of the package `cdp/jsonv2`. That
    package imports no other package of the module. A change to one of the two
    files must change the other one in the same way. Test both with `go test ./...` and with
    `GOEXPERIMENT=nojsonv2 go test -tags cdproto_jsoncompat ./...`. See
    `docs/decisions/2026-10-06-the-subpackage-cdp-jsonv2-hides-the-json-package.md`.

## Layout

- `main.go` starts the command with `ox`, and holds nothing else. `gencmd/`
  holds the command: the `Args` struct, whose `ox` tags define the flags, and
  the code that loads the protocol, calls the rewrites and the generator, and
  writes the files.
- `grab.go` is a program that you start with `go run grab.go`. It caches the
  combined protocol files of the recent Chromium releases. The build tag
  `ignore` keeps it out of `go build`.
- `pdl/` parses the protocol definition language and holds the types that
  describe a domain. `pdl/dep.go` lists the types that move to the `cdp`
  package. `pdl/har.go` holds the HAR domain, which the protocol files do not
  define, and `pdl/gen.go` writes it.
- `fixup/` removes name stuttering and extracts inline enums into named types.
  It is small on purpose.
- `gen/` holds the Go generator. `gen/gotpl/` holds the templates and the
  functions they call, and `gen/genutil/` holds the comment and name helpers.
- `util/` retrieves and caches the files, and compares versions.
- `relnotes/` and `cmd/relnotes/` write the notes of a release from the output
  of `apidiff`, for the commit, the tag and the changelog.
- `diff/` prints the difference between two protocol files.
- `.github/workflows/` holds the `Test` and `Update` workflows.
- `.github/scripts/update.sh` regenerates `cdproto`, and makes the tag and the
  commit.
- `docs/` holds the documents. `docs/decisions/` holds the decisions.
- `.agents/skills/` and `.claude/skills/` hold the two agent skills, as copies.
  `skills-lock.json` names their sources.

## Go conventions

Wrap every error with `%w`, never `%s` or `%v`:

```go
if err != nil {
	return nil, fmt.Errorf("reading %s: %w", name, err)
}
```

Write error messages in lower case, starting with a gerund. Do not write
"failed to" or "error". Name the object that failed.

Accept interfaces and return concrete structs. Keep an interface to three
methods or fewer. Define an interface where it is consumed.

Put `context.Context` first in every parameter list and name it `ctx`. Never
store it in a struct.

Define a flag as a field of `gencmd.Args` with an `ox` tag, and set its default
in `gencmd.New`. A default in a tag cannot hold a comma or a duration. Do not
read a flag from a package level variable.

Name a package with one short lower case word. Do not stutter. Name a receiver
with one or two lower case letters, and use the same name on every method of
the type. Never write `this` or `self`.

Group struct fields by purpose, with exported fields first and a blank line
between groups.

A template function that the templates call lives in `gen/gotpl/gotpl.go` or
`gen/gotpl/util.go`, and `funcMap` is the only place that names them. A
template must not hold logic that a function can hold. Put a condition or a
loop in the template, and put a computed value in a function.

Test the generator with a small protocol file in the test, as `gen/gen_test.go`
does. Do not test it against the real protocol, because that needs the network.

## Linting

No linter configuration exists yet. `go vet ./...` and `gofmt -l .` are the
checks, and the `Test` workflow runs both. Adding `golangci-lint` is in
`docs/BACKLOG.md`. Until it exists, follow the `go-pedantry` skill by reading.

## Before you commit

Run these:

```bash
gofmt -l . && go vet ./... && go build ./... && go test -race -count=2 ./...
go mod tidy && git diff --exit-code go.mod go.sum
```

`gofmt -l .` must print nothing.

If you changed a template or the generator, also compare the output. The steps
are in `docs/GENERATOR.md`, under Changing a template. The comparison must show
no difference unless the output is what you changed.

## Writing documentation

A new document goes in `docs/`. Only `README.md`, `AGENTS.md`, `CLAUDE.md`,
`CONTRIBUTING.md` and `LICENSE` belong in the repository root, and a test
enforces that. Add the document to the table under Which document to read. Add
it to the table in `README.md` too. A document that nobody can find is a document that
nobody reads.

A decision goes in a file of its own in `docs/decisions/` and nowhere else.
Name it `YYYY-MM-DD-short-title.md` with the date of the decision. Do not give
a decision a number. The file opens with `# <Title>`, a blank line and
`Status: <status>.`. The status is `Decided`, `Proposed`, `Open`,
`Amends <file>`, `Amended by <file>` or `Superseded by <file>`. Refer to a decision by its file
name. Add its row to `docs/decisions/README.md`. If your decision changes part
of an earlier one, the new status is `Amends <file>` and the old status is
`Amended by <file>`. If it replaces an earlier one, the new status is `Decided`
and its first sentence names the old file. The old status is
`Superseded by <file>`. A reader who finds the older one must be told.

Write plain English. Use short sentences and the active voice. Use `can`,
`will`, and `must`, and do not use `should`, `may`, or `might`. Do not use
semicolons or em dashes. Put the condition before the command: "If the build
fails, read the log."

Load the `simple-english` skill before you write any text that a person reads.
Follow it for that text. Its rules include the ones above and add more, such as
no contractions and one word for one meaning. The maintainer asked for this.

`docs/docs_test.go` checks the rules of the skill that a machine can check. It
checks the Markdown files and the comments of the Go files that we write. It
also checks the links, the document tables, the decision index and the skill
copies. If it reports a sentence, rewrite the sentence. Sentence length and the voice are still yours
to check.

Work that is known and not done goes in `docs/BACKLOG.md`, with the decision or
the measurement that found it. When an item is done, delete it, and record
anything decided in `docs/decisions/`.
