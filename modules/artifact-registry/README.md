# Module: artifact-registry

One Google Artifact Registry repository for the dependency authority trust
zones: GO, NPM or PYTHON package repositories (standard or remote mode),
GENERIC evidence repositories and DOCKER workload image repositories
(standard mode only).

## Boundary

- Never carries organization, tenant, project, identity, network, secret or
  registry bindings; every concrete value is an instance-supplied variable.
- Remote intake for Go maps to `common_repository.uri = "https://proxy.golang.org"`:
  the Artifact Registry remote-source union has no Go-specific field, and the
  `gcloud --remote-go-repo` flag maps to the same common repository URI.
- DOCKER workload image repositories always use standard mode and never bind
  a remote upstream: the workload classes (`staging-*`, `release-*`) are
  filled through the governed producer channel and promotion, never proxied.
- Revocation download rules (`google_artifact_registry_rule`) are a runtime
  operation of the revocation controller and are intentionally never created
  by this module.
- Virtual consumer endpoints are out of scope; an optional virtual endpoint
  may only ever aggregate Approved standard repositories and is a separate
  governed decision.

## Usage

```hcl
module "go_intake" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/artifact-registry?ref=<exact-version>"

  project_id      = "<organization>-dep-intake"
  location        = "<region>"
  repository_id   = "go-dependencies-intake"
  description     = "Go intake: controlled remote upstream acquisition"
  format          = "GO"
  mode            = "REMOTE_REPOSITORY"
  remote_upstream = { common_uri = "https://proxy.golang.org" }

  labels = {
    boundary  = "dependency-authority"
    zone      = "intake"
    ecosystem = "go"
  }
}
```

```hcl
module "staging_controller_images" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/artifact-registry?ref=<exact-version>"

  project_id    = "<organization>-dep-control"
  location      = "<region>"
  repository_id = "staging-controller-images"
  description   = "Workload image staging: governed producer delivery target, never a workload source"
  format        = "DOCKER"
  mode          = "STANDARD_REPOSITORY"

  labels = {
    boundary = "dependency-authority"
    zone     = "control"
  }
}
```

## Notes

- `kms_key_name` is immutable after repository creation; the instance controls
  provisioning order.
- `cleanup_policy_dry_run` defaults to `true` so retention rules never delete
  unless an instance explicitly arms them.
