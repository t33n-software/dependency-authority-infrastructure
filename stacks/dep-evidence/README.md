# Stack: dep-evidence

Reference stack of the evidence trust zone: generic evidence repositories per
ecosystem, the long-term immutable retention archive, the evidence writer and
auditor workload identities, repository-scoped bindings, project-level policy
compensation, the zone's own audit export into the archive and the zone's
workload jobs of the in-perimeter execution substrate (`dep-evidence-write`
and `dep-evidence-audit`, executed as the writer and auditor identities).

## Boundary

- Evidence is append-only: the writer appends, the auditor reads, and no
  identity receives routine delete authority. The only additional writers and
  readers are the control-plane identities of the canonical IAM target matrix
  (canonically the admission, revalidation and revocation lanes writing, the
  promotion lane reading), wired through the instance-supplied member inputs.
- The retention archive pairs the operational evidence repositories; a
  deletable repository version alone is not a long-term evidence control.
- `lock_retention_policy` stays `false` until the retention and legal-hold
  duration is approved and recorded by the organization instance; locking is
  irreversible.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- Provisioning order: this stack lands first, because every other zone exports
  its audit logs into the archive bucket created here.
- The workload jobs consume their images by full immutable digest from the
  release-class workload image registry only; the digests are instance
  bindings (`planned` with documented placeholders until the promotion
  read-back proofs flip them to `bound`), never stack defaults.

## Inputs

`project_id`, `location`, `ecosystems` (default `["go"]`), `pool_id`,
`writer` and `auditor` (OIDC bindings), `workload_job_images` (the
instance-bound image digests keyed by canonical job name),
`archive_bucket_name`, `retention_period_seconds`, `lock_retention_policy`,
optional `archive_kms_key_name`, the matrix-bound `additional_writer_members`
and `additional_auditor_members`, audit sink settings and
`policy_constraints`.

## Outputs

Evidence repository IDs per ecosystem, the archive bucket name (consumed by
the other zone stacks), writer and auditor service account emails, the pool
resource name, the audit sink writer identity, the enforced policy
constraints and the workload job resource IDs keyed by canonical job name.
