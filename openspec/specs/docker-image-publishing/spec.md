### Requirement: Tag-triggered image build and publish
The system SHALL build a Docker image from the repository's `Dockerfile` and publish it to GitHub
Container Registry (`ghcr.io/krom/cheap-switch-exporter`) automatically via GitHub Actions,
triggered only by pushing a version tag, without requiring any manually-provisioned registry
credentials.

#### Scenario: Version tag publishes semantic version image tags
- **WHEN** a tag matching `v*` (e.g. `v1.0.0`) is pushed
- **THEN** the workflow builds the Docker image and pushes it to
  `ghcr.io/krom/cheap-switch-exporter` tagged with the version stripped of its `v` prefix at three
  granularities (`1.0.0`, `1.0`, `1`) and also as `latest`

#### Scenario: Plain push to main does not publish
- **WHEN** a commit is pushed to the `main` branch without an accompanying version tag
- **THEN** the workflow does not run and no image is built or published

#### Scenario: Build failure blocks publish
- **WHEN** the Docker image build step fails (e.g. compile error) during a tagged build
- **THEN** the workflow fails and no image is pushed to the registry

### Requirement: Published image is discoverable from documentation
The system SHALL document the published GHCR image location in `README.md` so users can pull the
prebuilt image instead of building it locally.

#### Scenario: User reads Docker deployment instructions
- **WHEN** a user reads the README's Docker Deployment section
- **THEN** they find a `docker pull ghcr.io/krom/cheap-switch-exporter:latest` command and a
  pinned-version example using the bare major version tag (e.g. `:1`, no `v` prefix), as an
  alternative to `docker build`
