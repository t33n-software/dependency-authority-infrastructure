# Stack: dep-intake

Reference stack of the intake trust zone: remote intake repositories per
ecosystem, the intake fetcher workload identity, repository-scoped writer
binding, project-level policy compensation, the audit export into the
evidence archive, the zone workload network origin (one VPC with one Private
Google Access subnetwork in the job region, the restricted-range DNS response
policy covering `*.googleapis.com` and the Artifact Registry data plane
`*.pkg.dev`, and the egress firewall pair) and the zone's workload job of the
in-perimeter execution substrate (`dep-intake-fetch`, executed as the intake
fetcher identity and invoked only through its dedicated invoke-only trigger
identity `dep-intake-fetch-trigger`). The stack additionally declares the
read-only diagnostic bindings of the forensics reader access class on the zone
project and opts in to the registry-platform upstream allowance of the intake
zone (the zone-level VPC Service Controls configuration that permits the
remote repositories' configured upstreams; never a perimeter egress rule).

## Boundary

- Intake repositories are never consumer endpoints; only the intake fetcher
  writes, and no consumer identity receives access here. The only additional
  readers are the control-plane identities of the canonical IAM target matrix
  (canonically the admission, revalidation and promotion lanes), wired through
  the instance-supplied member input.
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
- The workload job of the zone follows the engine-active surface composition
  of the operating model: the stack consumes the instance-bound activation
  set (`enabled_workload_jobs`) — exactly the bound jobs plus the jobs being
  provisioned in the current window — and a declared-but-planned job never
  enters the plan until its provisioning window activates it. The activation
  set always carries every bound job (a bound job dropped from the active set
  would plan its own destruction), and the declaration binds the consistency
  fail-closed.
- The workload job attaches to the zone VPC declared by this stack and routes
  all outgoing traffic through it (Direct VPC egress, all-traffic): the
  workload network origin is part of the execution contract, and the stack
  enforces the form through the Cloud Run organization policies
  (`run.allowedVPCEgress` allows only all-traffic, `run.allowedIngress`
  allows only internal). A job without the zone network attachment presents
  no in-perimeter network origin and fails closed at the perimeter.
- The lane federates to the dedicated invoke-only trigger identity of the
  job, never to the execution identity; the trigger identity holds invoke on
  exactly `dep-intake-fetch` and no data-plane grant.
- The forensics reader access class: the organization-owned forensics group
  (instance-supplied) holds exactly `roles/logging.viewer` and
  `roles/run.viewer` on this zone project and no other grant — the read-only
  diagnostic bindings are declared through the forensics-readers module.
- The remote upstream allowance: the stack opts in
  (`vpcsc_upstream_allowance`), declaring the zone-level registry-platform
  singleton that permits the remote repositories' configured upstreams inside
  the perimeter. The platform default is deny; the declaration binds the
  exactly pinned `google-beta` provider because the pinned GA provider carries
  no resource for the surface; no other zone declares the allowance (zone
  purity).
- The recovery identity of the intake zone: the stack declares the dedicated
  identity through the recovery module — its elevated capability exists only
  as the declared, dormant privileged-access entitlement (never a standing
  grant), it is never used in normal operation and never federated from CI,
  and it holds no data-plane grant. Every activation is approval- and
  justification-bound and time-boxed by the platform-enforced grant duration;
  the duration and the approver set are approved instance decisions, supplied
  through the `break_glass_recovery` input.
- The stack consumes the zone state home — the dedicated state bucket of the
  zone holding that zone's root states and nothing else — through the final
  `gcs` backend binding with the state-key grammar prefix identifying exactly
  this root; the bucket is provisioned by the converged foundation, never by
  this stack and never by hand, and every state and plan artifact of this
  root is client-side encrypted through the engine layer of the dual
  fortress state-encryption standard, fail-closed enforced.

## Inputs

`project_id`, `project_number` (the instance-bound numeric project number bound by the zone workload identity pool — the pool's provider state carries the number, never the ID), `location`, `ecosystems` (default `["go"]`), `pool_id`,
`fetcher` (OIDC bindings of the intake fetcher), `additional_reader_members`
(the matrix-bound control-plane readers), `workload_job_images`
(the instance-bound image digests keyed by canonical job name),
`enabled_workload_jobs` (the instance-bound activation set of the zone's
workload jobs: exactly the bound jobs plus the jobs being provisioned in the
current window), `break_glass_recovery` (the approved recovery binding of the
intake zone: the per-activation grant duration of the recovery entitlement
and the approver principal set of its approval workflow), `workload_network` (the instance-bound zone VPC names and
CIDR),
`forensics_group` (the instance-bound forensics reader group),
`evidence_bucket_name`, `state_bucket_name` (the instance-bound zone state
home bucket, provisioned by the converged foundation — never by this stack),
`state_encryption_key` (the instance-bound engine key reference of this
root's client-side state and plan encryption), audit sink settings and
`policy_constraints`.

## Outputs

Repository IDs and URIs per ecosystem, the fetcher service account email and
its invoke-only trigger identity email, the pool resource name, the audit
sink writer identity, the enforced policy constraints, the workload job
resource IDs keyed by canonical job name, the workload network and
subnetwork resource IDs, the recovery identity email and the recovery
entitlement resource name.
