## Why

The project has a `Dockerfile` but no automated way to build and publish an image, so users must
clone the repo and build it themselves. Publishing a prebuilt image to GitHub Container Registry
(GHCR) via GitHub Actions lets users `docker pull`/`docker run` directly, and a README link makes
the published image discoverable.

## What Changes

- Add a GitHub Actions workflow (`.github/workflows/docker-publish.yml`) that builds the Docker
  image **only when a version tag (`v*`, e.g. `v1.0.0`) is pushed** and publishes it to
  `ghcr.io/krom/cheap-switch-exporter`. Plain pushes to `main` do NOT trigger a build/publish.
  - Tag pushed: the exact git tag as the Docker tag (e.g. `v1.0.0` → image tag `v1.0.0`), plus
    `latest` pointing at the same image.
  - Uses `docker/build-push-action` with GHCR login via `GITHUB_TOKEN` (no new secrets needed).
- Update `Dockerfile` `org.opencontainers.image.source` label from the upstream fork
  (`pvelati/cheap-switch-exporter`) to this repo (`krom/cheap-switch-exporter`), since GHCR uses
  this label to link the package to its source repo.
- Update `README.md`'s Docker Deployment section with a link/pull command for the published GHCR
  image, as an alternative to building locally.

## Capabilities

### New Capabilities
- `docker-image-publishing`: automated CI build and publish of the Docker image to GHCR, triggered
  only by version tags, using the tag itself as the image tag.

### Modified Capabilities
(none — no existing spec covers CI/CD or packaging)

## Impact

- Affected files: new `.github/workflows/docker-publish.yml`, `Dockerfile` (label fix), `README.md`
  (Docker section).
- No changes to exporter runtime behavior, config schema, or metrics.
- Requires GHCR package visibility to be set to public (one-time manual step after first publish,
  or via workflow permissions) for users to pull without authentication.
