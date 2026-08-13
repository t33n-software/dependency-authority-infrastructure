# Stack: dep-quarantine

Reference stack of the quarantine trust zone: standard repositories per
ecosystem for blocked or incomplete candidates, the zone workload identity
pool, repository-scoped writer and reader bindings, project-level policy
compensation and the audit export into the evidence archive.

## Boundary

- Quarantine repositories are never consumer endpoints; developer, CI and
  builder consumption is forbidden here.
- Writers are instance-wired members (canonically the admission controller of
  the control zone); the stack never invents identities.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- The audit export requires the dep-evidence archive bucket; provisioning
  order is dep-evidence first.

## Inputs

`project_id`, `location`, `ecosystems` (default `["go"]`), `pool_id`,
optional zone `identities`, `writer_members`, `reader_members`,
`evidence_bucket_name`, audit sink settings and `policy_constraints`.

## Outputs

Repository IDs per ecosystem, the pool resource name, zone service account
emails, the audit sink writer identity and the enforced policy constraints.
