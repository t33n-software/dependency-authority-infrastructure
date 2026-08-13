# policy-bindings

Project-level organization-policy constraints for one trust zone. They
compensate the absent organization node: where an organization would enforce
these constraints uniformly, every zone project enforces them itself until the
organization migration lands.

## Boundary

- The enforced posture is the default; disabling a constraint is an explicit,
  reviewable instance decision.
- This module only covers the four canonical constraints
  (`iam.disableServiceAccountKeyCreation`, `iam.disableServiceAccountKeyUpload`,
  `storage.uniformBucketLevelAccess`, `storage.publicAccessPrevention`);
  additional constraints enter through a governed change.
- Never carries organization, tenant, identity, network, secret or registry
  bindings beyond the instance-supplied project ID.

## Usage

```hcl
module "policy_bindings" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//policy-bindings?ref=<exact-version>"

  project_id = "<organization>-dep-intake"
}
```
