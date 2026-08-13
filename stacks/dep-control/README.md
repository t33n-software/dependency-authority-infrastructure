# Stack: dep-control

Reference stack of the control trust zone: the control-plane controller
workload identities (admission, revalidation and revocation lanes), the zone
workload identity pool, project-level policy compensation and the audit export
into the evidence archive.

## Boundary

- The control zone hosts decision logic, not packages: it creates no
  dependency repositories and never serves consumers.
- Controller identities receive their repository access in the intake,
  quarantine, approved and evidence zones through those zones' member inputs;
  cross-zone authority is never granted project-wide here.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- The audit export requires the dep-evidence archive bucket; provisioning
  order is dep-evidence first.

## Inputs

`project_id`, `pool_id`, `controllers` (OIDC bindings of the controller
identities), `evidence_bucket_name`, audit sink settings and
`policy_constraints`.

## Outputs

Controller service account emails keyed by lane, the pool resource name, the
audit sink writer identity and the enforced policy constraints.
