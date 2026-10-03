# Contributing to pdlgen

Read three things before you change anything.

[`AGENTS.md`](AGENTS.md) holds the rules. It is written for a coding agent, and
everything in it applies to a person. `CLAUDE.md` holds one line that imports
it, for Claude Code.

[`docs/GENERATOR.md`](docs/GENERATOR.md) describes each step from a protocol
definition to a Go file, and how to change a template without changing the
output by accident.

[`docs/decisions/`](docs/decisions/README.md) holds every decision of this
project, one file each, named by date. Read the status of a decision before you
read the decision.

Do not decide an open question on your own. The open questions are at the end
of [`docs/PLAN.md`](docs/PLAN.md). Ask the maintainer.

## Issues for cdproto

Report a problem in the `cdproto` package here, in this repository. The
`cdproto` repository holds generated output, and the next daily run replaces
any change made there.

## Before you send a change

```bash
gofmt -l . && go vet ./... && go build ./... && go test -race -count=2 ./...
go mod tidy && git diff --exit-code go.mod go.sum
```

`gofmt -l .` must print nothing.

If you changed a template or the generator, compare the generated code from
before and after your change. The output must be the same unless the output is
what you changed. [`docs/GENERATOR.md`](docs/GENERATOR.md), under Changing a
template, gives the steps.

## Releases

You do not make a release. The `Update` workflow regenerates `cdproto` each day
and tags it. [`docs/RELEASES.md`](docs/RELEASES.md) describes the tags.

## Agent skills

The repository carries two agent skills. A skill is a set of instructions that
a coding agent loads for a task. `simple-english` sets how prose is written,
and `go-pedantry` sets how Go is written. `docs/docs_test.go` checks the rules
of `simple-english` that a machine can check, in the Markdown files and in the
Go comments. If it reports a sentence, rewrite the sentence.

`skills-lock.json` names the source of each skill. The `skills` command from npm
writes it, and writes each skill into two folders. Codex and the other agents
read `.agents/skills/<name>`, and Claude Code reads `.claude/skills/<name>`.

To add a skill or to update one, run this in the repository root. The example
updates `simple-english`:

```bash
npx skills@1.7.0 add AminBlg/SimpleEnglish --skill simple-english --agent codex claude-code --copy -y
```

Keep `--copy`. Without it, the command writes `.claude/skills/<name>` as a
symbolic link. A Windows checkout writes a symbolic link as a text file, and
Claude Code then loads no skill and says nothing. The test fails on a link and
when the two folders differ.

`.claude/settings.local.json` holds the Claude Code permissions of one person.
The `.gitignore` ignores it.

## What the reviewer will ask

Is the output the same as before, when the change is a refactor? Does the
generated code build and pass `go vet`? Is the new rule in the generator one
that the protocol needs, or is it a convenience that belongs in `chromedp`?
Does a new document have a row in the tables of `AGENTS.md` and `README.md`?
