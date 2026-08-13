# Stack: dep-approved

Reference stack of the approved trust zone: standard repositories per
ecosystem as the only dependency consumer endpoints, the approved promoter
workload identity, repository-scoped writer and consumer reader bindings,
project-level policy compensation and the audit export into the evidence
archive.

## Boundary

- Approved repositories are the only dependency consumer endpoints; only the
  promoter writes, consumers receive read-only access.
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

## Inputs

`project_id`, `location`, `ecosystems` (default `["go"]`), `pool_id`,
`promoter` (OIDC bindings of the approved promoter), `consumer_members`,
`evidence_bucket_name`, audit sink settings and `policy_constraints`.

## Outputs

Repository IDs and consumer endpoint URIs per ecosystem, the promoter service
account email, the pool resource name, the audit sink writer identity and the
enforced policy constraints.
