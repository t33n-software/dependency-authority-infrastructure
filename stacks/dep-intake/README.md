# Stack: dep-intake

Reference stack of the intake trust zone: remote intake repositories per
ecosystem, the intake fetcher workload identity, repository-scoped writer
binding, project-level policy compensation and the audit export into the
evidence archive.

## Boundary

- Intake repositories are never consumer endpoints; only the intake fetcher
  writes, and no consumer identity receives access here.
- Go remote intake maps to `common_repository.uri = "https://proxy.golang.org"`;
  npm and python map to their public upstream enums. Private Go modules never
  enter through this remote repository; they require the separately governed
  intake adapter.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- The audit export requires the dep-evidence archive bucket; provisioning
  order is dep-evidence first.

## Inputs

`project_id`, `location`, `ecosystems` (default `["go"]`), `pool_id`,
`fetcher` (OIDC bindings of the intake fetcher), `evidence_bucket_name`,
audit sink settings and `policy_constraints`.

## Outputs

Repository IDs and URIs per ecosystem, the fetcher service account email, the
pool resource name, the audit sink writer identity and the enforced policy
constraints.
