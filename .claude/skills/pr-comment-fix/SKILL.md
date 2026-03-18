---
name: pr-comment-fix
description: Apply fixes requested via a PR review submission or a top-level PR comment. Use when invoked via the claude-pr-comment workflow with a Trigger field in the prompt.
---

# PR Comment Fix

Workflow for applying fixes requested via a PR review or a top-level PR comment.

> **Network failures:** For any git or gh CLI command that contacts the network
> (git pull, git push, gh pr comment, etc.), retry up to 3 times with a 5-second
> delay between attempts before giving up.

## Step 1 — Read the PR

```bash
gh pr view <pr_number> --repo Outfitte/Outfitte
gh pr diff <pr_number> --repo Outfitte/Outfitte
```

Read the full PR description and diff before proceeding.

## Step 2 — Check out the PR branch

```bash
gh pr checkout <pr_number> --repo Outfitte/Outfitte
```

## Step 3 — Understand what needs to be fixed

The approach depends on the `Trigger` value in the prompt:

### Trigger: `review_submitted`

Fetch all inline comments from the review:

```bash
gh api repos/Outfitte/Outfitte/pulls/<pr_number>/reviews/<review_id>/comments
```

Each inline comment has a `path` (file), `line`, `original_line`, and `body` describing what needs to change. Read each referenced file at the indicated line and understand the requested fix. Collect all requested changes — they will all be applied in a single commit.

If any inline comment is ambiguous, post a PR comment asking for clarification and stop — do not proceed to Step 4.

### Trigger: `pr_comment`

The `Comment body` in the prompt describes what needs to change. Read the relevant files and locate the code to fix. If the request is ambiguous, post a PR comment asking for clarification and stop.

## Step 4 — Implement the fix

Load and follow the `tdd` skill **before writing any tests or production code**. The TDD loop must be followed strictly: write one failing test, make it pass with minimal code, then move to the next test. Never write multiple tests upfront, and never implement the full body before each individual test is red.

Read relevant existing code before writing. Follow the architecture rules in CLAUDE.md. Keep changes minimal and focused on the requested fixes.

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

## Step 8 — Acknowledge the fix

### Trigger: `review_submitted`

Reply to the submitted review by posting a top-level PR comment summarising what was fixed:

```bash
gh pr comment <pr_number> --repo Outfitte/Outfitte --body "$(cat <<'EOF'
Fixed in <commit_sha>: <one sentence per inline comment addressed>.
EOF
)"
```

### Trigger: `pr_comment`

Post a threaded reply on the original comment:

```bash
gh api repos/Outfitte/Outfitte/issues/comments/<comment_id>/replies \
  --method POST \
  --field body="Fixed in <commit_sha>: <one sentence describing what was changed>."
```

Use the `Comment ID` value from the prompt as `<comment_id>`.
