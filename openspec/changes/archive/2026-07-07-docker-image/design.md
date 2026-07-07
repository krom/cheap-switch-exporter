## Context

The repo has a working multi-stage `Dockerfile` (Go build stage + alpine runtime stage) but no CI.
`origin` points to `github.com/krom/cheap-switch-exporter`. GitHub Container Registry (GHCR) is the
natural registry choice since it needs no separate account/secrets — GitHub Actions can push to it
using the built-in `GITHUB_TOKEN` as long as the workflow requests `packages: write` permission.

## Goals / Non-Goals

**Goals:**
- Build the Docker image automatically **only when a version tag (`v*`) is pushed** — no build on
  ordinary `main` pushes.
- Publish to `ghcr.io/krom/cheap-switch-exporter`, using the pushed tag itself (e.g. `v1.0.0`) as
  the Docker image tag, plus a `latest` tag pointing at the same image.
- Keep the workflow using only credentials GitHub already provides (no new PATs/secrets to manage).
- Link the README to the published image so users can `docker pull` without building locally.

**Non-Goals:**
- Building/publishing on every `main` push — deliberately excluded; releases are tag-driven only.
- Multi-architecture builds (arm64/etc.) — single `linux/amd64` build only, matching current
  Dockerfile's explicit `GOARCH=amd64`. Can be added later as a separate change.
- Publishing to Docker Hub or any registry other than GHCR.
- Image signing/SBOM/provenance attestation.
- Automated GHCR package visibility toggling — first-run visibility (public) is a manual,
  one-time step in GitHub package settings, called out in tasks.

## Decisions

- **Registry: GHCR over Docker Hub.** No separate account/secret needed; auth is
  `docker/login-action` with `${{ secrets.GITHUB_TOKEN }}`. Docker Hub would need a separate
  username/token secret pair for no real benefit here.
- **Trigger: tags matching `v*` only.** No trigger on `main` push or PRs — a build only happens
  when a version tag like `v1.0.0` is pushed. The workflow's `on:` block uses `push: tags: ['v*']`
  with no `branches:` entry, so ordinary commits to `main` never build/publish an image.
- **Docker tag = git tag.** `docker/metadata-action` derives the image tag directly from
  `github.ref_name` (the pushed tag, e.g. `v1.0.0`), plus a `latest` tag pointing at the same
  build. No floating major/minor tags and no branch/SHA tags — keeps tagging simple and 1:1 with
  releases.
- **Actions used: `docker/setup-buildx-action`, `docker/login-action`, `docker/metadata-action`,
  `docker/build-push-action`.** These are the standard, actively maintained Docker-owned actions for
  this exact use case, avoiding hand-rolled `docker build`/`docker push` scripting.
- **`org.opencontainers.image.source` label fix.** GHCR uses this OCI label (already present in the
  Dockerfile, but pointing at the upstream fork `pvelati/cheap-switch-exporter`) to link a published
  package back to its source repo on the package page. Must be updated to `krom/cheap-switch-exporter`
  so the GHCR UI links correctly and provenance isn't misattributed.
- **No change to the Dockerfile's build logic itself** — only the label. The existing multi-stage
  build already produces a small, non-root final image; nothing about the CI publishing step
  requires changing how the image is built.

## Risks / Trade-offs

- [New GHCR packages default to private] → Document a one-time manual step in tasks.md: after the
  first successful workflow run, set the package visibility to public in GitHub package settings, so
  `docker pull` works without `docker login`.
- [`latest` always follows whichever tag was pushed most recently] → Acceptable: since builds are
  tag-only, `latest` simply tracks the newest release tag rather than an unreleased `main` HEAD.
  Users wanting a fixed version should pull the specific version tag instead. Documented in README.
- [No image available for unreleased `main` commits] → Intentional trade-off per this design: only
  tagged releases are published. Users needing an image from an unreleased commit must build
  locally with the existing `docker build` instructions.

## Migration Plan

- No migration needed — purely additive (new workflow file, one label edit, README addition).
- Rollback: delete/disable the workflow file; no effect on existing runtime behavior.
