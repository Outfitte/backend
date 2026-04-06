---
name: implement-task
description: Use this skill when the user invokes /implement-task or asks to implement a GitHub issue, work on a task/ticket, or start working on an issue by number. Always use this skill when the user says "implement task", "work on issue #N", "/implement-task", or similar.
---

# Implement Task

Full workflow for implementing a GitHub issue from start to PR review.
Follow all branch, commit, and PR conventions from CLAUDE.md.

## Step 1 — Get the issue number

If the user didn't provide an issue number, ask: "Which issue number should I work on?"

## Step 2 — Read the issue

```bash
gh issue view <number> --repo Outfitte/Outfitte
```

Read the full issue body and any linked context before proceeding.

> **Network failures:** For any git or gh CLI command that contacts the network
> (git pull, git push, gh pr create, etc.), retry up to 3 times with a 5-second
> delay between attempts before giving up.

## Step 3 — Sync main

```bash
git checkout main && git pull
```

## Step 4 — Create branch

Use the branch naming convention from CLAUDE.md. Derive the short name (2–4 lowercase hyphenated words) from the issue title.

```bash
git checkout -b <username>/<number>-short-name
```

To get the current git username: `git config user.name` (slugify if needed).

## Step 5 — Implement

Read relevant existing code before writing. Follow the architecture rules in CLAUDE.md. Keep changes minimal and focused on the issue scope.

If the task involves writing or modifying code, load and follow the `tdd` skill **before writing any tests or production code**. The TDD loop must be followed strictly: write one failing test, make it pass with minimal code, then move to the next test. Never write multiple tests upfront, and never implement the full body before each individual test is red.

## Step 6 — Commit

Stage only the files changed for this issue. Follow the commit message format from CLAUDE.md.

```bash
git add <files>
git commit -m "<number>: one sentence message"
```

## Step 7 — Push and open PR

```bash
git push -u origin <branch>
```

Follow the PR title format from CLAUDE.md. PR body must include `Closes #<number>`.

```bash
gh pr create --title "<number>: short description" --body "$(cat <<'EOF'
## Summary
- <bullet points>

## Test plan
- [ ] <verification steps>

Closes #<number>

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

## Step 7b — Post coverage report as PR comment

Identify which Go packages were modified from the staged files, then run coverage for those packages only:

```bash
go test -coverprofile=coverage.out ./internal/<modified-packages>/...
go tool cover -func=coverage.out
```

Coverage checks apply **exclusively to the modified packages** — unmodified packages are irrelevant to this task.

Extract the total coverage percentage from the last line of `go tool cover -func` output (e.g. `total: (statements) 87.5%`).

### If total coverage is 100%

Post the standard comment:

```bash
gh pr comment <pr_number> --repo Outfitte/Outfitte --body "$(cat <<'EOF'
## Coverage report
\`\`\`
<coverage output>
\`\`\`
EOF
)"
```

### If total coverage is less than 100%

1. Review the `go tool cover -func` output to identify every function below 100%
2. Read the relevant source lines to understand what is not covered
3. **If any uncovered lines can be covered with additional tests** — add those tests (return to the TDD loop), then re-run coverage and repeat this check
4. Only after exhausting all feasible tests, post a comment that includes both the coverage output AND a mandatory justification section — one entry per under-covered function:

```bash
gh pr comment <pr_number> --repo Outfitte/Outfitte --body "$(cat <<'EOF'
## Coverage report
\`\`\`
<coverage output>
\`\`\`

## Why coverage is not 100%

- `internal/adapter/store/json/foo.go: Bar()` — 85.7%: the `os.WriteFile` error path requires OS-level fault injection and cannot be triggered in unit tests
- ...one entry per under-covered function...
EOF
)"
```

A bare coverage report without a `## Why coverage is not 100%` section is **not acceptable** when coverage is below 100%.

## Step 8 — PR review (background)

Spawn two **background** subagents in parallel: a `pr-reviewer` (`subagent_type: "pr-reviewer"`) and a `Code Reviewer` (`subagent_type: "Code Reviewer"`). Do not pass file contents — let them fetch from GitHub.

### pr-reviewer subagent

Spawn with `subagent_type: "pr-reviewer"`. Do not pass file contents — let it fetch from GitHub:

```
Review PR #<pr_number> in the Outfitte/Outfitte repo.

Run:
  gh pr view <pr_number> --repo Outfitte/Outfitte
  git diff main

## Architecture checks (CLAUDE.md)
- Branch, commit, and PR title follow naming conventions
- Domain stays pure: internal/domain and internal/ports never import from adapters or api
- context.Context is first arg in every method, named ctx, never blanked with _
- ctx.Err() is checked before any lock or I/O at the start of functions
- Infrastructure errors are wrapped with domain sentinels at the adapter boundary (e.g. fmt.Errorf("%w: %w", domain.ErrIO, err)) — raw os/encoding errors must not leak to callers

## TDD checks
- Failure/error case tests come before happy-path tests
- New methods start from an `errors.New("not implemented")` stub (evidenced by commit history if available)
- Error assertions use require.ErrorIs against domain sentinels — not require.ErrorContains against stdlib message strings
- Test entities use explicit IDs (not zero values) where identity matters
- t.Context() is used instead of context.Background() in tests
- Extracted helpers (pure functions) have 100% coverage independently

Report any violations, concerns, or suggestions. If everything looks good, say so.
```

Report the review findings to the user once complete.

### Code Reviewer subagent

Spawn with `subagent_type: "Code Reviewer"`. Pass the following prompt so it knows where to find the code:

```
Review the code changes in PR #<pr_number> of the Outfitte/Outfitte GitHub repo.

Fetch the diff using:
  gh pr view <pr_number> --repo Outfitte/Outfitte
  git diff main

Focus on correctness, security, maintainability, and performance. Report any blockers, suggestions, or nits. If everything looks good, say so.
```

Report the review findings to the user once complete.

## Step 9 — Capture learnings

After the PR review is done, reflect on what happened during this implementation session. Ask: **did anything non-obvious come up that would help future sessions?**

Examples of things worth capturing:
- A pattern or convention discovered from reading existing code (e.g. how errors are wrapped, how tests are structured)
- A decision made that isn't obvious from the code (e.g. why a certain approach was chosen over another)
- A gotcha or edge case that caused confusion or required backtracking
- A new rule that emerged from PR review feedback

### What to update

- **CLAUDE.md** — for project-wide conventions, handler guidelines, architecture rules, or patterns that every future implementation should follow
- **Memory files** — for patterns specific to a sub-system (e.g. JSON adapter), feedback about how to collaborate, or anything that isn't a hard project rule

If nothing non-obvious came up, skip this step — do not invent learnings.

### How to update CLAUDE.md

Add the learning under the most relevant existing section, or create a new section if needed. Keep entries concise (1–3 lines). Do not duplicate what is already documented.

```bash
# Read CLAUDE.md first, then Edit to add the new entry
```

### How to update memory

Use the Write tool to create or update the relevant memory file in `/home/maxime/.claude/projects/-home-maxime-Development-Outfitte/memory/`, then update `MEMORY.md` with a pointer if it's a new file.
