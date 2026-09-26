# Stack: dep-quarantine

Reference stack of the quarantine trust zone: standard repositories per
ecosystem for blocked or incomplete candidates, the zone workload identity
pool, repository-scoped writer and reader bindings, project-level policy
compensation, the audit export into the evidence archive and the read-only
diagnostic bindings of the forensics reader access class on the zone project.

## Boundary

- Quarantine repositories are never consumer endpoints; developer, CI and
  builder consumption is forbidden here.
- Writers are instance-wired members (canonically the admission controller of
  the control zone); the stack never invents identities.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- The audit export requires the dep-evidence archive bucket; provisioning
  order is dep-evidence first.
- The forensics reader access class: the organization-owned forensics group
  (instance-supplied) holds exactly `roles/logging.viewer` and
  `roles/run.viewer` on this zone project and no other grant — the read-only
  diagnostic bindings are declared through the forensics-readers module.
- The recovery identity of the quarantine zone: the stack declares the
  dedicated identity through the recovery module — its elevated capability
  exists only as the declared, dormant privileged-access entitlement (never a
  standing grant), it is never used in normal operation and never federated
  from CI, and it holds no data-plane grant. Every activation is approval-
  and justification-bound and time-boxed by the platform-enforced grant
  duration; the duration and the approver set are approved instance
  decisions, supplied through the `break_glass_recovery` input.
- The stack consumes the zone state home — the dedicated state bucket of the
  zone holding that zone's root states and nothing else — through the final
  `gcs` backend binding with the state-key grammar prefix identifying exactly
  this root; the bucket is provisioned by the converged foundation, never by
  this stack and never by hand, and every state and plan artifact of this
  root is client-side encrypted through the engine layer of the dual
  fortress state-encryption standard, fail-closed enforced.

## Inputs

`project_id`, `project_number` (the instance-bound numeric project number bound by the zone workload identity pool — the pool's provider state carries the number, never the ID), `location`, `ecosystems` (default `["go"]`), `pool_id`,
optional zone `identities`, `writer_members`, `reader_members`,
`break_glass_recovery` (the approved recovery binding of the quarantine
zone: the per-activation grant duration of the recovery entitlement and the
approver principal set of its approval workflow),
`forensics_group` (the instance-bound forensics reader group),
`evidence_bucket_name`, `state_bucket_name` (the instance-bound zone state
home bucket, provisioned by the converged foundation — never by this stack),
`state_encryption_key` (the instance-bound engine key reference of this
root's client-side state and plan encryption), audit sink settings and
`policy_constraints`.

## Outputs

Repository IDs per ecosystem, the pool resource name, zone service account
emails, the audit sink writer identity, the enforced policy constraints, the
recovery identity email and the recovery entitlement resource name.
