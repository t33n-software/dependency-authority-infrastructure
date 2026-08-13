# Stack: dep-evidence

Reference stack of the evidence trust zone: generic evidence repositories per
ecosystem, the long-term immutable retention archive, the evidence writer and
auditor workload identities, repository-scoped bindings, project-level policy
compensation and the zone's own audit export into the archive.

## Boundary

- Evidence is append-only: the writer appends, the auditor reads, and no
  identity receives routine delete authority.
- The retention archive pairs the operational evidence repositories; a
  deletable repository version alone is not a long-term evidence control.
- `lock_retention_policy` stays `false` until the retention and legal-hold
  duration is approved and recorded by the organization instance; locking is
  irreversible.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- Provisioning order: this stack lands first, because every other zone exports
  its audit logs into the archive bucket created here.

## Inputs

`project_id`, `location`, `ecosystems` (default `["go"]`), `pool_id`,
`writer` and `auditor` (OIDC bindings), `archive_bucket_name`,
`retention_period_seconds`, `lock_retention_policy`, optional
`archive_kms_key_name` and `additional_auditor_members`, audit sink settings
and `policy_constraints`.

## Outputs

Evidence repository IDs per ecosystem, the archive bucket name (consumed by
the other zone stacks), writer and auditor service account emails, the pool
resource name, the audit sink writer identity and the enforced policy
constraints.
