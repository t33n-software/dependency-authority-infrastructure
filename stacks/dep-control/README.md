# Stack: dep-control

Reference stack of the control trust zone: the control-plane controller
workload identities (admission, promotion, revalidation, revocation and
consumer-verification lanes), the zone workload identity pool, project-level
policy compensation, the audit export into the evidence archive, the workload
image registries of the in-perimeter execution substrate, the zone workload
network origin (one VPC with one Private Google Access subnetwork in the job
region, the restricted-range DNS response policy covering `*.googleapis.com`
and the Artifact Registry data plane `*.pkg.dev`, and the egress firewall
pair) and the zone's five workload jobs (`dep-admission`, `dep-promotion`,
`dep-revalidation`, `dep-revocation`, `dep-consumer-verification`), each
executed as the existing zone workload identity of its lane and invoked only
through its dedicated invoke-only trigger identity
(`dep-<operation>-trigger`). The stack also
carries the forensics reader access class: the read-only diagnostic bindings
on the zone project and — declared exactly once here — the second, separate
perimeter ingress rule of the class.

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
- The canonical IAM target matrix of the workload image registries: every
  zone lane identity (the five control-plane lanes here, the other zones
  through the instance-supplied member input) receives read access on the
  release class; no identity ever receives a writer grant on either class,
  and the staging class carries no binding at all.
- The two workload image registries carry the declared, platform-executed
  workload image lifecycle: the organization instance binds the canonical
  convention values (the staging class deletes image versions older than 30
  days and always keeps the most recent 2 per package; the release class
  always keeps the most recent 5 per package and never carries a time-based
  deletion) through the required `workload_image_cleanup` input — never a
  stack default — with binding status planned until the
  list-cleanup-policies read-back proof flips them to bound; the dry-run
  activation state starts true (the fail-safe posture) and flips to false
  only through the governed activation window after the dry-run proof.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- The audit export requires the dep-evidence archive bucket; provisioning
  order is dep-evidence first.
- The workload jobs consume their images by full immutable digest from the
  release-class workload image registry only; the digests are instance
  bindings (`planned` with documented placeholders until the promotion
  read-back proofs flip them to `bound`), never stack defaults.
- The workload jobs of the zone follow the engine-active surface composition
  of the operating model: the stack consumes the instance-bound activation
  set (`enabled_workload_jobs`) — exactly the bound jobs plus the jobs being
  provisioned in the current window — and a declared-but-planned job never
  enters the plan until its provisioning window activates it. The activation
   set always carries every bound job (a bound job dropped from the active set
   would plan its own destruction), and the declaration binds the consistency
   fail-closed.
- The declaration owns the static, non-credential configuration of every
  workload job completely (the workload configuration ownership convention):
  the stack consumes the instance-bound `workload_job_env` input — the proven
  static environment bindings of every job, keyed by the canonical job name —
  never a stack default. Every value referencing another bound surface is a
  proven projection the instance verifier cross-binds fail-closed against its
  canonical source. Operation inputs travel as validated execution parameters
  of the invocation, never as baked-in values, and credentials never travel
  this surface.
- The recovery identity of the control zone: the stack declares the dedicated
  identity through the recovery module — its elevated capability exists only
  as the declared, dormant privileged-access entitlement (never a standing
  grant), it is never used in normal operation and never federated from CI,
  and it holds no data-plane grant. Every activation is approval- and
  justification-bound and time-boxed by the platform-enforced grant duration;
  the duration and the approver set are approved instance decisions, supplied
  through the `break_glass_recovery` input.
- The workload jobs attach to the zone VPC declared by this stack and route
  all outgoing traffic through it (Direct VPC egress, all-traffic): the
  workload network origin is part of the execution contract, and the stack
  enforces the form through the Cloud Run organization policies
  (`run.allowedVPCEgress` allows only all-traffic, `run.allowedIngress`
  allows only internal). A job without the zone network attachment presents
  no in-perimeter network origin and fails closed at the perimeter.
- Every managed surface of the zone workload network origin binds its
  canonical human-readable description (the mandatory description duty of the
  mandatory resource properties convention): the VPC and subnetwork
  descriptions are create-only surfaces bound byte-exact to the live values
  at the convergence window, and the firewall pair and DNS response policy
  descriptions are in-place surfaces — all bound through the required
  `workload_network` input fields, never stack defaults. The zone workload
  identity pool and the audit sink carry their canonical intent descriptions,
  and every controller identity binds its display name and description as
  instance-bound values.
- Each lane federates to the dedicated invoke-only trigger identity of its
  job, never to the execution identity; a trigger identity holds invoke on
  exactly its own job and no data-plane grant.
- The forensics reader access class: the organization-owned forensics group
  (instance-supplied) holds exactly `roles/logging.viewer` and
  `roles/run.viewer` on this zone project and no other grant. This stack
  additionally declares the second, separate perimeter ingress rule of the
  class exactly once for the whole boundary: the forensics group as the only
  identity, scoped to the read-only logging method
  `LoggingServiceV2.ListLogEntries` with the zone projects as resources,
  through the same identity-bound channel as the administration rule.
- The stack consumes the zone state home — the dedicated state bucket of the
  zone holding that zone's root states and nothing else — through the final
  `gcs` backend binding with the state-key grammar prefix identifying exactly
  this root; the bucket is provisioned by the converged foundation, never by
  this stack and never by hand, and every state and plan artifact of this
  root is client-side encrypted through the engine layer of the dual
  fortress state-encryption standard, fail-closed enforced.

## Inputs

`project_id`, `project_number` (the instance-bound numeric project number bound by the zone workload identity pool — the pool's provider state carries the number, never the ID), `location`, `pool_id`, `controllers` (the OIDC bindings and the
canonical display name and description surfaces of the controller
identities), `workload_job_images` (the instance-bound image
digests keyed by canonical job name), `workload_image_cleanup` (the
instance-bound lifecycle binding of the two workload image registries: the
per-class cleanup policies and the dry-run activation state),
`enabled_workload_jobs` (the
instance-bound activation set of the zone's workload jobs: exactly the bound
jobs plus the jobs being provisioned in the current window),
`workload_job_env` (the instance-bound static environment bindings of the
zone's workload jobs, keyed by the canonical job name — the declaration owns
every static, non-credential configuration value completely; credentials
never travel this surface),
`break_glass_recovery` (the approved recovery binding of the control zone:
the per-activation grant duration of the recovery entitlement and the
approver principal set of its approval workflow), `workload_network` (the instance-bound
zone VPC names and CIDR plus the canonical description surfaces of every
managed network surface: the create-only VPC and subnetwork descriptions
bound byte-exact to the live values, the in-place firewall pair and DNS
response policy descriptions), `cross_zone_workload_reader_members`
(the matrix-bound readers of the other zones), `forensics_group` (the
instance-bound forensics reader group), `perimeter_ingress` (the
instance-bound perimeter rule of the forensics reader access class, declared
exactly once here), `evidence_bucket_name`, `state_bucket_name` (the instance-bound zone state
home bucket, provisioned by the converged foundation — never by this stack),
`state_encryption_key` (the instance-bound engine key reference of this
root's client-side state and plan encryption), audit sink settings and
`policy_constraints`.

## Outputs

Controller service account emails and their invoke-only trigger identity
emails keyed by lane, the pool resource name, the audit sink writer
identity, the enforced policy constraints, the workload image repository IDs
and endpoint URIs keyed by class (`staging`, `release`), the workload job
resource IDs keyed by canonical job name, the workload network and
subnetwork resource IDs, the recovery identity email and the recovery
entitlement resource name.
