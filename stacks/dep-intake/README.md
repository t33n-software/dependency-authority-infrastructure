# Stack: dep-intake

Reference stack of the intake trust zone: remote intake repositories per
ecosystem, the intake fetcher workload identity, repository-scoped writer
binding, project-level policy compensation, the audit export into the
evidence archive, the zone workload network origin (one VPC with one Private
Google Access subnetwork in the job region, the restricted-range DNS response
policy and the egress firewall pair) and the zone's workload job of the
in-perimeter execution substrate (`dep-intake-fetch`, executed as the intake
fetcher identity and invoked only through its dedicated invoke-only trigger
identity `dep-intake-fetch-trigger`). The stack additionally declares the
read-only diagnostic bindings of the forensics reader access class on the zone
project.

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

## Inputs

`project_id`, `location`, `ecosystems` (default `["go"]`), `pool_id`,
`fetcher` (OIDC bindings of the intake fetcher), `additional_reader_members`
(the matrix-bound control-plane readers), `workload_job_images`
(the instance-bound image digests keyed by canonical job name),
`workload_network` (the instance-bound zone VPC names and CIDR),
`forensics_group` (the instance-bound forensics reader group),
`evidence_bucket_name`, audit sink settings and `policy_constraints`.

## Outputs

Repository IDs and URIs per ecosystem, the fetcher service account email and
its invoke-only trigger identity email, the pool resource name, the audit
sink writer identity, the enforced policy constraints, the workload job
resource IDs keyed by canonical job name and the workload network and
subnetwork resource IDs.
