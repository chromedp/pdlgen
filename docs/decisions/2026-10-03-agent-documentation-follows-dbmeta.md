# The agent documents follow the `dbmeta` layout, with dated decisions

Status: Decided.

The maintainer decided on 2026-10-03 to prepare this repository for coding agents in the
way of the `xo/dbmeta` repository. The `chromedp/chromedp` repository got the
same treatment.

## What was done

The repository has an `AGENTS.md` for the rules, a one line `CLAUDE.md` that
imports it, a `CONTRIBUTING.md` for a person, and a `docs/` directory with a
plan, a backlog, a progress file and a file for each decision. The two skills,
`go-pedantry` and `simple-english`, are committed as copies under
`.agents/skills` and `.claude/skills`.

## The difference from `dbmeta`

The template is not copied byte for byte. The text is written for this project,
and the parts that belong to databases are not here. One difference is on
purpose: the `chromedp` projects name a decision with a date and a title, as in
`2026-10-03-templates-are-standard-go-templates.md`, and not with a `D` and a
number. A decision refers to another by its file name. The file opens with its
title and its status, and `docs/decisions/README.md` is the index.

## Why a copy of the skills

A symbolic link in a Windows checkout becomes a text file, and then Claude Code
loads no skill and says nothing. A copy always works. `docs/docs_test.go` fails
when the two folders differ or when one holds a link.
