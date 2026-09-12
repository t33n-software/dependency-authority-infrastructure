# Stack: dep-evidence

Reference stack of the evidence trust zone: generic evidence repositories per
ecosystem, the long-term immutable retention archive, the evidence writer and
auditor workload identities, repository-scoped bindings, project-level policy
compensation, the zone's own audit export into the archive, the zone workload
network origin (one VPC with one Private Google Access subnetwork in the job
region, the restricted-range DNS response policy covering `*.googleapis.com`
and the Artifact Registry data plane `*.pkg.dev`, and the egress firewall
pair) and the zone's workload jobs of the in-perimeter execution substrate
(`dep-evidence-write` and `dep-evidence-audit`, executed as the writer and
auditor identities and invoked only through their dedicated invoke-only
trigger identities `dep-evidence-write-trigger` and
`dep-evidence-audit-trigger`). The stack additionally declares the read-only
diagnostic bindings of the forensics reader access class on the zone project.

## Boundary

- Evidence is append-only: the writer appends, the auditor reads, and no
  identity receives routine delete authority. The only additional writers and
  readers are the identities of the canonical IAM target matrix (canonically
  the intake fetcher writing its candidate records, the admission,
  revalidation, revocation and promotion lanes writing — the promotion lane
  writes its approved record into the evidence repository), wired through the
  instance-supplied member inputs.
- The retention archive pairs the operational evidence repositories; a
  deletable repository version alone is not a long-term evidence control.
- `lock_retention_policy` stays `false` until the retention and legal-hold
  duration is approved and recorded by the organization instance; locking is
  irreversible.
- The stack creates no project and enables no APIs; the organization instance
  provisions the project and its API surface first.
- Provisioning order: this stack lands first, because every other zone exports
  its audit logs into the archive bucket created here.
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
  `roles/run.viewer` on this zone project and no other grant — the read-only
  diagnostic bindings are declared through the forensics-readers module.

## Inputs

`project_id`, `location`, `ecosystems` (default `["go"]`), `pool_id`,
`writer` and `auditor` (OIDC bindings), `workload_job_images` (the
instance-bound image digests keyed by canonical job name), `workload_network`
(the instance-bound zone VPC names and CIDR), `archive_bucket_name`,
`retention_period_seconds`, `lock_retention_policy`,
optional `archive_kms_key_name`, the matrix-bound `additional_writer_members`
and `additional_auditor_members`, `forensics_group` (the instance-bound
forensics reader group), audit sink settings and
`policy_constraints`.

## Outputs

Evidence repository IDs per ecosystem, the archive bucket name (consumed by
the other zone stacks), writer and auditor service account emails and their
invoke-only trigger identity emails, the pool resource name, the audit sink
writer identity, the enforced policy constraints, the workload job resource
IDs keyed by canonical job name and the workload network and subnetwork
resource IDs.
