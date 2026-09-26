# Module: recovery

The break-glass recovery identity of a trust zone: a dedicated service account
whose elevated capability exists only as a declared, dormant
privileged-access entitlement — never a standing grant.

## Boundary

- Never used in normal operation; every use is an audited incident action with
  a recorded decision.
- The identity holds no standing grant on its zone project: the elevated
  capability is carried exclusively by the declared Privileged Access Manager
  entitlement, which binds the curated predefined-role set of the zone's
  declared surface as the privileged access, stays dormant until activated,
  and grants only per activation — approval- and justification-bound,
  time-boxed by the platform-enforced grant duration (`max_request_duration`),
  and audited. An unbounded or standing break-glass grant is a contract
  violation.
- The time-binding lives in the grant duration of the privileged-access
  mechanism, never in an IAM condition: platforms reject conditions on
  primitive roles, and the privileged-access mechanism itself does not admit
  the legacy basic roles (owner, editor, viewer) — both forms are proven
  non-deployable.
- The role set is derived from the zone's declared module inventory and proven
  complete by the infrastructure core's contract guard — never instance-bound;
  it carries only project-grantable roles: a class whose covering capability
  lives above the project level (the organization policies) is proven covered
  by the organization plane and is never forced into the zone set. The
  per-activation duration and the approver set are approved instance
  decisions, and the core never presets them.
- The module declares the standing platform setup of the privileged-access
  surface: the organization-level Privileged Access Manager service agent
  (derived from the instance-bound organization number) holds the project
  service-agent role on the zone project — engine-managed, never a window
  grant. The API activation and the agent's existence precede the apply
  through the governed operator channel (the instance declares the API in the
  zone's capability floor).
- The activation path is proven by exercise, never by existence alone: the
  drill activates the entitlement, proves the elevated capability and lets the
  grant expire — the activation and proof form lives in
  `docs/operations/break-glass-recovery-activation.md`.

## Usage

```hcl
module "break_glass" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/recovery?ref=<exact-version>"

  project_id           = "<organization>-dep-evidence"
  organization_number  = "<organization-number>" # the numeric organization number
  max_request_duration = "<approved-per-activation-duration>" # for example "7200s"
  approvers            = ["<approved-approver-principal>"]    # for example "group:<approver-group>"
}
```
