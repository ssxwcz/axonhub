---
description: "Workflow for merging upstream PRs into a fork: commit attribution, PR references, signing"
---

# Merging Upstream PRs into This Fork

When merging upstream `looplj/axonhub` PRs into this fork (`review/upstream-prs`), follow these rules so the history stays clean, attributable, and link-free.

## 1. PR / Issue References (avoid auto-linking)

- **Never** use `#NNN`, `owner/repo#NNN` (or any format GitHub auto-links) in commit title or message — it creates a trail of cross-repo references on the upstream PR/issue pages.
- Use plain text to keep traceability without linking: `(PR NNNN)`, `per PR NNNN`, `upstream PR NNNN`.

## 2. Author / Committer Attribution

- For commits that integrate an upstream PR: set **author = the upstream PR author** (via `git commit --amend --no-edit --author="Name <email>"`), keep **committer = self**. Manual conflict resolution and integration work is attributed through the committer field, never by claiming the author.
- For your own fixes, features, reverts, and chores: author = self (no rewrite needed).
- Query upstream author per PR:
  ```bash
  gh api "repos/looplj/axonhub/pulls/<NNN>/commits" --jq '.[0].commit.author | "\(.name) <\(.email)>"'
  ```
- Rewrite with per-commit author via a `GIT_SEQUENCE_EDITOR` script that injects `exec git commit --amend --no-edit -S --author="..."` after the matching `pick` lines, then `git rebase -i upstream/unstable`.

## 3. GPG Signing

- Every commit must be signed: `git commit -S` (config: `commit.gpgsign = true`, key `55132190BC9FEABD`).
- Any history rewrite (`filter-branch`, `rebase`) **drops signatures** — always re-sign afterwards:
  ```bash
  GIT_SEQUENCE_EDITOR=true GIT_EDITOR=true git rebase --exec 'git commit --amend --no-edit -S' upstream/unstable
  ```

## 4. pnpm Lockfile Pollution

- Local pnpm (v11) rewrites `frontend/pnpm-lock.yaml` (drops the `overrides: uuid` block, bumps many deps). CI uses pnpm 10 with `pnpm install --frozen-lockfile` and will fail on a polluted lockfile.
- After any local pnpm command, check the lockfile and restore it — never commit lockfile changes:
  ```bash
  git checkout -- frontend/pnpm-lock.yaml
  ```

## 5. Verification Before Merging to unstable

- Backend build incl. CI flag: `go build -ldflags "-s -w" -tags=nomsgpack -o ./tmp/axonhub ./cmd/axonhub`, plus `go build ./...`, `cd llm && go build ./...`, `cd cmd/schema && go build`.
- Tests: `make test-backend-all` (root module + `llm/`).
- Lint: `golangci-lint run --timeout 10m --max-same-issues 50` (`.golangci.yml` is v2 — use golangci-lint v2).
- For data migrations on the PostgreSQL deployment, verify the SQL against the live schema types (`channels.settings` is `jsonb`, `updated_at` is `timestamptz`) before shipping.

## 6. Fork Docker Workflows

- Keep docker image targets at `ghcr.io/${{ github.repository_owner }}/axonhub` (owner-scoped, `GITHUB_TOKEN` auth) — never hardcode the upstream `looplj/axonhub` Docker Hub namespace in `docker-publish.yml` / `docker-unstable.yml`.
- `helm-chart.yml` has an `owner == 'looplj'` guard and should stay unchanged.

## 7. Merging the PR into unstable

- Open the PR with base `unstable`, head `review/upstream-prs`.
- Merge with **squash** (`gh pr merge <n> --repo ssxwcz/axonhub --squash`), **never `--merge`** — one integration commit on `unstable`, keeping its history clean. Full per-commit history (original authors, GPG signatures) stays on the review branch.
- CI lint runs against the merge base and catches issues in upstream PR code that local `--new` checks miss (e.g. canonicalheader, misspell). If lint fails, fix on the review branch, re-push, and re-check before merging.
- If the PR already merged with `--merge`, squash afterwards: `git reset --soft <pre-merge tip>` on a branch based on `origin/unstable`, commit once, and `git push --force-with-lease origin <branch>:unstable`.
