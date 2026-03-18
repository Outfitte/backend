---
name: pr-comment-fix
description: Apply a fix requested in a PR inline review comment. Use when invoked via the claude-pr-comment workflow with a PR number and comment body.
---

# PR Comment Fix

Workflow for applying a fix requested via an inline PR review comment.

## Step 1 — Read the PR

```bash
gh pr view <pr_number> --repo Outfitte/Outfitte
gh pr diff <pr_number> --repo Outfitte/Outfitte
```

Read the full PR description and diff before proceeding.

> **Network failures:** For any git or gh CLI command that contacts the network
> (git pull, git push, gh pr comment, etc.), retry up to 3 times with a 5-second
> delay between attempts before giving up.

## Step 2 — Check out the PR branch

```bash
gh pr checkout <pr_number> --repo Outfitte/Outfitte
```

## Step 3 — Understand the comment

Parse the comment body to extract the concrete action requested. If the comment references a specific file and line, read that file and locate the relevant code. Identify exactly what needs to change.

If the requested change is ambiguous or unclear, post a PR comment asking for clarification and stop — do not proceed to Step 4.

## Step 4 — Implement the fix

Load and follow the `tdd` skill **before writing any tests or production code**. The TDD loop must be followed strictly: write one failing test, make it pass with minimal code, then move to the next test. Never write multiple tests upfront, and never implement the full body before each individual test is red.

Read relevant existing code before writing. Follow the architecture rules in CLAUDE.md. Keep changes minimal and focused on the comment's request.

## Step 5 — Commit

Stage only the files changed for this fix.

```bash
git add <files>
git commit -m "<pr_number>-fix: one sentence message"
```

Message format uses `<pr_number>-fix:` prefix since there is no issue number.

## Step 6 — Push

```bash
git push
```

The branch is already tracked. Retry up to 3 times with a 5-second delay on network failure.

## Step 7 — Post coverage report

Skip this step if no Go files were modified (e.g. the fix only touched YAML or Markdown).

Identify which Go packages were modified from the staged files, then run coverage for those packages:

```bash
go test -coverprofile=coverage.out ./internal/<modified-packages>/...
go tool cover -func=coverage.out
```

Post the output as a PR comment:

```bash
gh pr comment <pr_number> --repo Outfitte/Outfitte --body "$(cat <<'EOF'
## Coverage report
\`\`\`
<coverage output>
\`\`\`
EOF
)"
```

## Step 8 — Reply to the original comment

Post a threaded reply directly on the inline review comment that triggered this workflow, so the commenter receives a notification in context:

```bash
gh api repos/Outfitte/Outfitte/pulls/<pr_number>/comments/<comment_id>/replies \
  --method POST \
  --field body="Fixed in <commit_sha>: <one sentence describing what was changed>."
```

Use the `Comment ID` value provided in the prompt as `<comment_id>`.
