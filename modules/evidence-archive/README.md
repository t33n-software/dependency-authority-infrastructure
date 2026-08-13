# Module: evidence-archive

The long-term immutable evidence retention archive: one Cloud Storage bucket
with uniform bucket-level access, enforced public access prevention,
versioning and a retention policy.

## Boundary

- Never carries organization, tenant, identity, network, secret or registry
  bindings; every concrete value is an instance-supplied variable.
- `lock_retention_policy` defaults to `false` and is a one-way door: it may
  only be set to `true` after the retention and legal-hold duration is
  approved and recorded by the organization instance.
- The archive complements the operational Generic Artifact Registry evidence
  repositories; a deletable repository version alone is not a long-term
  evidence control.

## Usage

```hcl
module "evidence_archive" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/evidence-archive?ref=<exact-version>"

  project_id              = "<organization>-dep-evidence"
  name                    = "<archive-bucket-name>"
  location                = "<region>"
  retention_period_seconds = 94608000 # three years, as an approved example
  lock_retention_policy   = false     # flip only after the retention decision is approved

  labels = {
    boundary = "dependency-authority"
    zone     = "evidence"
  }
}
```
