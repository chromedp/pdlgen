# Decisions

Every decision of this project is a file in this directory. A file is named by
the date of the decision and its title, and this table is the index.

Each file opens with its title and its status. Decided means the maintainer chose it.
Proposed means somebody suggested it and the maintainer has not chosen. Open means nobody
has chosen. A decision that changes an earlier one says so in its status, as
"Amends 2026-10-03-example.md", and the earlier one says it back, as "Amended by
2026-10-04-example.md". Read the status before you read the decision.

A new decision gets a file of its own, named with the date of the decision.
Add its row here. `docs/docs_test.go` fails when a decision has no row or a row
is wrong, and it prints the row to add.

| Date | Decision | Status |
| --- | --- | --- |
| 2025-02-22 | [The generated code uses go-json-experiment instead of easyjson](2025-02-22-the-generated-code-uses-go-json-experiment.md) | Amended by 2026-10-03-the-generated-code-uses-encoding-json-v2.md |
| 2026-10-03 | [The agent documents follow the `dbmeta` layout, with dated decisions](2026-10-03-agent-documentation-follows-dbmeta.md) | Decided |
| 2026-10-03 | [`cdproto` is tagged v0.<Chromium major>.<patch>](2026-10-03-cdproto-is-tagged-v0-chromium-major-patch.md) | Decided |
| 2026-10-03 | [The `old` branch became `main`, and the old `main` became `wip`](2026-10-03-main-is-the-former-old-branch.md) | Decided |
| 2026-10-03 | [Generate a typed API that uses generics and iterators](2026-10-03-proposed-generics-and-iterators-api.md) | Proposed |
| 2026-10-03 | [The templates are standard Go templates](2026-10-03-templates-are-standard-go-templates.md) | Decided |
| 2026-10-03 | [The command uses ox, and names use ox/strcase](2026-10-03-the-command-uses-ox-and-strcase.md) | Decided |
| 2026-10-03 | [The generated code uses encoding/json/v2 from the standard library](2026-10-03-the-generated-code-uses-encoding-json-v2.md) | Amends 2025-02-22-the-generated-code-uses-go-json-experiment.md |
| 2026-10-03 | [The only fixup removes name stuttering](2026-10-03-the-only-fixup-removes-name-stuttering.md) | Decided |
| 2026-10-03 | [The `Update` workflow regenerates and tags `cdproto` every day](2026-10-03-the-update-workflow-tags-cdproto-daily.md) | Decided |
