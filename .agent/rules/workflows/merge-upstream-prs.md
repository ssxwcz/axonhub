---
description: "Fork maintenance workflow: sync upstream, own features on separate branches, commit attribution and signing"
---

# Fork Maintenance Workflow (follow upstream + own changes)

This fork (`ssxwcz/axonhub`) follows upstream `looplj/axonhub` closely while carrying a few own changes. The previous manual-PR-merge workflow was archived at tag `archive/review-upstream-prs`.

## 0. Maintenance model (how this fork is kept)

- **`unstable` = the follow-upstream mainline.** It equals upstream plus a small, stable set of own changes. Do own work in separate branches, not directly on `unstable`.
- **Sync upstream instead of manually merging PRs.** Upstream merges PRs on its own; you get them for free via sync. Do **not** re-implement upstream PRs by hand.
  ```bash
  git checkout unstable && git pull origin unstable
  git fetch upstream && git merge upstream/unstable
  # resolve conflicts, build + test, then
  git push origin unstable
  ```
- **If you need an upstream PR that is not merged yet**, `git cherry-pick` its commit onto your branch (keeps the original author), and note the source in the message (`from upstream PR NNNN`). When upstream later merges the same PR, sync aligns the content.
- **Own features** go on `feat/<name>` branches cut from a freshly-synced `unstable`, then merge back with squash.
- **Own changes that stay on `unstable`** (re-apply if a sync conflict drops them):
  - `.goreleaser.yml` — homebrew `brews` section removed (upstream tap cannot be written by this fork's token)
  - `docker-publish.yml`, `docker-unstable.yml` — publish to `ghcr.io/<owner>/axonhub` (GHCR, `GITHUB_TOKEN`)
  - `.agent/rules/workflows/merge-upstream-prs.md` — this doc
  - `internal/build/VERSION` — keep aligned with the deployed release (currently `v1.0.0-beta9`)
  - frontend price trim incl. schedule overrides in `channels-model-price-dialog.tsx`
- **Release**: from `unstable`, tag `v<version>` — triggers the Release (binaries) and Docker image workflows (GHCR). Never commit auto-linkable refs.

## 1. PR / Issue References (avoid auto-linking)

- **Never** use `#NNN`, `owner/repo#NNN` (or any format GitHub auto-links) in commit title or message — it creates a trail of cross-repo references on the upstream PR/issue pages.
- Use plain text to keep traceability without linking: `(PR NNNN)`, `per PR NNNN`, `upstream PR NNNN`.

## 2. Author / Committer Attribution

- When cherry-picking or integrating an upstream commit: keep **author = the upstream author**, **committer = self**. Manual conflict resolution and integration work is attributed through the committer field, never by claiming the author.
- For your own fixes, features, reverts, and chores: author = self.
- If a commit was already created with self as author, rewrite per-commit author via a `GIT_SEQUENCE_EDITOR` script that injects `exec git commit --amend --no-edit -S --author="Name <email>"` after the matching `pick` lines, then `git rebase -i upstream/unstable`. Query upstream author per PR:
  ```bash
  gh api "repos/looplj/axonhub/pulls/<NNN>/commits" --jq '.[0].commit.author | "\(.name) <\(.email)>"'
  ```

## 3. GPG Signing

- Every commit must be signed: `git commit -S` (config: `commit.gpgsign = true`, key `55132190BC9FEABD`).
- Any history rewrite (`filter-branch`, `rebase`) **drops signatures** — always re-sign afterwards:
  ```bash
  GIT_SEQUENCE_EDITOR=true GIT_EDITOR=true git rebase --exec 'git commit --amend --no-edit -S' <base>
  ```
- Creating a git tag in this environment hangs on gpg signing (`pinentry-gnome3` + nvim); use `git -c tag.gpgsign=false tag <name> <ref>` instead.

## 4. pnpm Lockfile Pollution

- Local pnpm (v11) rewrites `frontend/pnpm-lock.yaml` (drops the `overrides: uuid` block, bumps many deps). CI uses pnpm 10 with `pnpm install --frozen-lockfile` and will fail on a polluted lockfile.
- After any local pnpm command, check the lockfile and restore it — never commit lockfile changes:
  ```bash
  git checkout -- frontend/pnpm-lock.yaml
  ```

## 5. Verification Before Pushing to unstable

- Backend build incl. CI flag: `go build -ldflags "-s -w" -tags=nomsgpack -o ./tmp/axonhub ./cmd/axonhub`, plus `go build ./...`, `cd llm && go build ./...`, `cd cmd/schema && go build`.
- Tests: `make test-backend-all` (root module + `llm/`).
- Lint: `golangci-lint run --timeout 10m --max-same-issues 50` (`.golangci.yml` is v2 — use golangci-lint v2).
- For data migrations on the PostgreSQL deployment, verify the SQL against the live schema types (`channels.settings` is `jsonb`, `updated_at` is `timestamptz`) before shipping.

## 6. Fork Docker Workflows

- Keep docker image targets at `ghcr.io/${{ github.repository_owner }}/axonhub` (owner-scoped, `GITHUB_TOKEN` auth) — never hardcode the upstream `looplj/axonhub` Docker Hub namespace in `docker-publish.yml` / `docker-unstable.yml`.
- `helm-chart.yml` has an `owner == 'looplj'` guard and should stay unchanged.
- Fork's Actions need **Settings → Actions → General → Workflow permissions = Read and write** for the multi-arch manifest step; the `packages: write` job permission covers single-arch pushes but not `docker buildx imagetools create`.

## 7. Merging PRs into unstable (own work)

- Open PRs with base `unstable`, head `feat/<name>`.
- Merge with **squash** (`gh pr merge <n> --repo ssxwcz/axonhub --squash`), **never `--merge`** — one integration commit on `unstable`, keeping its history clean.
- CI lint runs against the merge base and catches issues in upstream PR code that local `--new` checks miss (e.g. canonicalheader, misspell). If lint fails, fix, re-push, and re-check before merging.
