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
- **Own features and small fixes** are committed directly on `unstable` (sync it first, then `git commit -S && git push origin unstable`). Do **not** open PRs for own changes — this fork has no collaborators and no protected branches. Only use a PR when an external review is explicitly wanted.
- **Own changes that stay on `unstable`** (re-apply if a sync conflict drops them):
  - `.goreleaser.yml` — homebrew `brews` section removed (upstream tap cannot be written by this fork's token)
  - `docker-publish.yml`, `docker-unstable.yml` — publish to `ghcr.io/<owner>/axonhub` (GHCR, `GITHUB_TOKEN`)
  - `.agent/rules/workflows/merge-upstream-prs.md` — this doc
  - `internal/build/VERSION` — keep aligned with the deployed release (currently `v1.0.0-beta9`)
  - frontend price trim incl. schedule overrides in `channels-model-price-dialog.tsx`
- **Release**: from `unstable`, tag `v<version>` — triggers the Release (binaries) and Docker image workflows (GHCR). Never commit auto-linkable refs.

## 1. Commit message format (Google style) — MANDATORY for every commit

Every commit in this fork — normal commits, cherry-picks, merges, and squash PRs — must follow the [Google commit-message guidelines](https://google.github.io/eng-practices/review/developer/cl-descriptions.html):

1. **Separate subject from body with a blank line.**
2. **Limit the subject line to 50 characters.** Subject format: `type(scope): imperative summary` (type: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `ci`).
3. **Capitalize the subject line; do not end it with a period.**
4. **Use the imperative mood** in the subject ("Add", "Fix", "Merge" — not "Added", "Fixes", "Merged").
5. **Wrap the body at 72 characters.**
6. **Use the body to explain WHAT and WHY, not HOW** — the diff shows how; the message should say why the change is needed and what it changes. Never "misc fixes".

Example (own change):
```
feat(channels): add price import and export as JSON

Channels currently manage prices one row at a time, which is slow to
set up and error-prone. Add bulk import/export so operators can migrate
prices across channels or from spreadsheets.
```

Example (sync merge):
```
Merge upstream/unstable into fork unstable

Bring in 9 upstream commits: prompt cache key, price trim/export,
viewport floor, model developers data, SQLite rebuild fix, 100-row
pagination, clipboard copy, and ordering weight validation. Duplicate
PR conflicts were resolved toward upstream; local zai GLM-5.2/5.3
thinking tests were kept because upstream has no equivalent yet.
```

Example (squash PR):
```
feat: integrate upstream PRs and review fixes into unstable

Merge 20 reviewed upstream PRs plus review fixes as one integration
commit. Full per-PR history (original authors and GPG signatures) is
preserved on the source branch. [group the included PRs and why]
```

## 2. PR / Issue References (avoid auto-linking)

- **Never** use `#NNN`, `owner/repo#NNN` (or any format GitHub auto-links) in commit title or message — it creates a trail of cross-repo references on the upstream PR/issue pages.
- Use plain text to keep traceability without linking: `(PR NNNN)`, `per PR NNNN`, `upstream PR NNNN`.

## 3. Author / Committer Attribution

- When cherry-picking or integrating an upstream commit: keep **author = the upstream author**, **committer = self**. Manual conflict resolution and integration work is attributed through the committer field, never by claiming the author.
- For your own fixes, features, reverts, and chores: author = self.
- If a commit was already created with self as author, rewrite per-commit author via a `GIT_SEQUENCE_EDITOR` script that injects `exec git commit --amend --no-edit -S --author="Name <email>"` after the matching `pick` lines, then `git rebase -i upstream/unstable`. Query upstream author per PR:
  ```bash
  gh api "repos/looplj/axonhub/pulls/<NNN>/commits" --jq '.[0].commit.author | "\(.name) <\(.email)>"'
  ```

## 4. GPG Signing

- Every commit must be signed: `git commit -S` (config: `commit.gpgsign = true`, key `55132190BC9FEABD`).
- Any history rewrite (`filter-branch`, `rebase`) **drops signatures** — always re-sign afterwards:
  ```bash
  GIT_SEQUENCE_EDITOR=true GIT_EDITOR=true git rebase --exec 'git commit --amend --no-edit -S' <base>
  ```
- Creating a git tag in this environment hangs on gpg signing (`pinentry-gnome3` + nvim); use `git -c tag.gpgsign=false tag <name> <ref>` instead.

## 5. pnpm Lockfile Pollution

- Local pnpm (v11) rewrites `frontend/pnpm-lock.yaml` (drops the `overrides: uuid` block, bumps many deps). CI uses pnpm 10 with `pnpm install --frozen-lockfile` and will fail on a polluted lockfile.
- After any local pnpm command, check the lockfile and restore it — never commit lockfile changes:
  ```bash
  git checkout -- frontend/pnpm-lock.yaml
  ```

## 6. Verification Before Pushing to unstable

- Backend build incl. CI flag: `go build -ldflags "-s -w" -tags=nomsgpack -o ./tmp/axonhub ./cmd/axonhub`, plus `go build ./...`, `cd llm && go build ./...`, `cd cmd/schema && go build`.
- Tests: `make test-backend-all` (root module + `llm/`).
- Lint: `golangci-lint run --timeout 10m --max-same-issues 50` (`.golangci.yml` is v2 — use golangci-lint v2).
- For data migrations on the PostgreSQL deployment, verify the SQL against the live schema types (`channels.settings` is `jsonb`, `updated_at` is `timestamptz`) before shipping.

## 7. Fork Docker Workflows

- Keep docker image targets at `ghcr.io/${{ github.repository_owner }}/axonhub` (owner-scoped, `GITHUB_TOKEN` auth) — never hardcode the upstream `looplj/axonhub` Docker Hub namespace in `docker-publish.yml` / `docker-unstable.yml`.
- `helm-chart.yml` has an `owner == 'looplj'` guard and should stay unchanged.
- Fork's Actions need **Settings → Actions → General → Workflow permissions = Read and write** for the multi-arch manifest step; the `packages: write` job permission covers single-arch pushes but not `docker buildx imagetools create`.

## 8. Landing own work (no PRs)

- Commit own fixes/features directly to `unstable`: sync upstream first (`git merge upstream/unstable`), then commit and push.
- Only open a PR when an external review is explicitly requested; otherwise never — this fork has no branch protection.
- Merge commits from upstream sync keep `Merge upstream/unstable into fork unstable` style subjects.
