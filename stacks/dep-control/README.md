# Stack: dep-control

Reference stack of the control trust zone: the control-plane controller
workload identities (admission, promotion, revalidation and revocation
lanes), the zone workload identity pool, project-level policy compensation,
the audit export into the evidence archive, the workload image registries of
the in-perimeter execution substrate, the zone workload network origin (one
VPC with one Private Google Access subnetwork in the job region, the
restricted-range DNS response policy covering `*.googleapis.com` and the
Artifact Registry data plane `*.pkg.dev`, and the egress firewall pair) and the
zone's four workload jobs (`dep-admission`, `dep-promotion`,
`dep-revalidation`, `dep-revocation`), each executed as the existing zone
workload identity of its lane and invoked only through its dedicated
invoke-only trigger identity (`dep-<operation>-trigger`). The stack also
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
  zone lane identity (the four control-plane lanes here, the other zones
  through the instance-supplied member input) receives read access on the
  release class; no identity ever receives a writer grant on either class,
  and the staging class carries no binding at all.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- The audit export requires the dep-evidence archive bucket; provisioning
  order is dep-evidence first.
- The workload jobs consume their images by full immutable digest from the
  release-class workload image registry only; the digests are instance
  bindings (`planned` with documented placeholders until the promotion
  read-back proofs flip them to `bound`), never stack defaults.
- The workload jobs attach to the zone VPC declared by this stack and route
  all outgoing traffic through it (Direct VPC egress, all-traffic): the
  workload network origin is part of the execution contract, and the stack
  enforces the form through the Cloud Run organization policies
  (`run.allowedVPCEgress` allows only all-traffic, `run.allowedIngress`
  allows only internal). A job without the zone network attachment presents
  no in-perimeter network origin and fails closed at the perimeter.
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

## Inputs

`project_id`, `location`, `pool_id`, `controllers` (OIDC bindings of the
controller identities), `workload_job_images` (the instance-bound image
digests keyed by canonical job name), `workload_network` (the instance-bound
zone VPC names and CIDR), `cross_zone_workload_reader_members`
(the matrix-bound readers of the other zones), `forensics_group` (the
instance-bound forensics reader group), `perimeter_ingress` (the
instance-bound perimeter rule of the forensics reader access class, declared
exactly once here), `evidence_bucket_name`, audit sink settings and
`policy_constraints`.

## Outputs

Controller service account emails and their invoke-only trigger identity
emails keyed by lane, the pool resource name, the audit sink writer
identity, the enforced policy constraints, the workload image repository IDs
and endpoint URIs keyed by class (`staging`, `release`), the workload job
resource IDs keyed by canonical job name and the workload network and
subnetwork resource IDs.
