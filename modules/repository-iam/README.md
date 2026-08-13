# Module: repository-iam

Repository-scoped IAM bindings for one Artifact Registry repository: a reader
set and a writer set, nothing else.

## Boundary

- Never constructs members; callers pass fully formed member strings (for
  example `serviceAccount:<email>`), so the module cannot invent identities.
- Never grants project-level roles, owner or editor roles, or any role outside
  `roles/artifactregistry.reader` and `roles/artifactregistry.writer`.
- Evidence writers receive no routine delete authority through this module;
  Artifact Registry writer on a generic evidence repository is append-focused
  and is paired with the retention archive in `modules/evidence-archive`.

## Usage

```hcl
module "go_approved_iam" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/repository-iam?ref=<exact-version>"

  project_id  = "<organization>-dep-approved"
  location    = "<region>"
  repository  = module.go_approved.id
  writers     = ["serviceAccount:<promoter-service-account>"]
  readers     = ["serviceAccount:<consumer-service-account>"]
}
```
