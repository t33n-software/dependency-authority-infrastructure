# Stack: dep-approved

Reference stack of the approved trust zone: standard repositories per
ecosystem as the only dependency consumer endpoints, the repository-scoped
promotion, revocation and revalidation bindings of the control-zone
identities, the consumer reader bindings, project-level policy compensation,
the audit export into the evidence archive and the read-only diagnostic
bindings of the forensics reader access class on the zone project.

## Boundary

- Approved repositories are the only dependency consumer endpoints; write
  access is bound to the control-zone promoter and revocation controller
  identities and read access to consumers and the control-zone revalidation
  controller, all through the instance-supplied member inputs of the
  canonical IAM target matrix. The stack creates no zone-local workload
  identity by default (the zone pool is created without providers); any
  future zone-local identity is a governed change.
- Revocation is enforced at runtime through Artifact Registry download rules
  (`google_artifact_registry_rule`, `action = "DENY"`, `operation =
  "DOWNLOAD"`) by the revocation controller. This stack intentionally creates
  no static download rules; a revoked package or version is denied before a
  consumer receives it, and that decision is evidence-bound runtime behavior.
- A virtual consumer endpoint may only ever aggregate approved standard
  repositories and is a separate governed decision, never part of this stack.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- The audit export requires the dep-evidence archive bucket; provisioning
  order is dep-evidence first.
- The forensics reader access class: the organization-owned forensics group
  (instance-supplied) holds exactly `roles/logging.viewer` and
  `roles/run.viewer` on this zone project and no other grant — the read-only
  diagnostic bindings are declared through the forensics-readers module.

## Inputs

`project_id`, `location`, `ecosystems` (default `["go"]`), `pool_id`,
`identities` (default empty), the matrix-bound member inputs
`promoter_member`, `revocation_member` and `revalidation_reader_member`,
`consumer_members`, `forensics_group` (the instance-bound forensics reader
group), `evidence_bucket_name`, audit sink settings and
`policy_constraints`.

## Outputs

Repository IDs and consumer endpoint URIs per ecosystem, the pool resource
name, the audit sink writer identity and the enforced policy constraints.
