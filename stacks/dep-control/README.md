# Stack: dep-control

Reference stack of the control trust zone: the control-plane controller
workload identities (admission, revalidation and revocation lanes), the zone
workload identity pool, project-level policy compensation, the audit export
into the evidence archive and the workload image registries of the
in-perimeter execution substrate.

## Boundary

- The control zone hosts decision logic, not packages: it creates no
  dependency repositories and never serves consumers. Its two workload image
  registries are DOCKER standard repositories of the classes staging and
  release: `staging-controller-images` is the only delivery target of the
  governed producer channel, `release-controller-images` is the only workload
  consumption source and is filled exclusively through promotion of a proven
  staging digest; neither class is a dependency repository and neither ever
  binds a remote upstream.
- Controller identities receive their repository access in the intake,
  quarantine, approved and evidence zones through those zones' member inputs;
  cross-zone authority is never granted project-wide here.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- The audit export requires the dep-evidence archive bucket; provisioning
  order is dep-evidence first.

## Inputs

`project_id`, `location`, `pool_id`, `controllers` (OIDC bindings of the
controller identities), `evidence_bucket_name`, audit sink settings and
`policy_constraints`.

## Outputs

Controller service account emails keyed by lane, the pool resource name, the
audit sink writer identity, the enforced policy constraints and the workload
image repository IDs and endpoint URIs keyed by class (`staging`, `release`).
