# Module: forensics-readers

The forensics reader access class of one trust zone: the read-only diagnostic
bindings of the organization-owned forensics group and — declared exactly once,
by the control-zone stack — the second, separate perimeter ingress rule that
lets the group read logs across the boundary.

## Boundary

- Never carries organization, tenant, identity, secret or registry bindings;
  the forensics group member string, the perimeter resource name and the zone
  projects are instance-supplied values.
- The group holds exactly `roles/logging.viewer` and `roles/run.viewer` on each
  zone project — project-scoped, never organization-scoped — and no other
  grant on any plane; the read-only form is enforced by the granted roles,
  never by convention.
- The group is created and membership-managed on the organization identity
  plane outside this core; this module only binds its access class.
- The perimeter ingress rule carries the forensics group as the only identity,
  scoped to the read-only logging method `logging.logEntries.list`, with the
  zone projects as resources, through the same identity-bound channel as the
  administration rule; the administration ingress rule
  never carries the forensics identity, and every additional read method is a
  governed change to the rule.
- The execution status read-back travels the deliberately non-restricted
  compute control plane and needs no perimeter rule.

## Usage

```hcl
module "forensics_readers" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/forensics-readers?ref=<exact-version>"

  project_id      = "<organization>-dep-intake"
  forensics_group = "group:dep-forensics-readers@<organization-domain>"

  # Only the control-zone stack binds the perimeter rule:
  perimeter_ingress = {
    perimeter_name = "accessPolicies/<access-policy-id>/servicePerimeters/<perimeter>"
    zone_projects  = ["projects/<zone-project-number>"]
  }
}
```

## Notes

- The module creates no project, enables no APIs and provisions no perimeter;
  the organization instance binds the existing perimeter by name.
- Every forensics read is exported through the zone audit sinks into the
  evidence archive with the forensics group as the actor.
