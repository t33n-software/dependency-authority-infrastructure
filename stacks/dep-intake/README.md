# Stack: dep-intake

Reference stack of the intake trust zone: remote intake repositories per
ecosystem, the intake fetcher workload identity, repository-scoped writer
binding, project-level policy compensation, the audit export into the
evidence archive and the zone's workload job of the in-perimeter execution
substrate (`dep-intake-fetch`, executed as the intake fetcher identity).

## Boundary

- Intake repositories are never consumer endpoints; only the intake fetcher
  writes, and no consumer identity receives access here. The only additional
  readers are the control-plane identities of the canonical IAM target matrix
  (canonically the admission and promotion lanes), wired through the
  instance-supplied member input.
- Go remote intake maps to `common_repository.uri = "https://proxy.golang.org"`;
  npm and python map to their public upstream enums. Private Go modules never
  enter through this remote repository; they require the separately governed
  intake adapter.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- The audit export requires the dep-evidence archive bucket; provisioning
  order is dep-evidence first.
- The workload job consumes its image by full immutable digest from the
  release-class workload image registry only; the digest is an instance
  binding (`planned` with a documented placeholder until the promotion
  read-back proof flips it to `bound`), never a stack default.

## Inputs

`project_id`, `location`, `ecosystems` (default `["go"]`), `pool_id`,
`fetcher` (OIDC bindings of the intake fetcher), `additional_reader_members`
(the matrix-bound control-plane readers), `workload_job_images`
(the instance-bound image digests keyed by canonical job name),
`evidence_bucket_name`, audit sink settings and `policy_constraints`.

## Outputs

Repository IDs and URIs per ecosystem, the fetcher service account email, the
pool resource name, the audit sink writer identity, the enforced policy
constraints and the workload job resource IDs keyed by canonical job name.
