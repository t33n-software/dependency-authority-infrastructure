# policy-bindings

Project-level organization-policy constraints for one trust zone. They
compensate the absent organization node: where an organization would enforce
these constraints uniformly, every zone project enforces them itself until the
organization migration lands.

## Boundary

- The enforced posture is the default; disabling a constraint is an explicit,
  reviewable instance decision.
- This module covers the four canonical boolean constraints
  (`iam.disableServiceAccountKeyCreation`, `iam.disableServiceAccountKeyUpload`,
  `storage.uniformBucketLevelAccess`, `storage.publicAccessPrevention`) plus
  the opt-in Cloud Run enforcement surface of the workload network origin
  form (`run.allowedVPCEgress` restricted to `all-traffic` and
  `run.allowedIngress` restricted to `internal`, both disabled by default and
  enabled only by the job-owning zones); additional constraints enter through
  a governed change.
- Every organization policy is a number-addressed resource (its name and
  parent carry the `projects/<number>` form), so the module binds the
  instance-supplied numeric project number (`project_number`) and never the
  project ID — binding the project ID would force a destroy-and-recreate of
  the live policies at the convergence window.
- Never carries organization, tenant, identity, network, secret or registry
  bindings beyond the instance-supplied project number.

## Usage

```hcl
module "policy_bindings" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//policy-bindings?ref=<exact-version>"

  project_number = "<numeric-project-number-of-the-zone>"

  # The job-owning zones enable the Cloud Run enforcement surface.
  cloud_run_vpc_egress_all_traffic_only = true
  cloud_run_ingress_internal_only       = true
}
```
