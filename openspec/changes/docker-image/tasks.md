## 1. Workflow

- [x] 1.1 Create `.github/workflows/docker-publish.yml` triggered ONLY on push of tags matching
      `v*` (`on: push: tags: ['v*']`, no `branches:` trigger), with
      `permissions: packages: write, contents: read`.
- [x] 1.2 Add steps: checkout, `docker/setup-buildx-action`, `docker/login-action` against
      `ghcr.io` using `${{ github.actor }}` / `${{ secrets.GITHUB_TOKEN }}`.
- [x] 1.3 Add `docker/metadata-action` to derive tags from the pushed git tag itself
      (`type=ref,event=tag`, e.g. `v1.0.0`) plus a `latest` tag on the same image — no
      floating major/minor or SHA tags.
- [x] 1.4 Add `docker/build-push-action` using the repo's existing `Dockerfile`, `push: true`,
      and the tags/labels from the metadata step.

## 2. Dockerfile

- [x] 2.1 Update `org.opencontainers.image.source` label in `Dockerfile` from
      `https://github.com/pvelati/cheap-switch-exporter` to
      `https://github.com/krom/cheap-switch-exporter`.

## 3. Documentation

- [x] 3.1 Add a `docker pull ghcr.io/krom/cheap-switch-exporter:latest` example (and/or a link to
      the GHCR package page) to the README's Docker Deployment section, alongside the existing
      `docker build` instructions.

## 4. Verification

- [x] 4.1 Validate the workflow YAML (e.g. `actionlint` or GitHub's workflow syntax check).
- [ ] 4.2 Push a test version tag (e.g. `v0.0.1-test`) and confirm the workflow runs and publishes
      an image to GHCR tagged with that version; confirm a plain push to `main` does NOT trigger it.
- [ ] 4.3 One-time manual step: set the new GHCR package visibility to public in GitHub package
      settings so `docker pull` works without authentication.
