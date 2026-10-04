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
- The audit export writer surface is optional and append-focused: the
  consuming stack wires the Cloud Logging service agents of the
  organization-plane audit export anchors, and the module binds them with
  the fixed canonical destination role `roles/storage.objectCreator` —
  never read, never delete, never a configurable wider role and never a
  window grant.

## Usage

```hcl
module "evidence_archive" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/evidence-archive?ref=<exact-version>"

  project_id              = "<organization>-dep-evidence"
  name                    = "<archive-bucket-name>"
  location                = "<region>"
  retention_period_seconds = 94608000 # three years, as an approved example
  lock_retention_policy   = false     # flip only after the retention decision is approved

  # Optional: the standing audit export writers of the organization plane
  # (the Cloud Logging service agents of the audit export anchors).
  logging_export_writer_members = toset([
    "serviceAccount:service-org-123456789@gcp-sa-logging.iam.gserviceaccount.com",
  ])

  labels = {
    boundary = "dependency-authority"
    zone     = "evidence"
  }
}
```
