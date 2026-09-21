# ADR-0001: Dependency Authority Infrastructure Core

## Status

Accepted

## Context

The federated multi-tenant supply chain architecture requires a dependency
authority: one logical bounded context with five physical trust zones
(`control`, `intake`, `quarantine`, `approved`, `evidence`) that governs how
package dependencies enter an organization, are admitted, promoted into the
approved zone, revalidated, and revoked. Without an organization-agnostic
infrastructure core, every organization would re-implement the zone topology,
identity classes and evidence boundaries and drift from the canonical
architecture.

## Decision

This repository is the organization-agnostic dependency authority
infrastructure core.

1. It owns the generic modules under `modules/`: artifact repositories,
   repository-scoped IAM, workload identity, audit logging, the evidence
   retention archive, private DNS, time-bounded break-glass recovery and the
   forensics reader access class bindings. It
   owns the project-level organization-policy compensation under
   `policy-bindings/` and the parameterized reference stacks under `stacks/`
   for the five trust zones.
2. Infrastructure as code is written in HCL and executed exclusively with
   OpenTofu. The engine and the Google providers are exactly pinned, provider
   GPG validation is enforced everywhere, and every reference stack commits
   its `.terraform.lock.hcl`. The engine decision rationale lives in the
   shared-kernel capability-pack contract (the `opentofu@1` pack); this
   core's provider-binding rules live in `docs/conventions/provider-binding/`.
3. This core never contains concrete organization bindings.
   This core never contains tenant bindings.
   It contains no credentials, tokens, private keys, or authorization
   headers, and no live state, plans, or variable binding files.
4. Zone semantics stay physically separated: the intake stack binds remote
   upstream repositories, the quarantine and approved stacks bind standard
   repositories, and the evidence stack binds generic evidence repositories
   plus the retention archive. The approved zone is the only consumer
   endpoint; intake and quarantine never serve consumers. The control stack
   additionally binds the workload image registries of the in-perimeter
   execution substrate: DOCKER standard repositories of the classes staging
   (the only governed producer delivery target) and release (the only
   workload consumption source, filled exclusively through promotion of a
   proven staging digest), never remote and never dependency repositories.
5. Revocation download rules (`google_artifact_registry_rule`) are runtime
   operations of the revocation controller and are intentionally not part of
   the static stacks. A revoked package or version is denied before a
   consumer receives it; that behavior is evidence-bound runtime evidence,
   not static configuration.
6. Instances consume this core only through the three-pin consumption
   contract: module version pins for infrastructure, artifact digest pins for
   runtime, and schema version pins for policies and evidence. Instances wire
   the concrete project IDs, regions, OIDC bindings, members and retention
   values as reviewed instance configuration.
7. The canonical IAM target matrix binds the data-plane roles of the eight
   zone workload identities at repository scope through
   `modules/repository-iam`, never at project scope, and cross-zone
   authority only through the owning zone's instance-wired member inputs:

   ```text
   identity                     zone      repository-scoped grants
   dep-intake-fetcher           intake    writer on *-dependencies-intake and
                                           *-dependencies-evidence (the intake
                                           use case writes its candidate
                                           records into the evidence
                                           repository);
                                           reader on release-controller-images
   dep-admission-controller     control   reader on *-dependencies-intake;
                                          writer on *-dependencies-evidence;
                                          reader on release-controller-images
   dep-approved-promoter        control   reader on *-dependencies-intake;
                                           writer on *-dependencies-approved
                                           and *-dependencies-evidence (the
                                           promotion writes its approved
                                           record into the evidence
                                           repository);
                                           reader on release-controller-images
   dep-revalidation-controller  control   reader on *-dependencies-intake (the
                                           revalidation lane materializes the
                                           candidate content from the
                                           controlled intake boundary) and
                                           *-dependencies-approved;
                                           writer on *-dependencies-evidence;
                                           reader on release-controller-images
   dep-revocation-controller    control   writer on *-dependencies-approved
                                           (the revocation download rules stay
                                           runtime operations) and
                                           *-dependencies-evidence;
                                           reader on release-controller-images
   dep-consumer-verifier        control   reader on *-dependencies-approved
                                           (the consumer read path of the
                                           consumer verification lane);
                                           writer on *-dependencies-evidence
                                           (the consumer verification writes
                                           its lane evidence into the evidence
                                           repository);
                                           reader on release-controller-images
   dep-evidence-writer          evidence  writer on *-dependencies-evidence;
                                          reader on release-controller-images
   dep-evidence-auditor         evidence  reader on *-dependencies-evidence;
                                          reader on release-controller-images
   dep-break-glass-recovery     all zones no data-plane grant: time-bounded
                                          break-glass recovery only, never
                                          federated from CI — one dedicated
                                          identity per zone project
   ```

   The matrix binds its exclusions with the same force: the quarantine
   repositories carry no routine identity grant (investigation is an
   operator-bound activity through the governed operator channel, and any
   future quarantine writer is a governed change); no identity ever holds a
   writer grant on either workload image registry class; the staging class
   carries no binding at all; the scanner invocation of the admission
   controller is an external API call and carries no plane grant; and the
   approved zone declares no zone-local workload identity by default, because
   the approved promoter is a control-zone identity bound through the member
   inputs of the approved stack.
8. The external trigger seam binds a dedicated invoke-only trigger identity
   per lane operation (`dep-<operation>-trigger`): the lane's
   environment-scoped principal set federates to the trigger identity, never
   to the execution identity, and the trigger identity holds
   `roles/run.jobsExecutorWithOverrides` (carrying `run.jobs.run`,
   `run.jobs.runWithOverrides` and `run.executions.cancel`) and
   `roles/run.viewer` (carrying the execution status read-back
   `run.executions.get` and `run.executions.list`) resource-scoped to exactly
   its own job and no other grant anywhere — no data-plane role, no
   project-level invoke permission and no shared trigger identity across
   lanes. The lane passes the operation inputs as execution-parameter
   overrides of the invocation, so the invoke call is an override execution
   that requires `run.jobs.runWithOverrides`; the plain `roles/run.invoker`
   form is insufficient for it. The execution identity is never federated
   from CI and keeps the data-plane matrix of item 7. The role contents are
   proven against the provider (`gcloud iam roles describe`), never assumed;
   the remaining viewer permissions do not apply to a job resource.
9. The workload network origin is part of the execution contract: every
   job-owning zone (intake, control, evidence) declares exactly one VPC with
   one subnetwork in the job region carrying Private Google Access through
   the network module, and every workload job attaches to its zone VPC with
   Direct VPC egress routing all outgoing traffic through it (the
   `cloud-run-job` module hardcodes the egress setting `ALL_TRAFFIC` and
   takes the network and subnetwork as mandatory, fail-closed validated
   instance inputs). A serverless job without the zone network attachment
   presents no in-perimeter network origin: its calls to the restricted
   planes are evaluated as external to the perimeter and fail closed, and the
   zone-project membership of the workload does not by itself place its calls
   inside — the network origin is the third perimeter dimension beside
   identity and resource. The zone VPC carries the restricted-range DNS
   response policy (`*.googleapis.com` and the Artifact Registry data-plane
   domain `*.pkg.dev` resolve to `restricted.googleapis.com`,
   `199.36.153.4/30`; the registry domains are served by the restricted VIP,
   and without the mapping the workload's registry calls resolve to public
   addresses and fail against the deny-all egress rule) and exactly two
   egress firewall rules (allow TCP 443 to the restricted range ordered
   before priority 1000, deny all egress ordered after priority 1000 — the
   allow rule targets the restricted range rather than a domain, so it
   already covers the data plane), and the project-level policy compensation
   restricts
   Cloud Run to exactly this form (`run.allowedVPCEgress` allows only
   `all-traffic`, `run.allowedIngress` allows only `internal`), so the
   platform enforces the form rather than convention alone. The quarantine
   and approved zones carry no workload network: they own no jobs (zone
   purity).
10. The forensics reader access class binds the raw-substrate forensics
    capability of the operator access classes: the organization-owned
    forensics group (the canonical identity class `dep-forensics-readers`,
    created and membership-managed on the organization identity plane, never
    by this core) holds exactly `roles/logging.viewer` and `roles/run.viewer`
    project-scoped on every zone project through the `forensics-readers`
    module — never organization-scoped — and no other grant on any plane, so
    the read-only form is enforced by the granted roles rather than by
    convention. The second, separate perimeter ingress rule carries the
    forensics group as the only identity, scoped to the read-only logging
    method `LoggingServiceV2.ListLogEntries` (the platform-supported
    method-selector form for Cloud Logging) with the zone projects as
    resources, entering through the same identity-bound channel as the
    administration rule; the administration ingress rule never carries the
    forensics identity, and every additional read method is a governed
    change to the rule. The execution status read-back travels the deliberately
    non-restricted compute control plane and needs no perimeter rule. The
    boundary-level rule is declared exactly once by the control-zone stack
    (the boundary governance zone); every other stack binds only the zone
    bindings of the class. The forensics group member string, the perimeter
    resource name and the zone project numbers are instance bindings, never
    core literals.
11. The provider supply chain of the declaration plane carries exactly two
    exactly pinned providers of the same publisher and signing trust anchor:
    `hashicorp/google` (the generally available provider) and
    `hashicorp/google-beta`. The beta provider is introduced for exactly one
    proven case: the Artifact Registry VPC Service Controls configuration
    (`google_artifact_registry_vpcsc_config`) exists only in the beta
    provider, while the underlying API is generally available and the pinned
    GA provider schema provably carries no resource for it (the proof is
    produced against the pinned provider schema, never assumed). The beta
    provider declares a resource only when the pinned GA provider provably
    lacks it; the packaging contract binds this scope fail-closed. The
    artifact-registry module declares the remote upstream allowance as a
    null-gated opt-in surface (`vpcsc_upstream_allowance`, default fail-closed
    denied, remote mode required) binding the configuration with
    `vpcsc_policy = "ALLOW"` scoped to the project and location; only a zone
    with a remote repository inside the perimeter opts in — canonically the
    intake zone — and no other zone carries the allowance (zone purity). The
    allowance is the zone-level registry-platform singleton, covers only the
    configured upstreams of the zone's remote repositories and is never a
    perimeter egress rule. When the GA provider carries the resource, a
    governed change flips the provider reference and removes the beta
    provider once its last resource is gone. The standing rule lives in
    `docs/conventions/provider-binding/beta-stage-resources.md`.
12. The state architecture of the declaration plane follows the operating
    model: one state per root, and every trust zone owns exactly one
    dedicated state home — the dedicated state bucket of the zone holding
    that zone's root states and nothing else. The provisioning ownership of
    every zone state home sits with the foundation layer: the converged
    foundation provisions every zone state home through the plan-gated
    apply, never by the zone's own roots and never by hand, so no zone root
    ever applies with local state and every zone root carries its backend
    configuration final from birth. Every zone stack binds the `gcs`
    backend into the instance-bound zone state bucket with the state-key
    grammar prefix identifying exactly the root, and the engine layer of
    the dual fortress state-encryption standard (the `gcp_kms` key provider
    with `key_length = 32`, the `aes_gcm` method, `enforced = true` on
    state and plan, the `remote_state_data_sources` default read edge and
    the stable `encrypted_metadata_alias`), so every state and plan
    artifact of every root is client-side encrypted before it reaches any
    backend. The concrete bucket names and key references are instance
    bindings without defaults; the keys exist before any apply through the
    governed operator channel and converge into engine management with the
    foundation convergence. The bucket form itself — object versioning,
    uniform bucket-level access, enforced public access prevention and the
    mandatory bucket CMEK with the second, cryptographically separate key
    (never the engine key), plus exactly the operator object-admin data
    plane, and never a retention policy, because the state layer is the
    recovery root, not an archive — is declared by the foundation layer of
    the developer platform infrastructure core, never by this core.
13. The workload job surfaces of the job-owning stacks follow the
    engine-active surface composition of the operating model: every
    job-owning stack (dep-intake, dep-control, dep-evidence) consumes the
    instance-bound activation set through the required
    `enabled_workload_jobs` input (never a stack default) and filters the
    declared job topology through it
    (`for_each = { for job, spec in local.workload_jobs : job => spec if contains(var.enabled_workload_jobs, job) }`),
    so a declared-but-planned job never enters the plan until its
    provisioning window activates it — the convergence zero-drift proof and
    the steady-state drift-detection cadence read empty for
    declared-but-planned surfaces instead of reporting the declaration
    chain's forward path as drift. The declaration binds the consistency
    fail-closed: the activation set references only declared jobs of the
    zone topology (proven against the pinned engine), and the module's
    fail-closed image validation keeps an activated job always on a proven
    immutable digest. The activation of a planned surface is a reviewed
    change of the instance binding — the same governed form as every
    binding change. Every zone stack declares the recovery identity of its
    zone through the recovery module (decision 15): the dedicated identity
    whose elevated project role exists only under the mandatory time-bound
    IAM condition, never federated from CI and never carrying a data-plane
    grant, with the role and the end time as approved instance decisions
    supplied through the `break_glass_recovery` input.
14. The workload image registries carry the declared, platform-executed
    workload image lifecycle of the workload image lifecycle and retention
    convention: the artifact-registry module owns the cleanup policy surface
    (the `cleanup_policies` map keyed by policy ID and the
    `cleanup_policy_dry_run` flag, defaulting to the fail-safe dry-run
    posture) and binds the class rule fail-closed — only DOCKER workload
    image repositories carry cleanup policies, because the dependency
    repositories and the evidence plane are append-only supply-chain
    records. The policy block form is proven against the pinned provider
    schema (google 7.44.0) through `tofu providers schema -json`, never
    assumed: `tag_state` binds `TAGGED`, `UNTAGGED` or `ANY`,
    `most_recent_versions` pairs only with the `KEEP` action, and
    `older_than`/`newer_than` carry duration strings. The control stack
    binds the two registries through the required instance-bound
    `workload_image_cleanup` input (never a stack default): the per-class
    policy content and the dry-run activation state, with the convention's
    structural rules proven fail-closed — the release class carries keep
    policies only and never a time-based deletion, and the staging class
    carries the time-based delete plus the keep floor. The concrete values
    (30 days, keep 2, keep 5) are the canonical convention values, bound by
    the organization instance with binding status planned until the
    list-cleanup-policies read-back proof flips them to bound; the dry-run
    state flips to active only through the governed activation window after
    the dry-run proof. No other stack ever carries the binding (zone
    purity), and the behavioral proofs live beside the code: the module
    fixture proves the class rule offline, and the stack fixture proves the
    structural rules in the governed execution window.
 15. Every trust zone carries its own break-glass recovery identity — five
     zones, five dedicated identities, exactly one per zone project, each
     declared by the zone's own stack through the recovery module and bound
     to that project. The identity is standing but dormant: never used in
     normal operation, never federated from CI and never carrying a
     data-plane grant; its elevated project role exists only under the
     mandatory time-bound IAM condition, and an unbounded grant is a
     contract violation. The role and the end time are approved instance
     decisions supplied through the `break_glass_recovery` input of every
     zone stack (never stack defaults), and every re-binding of the end
     time is a reviewed instance change. The identity is not a fourth
     operator access class: the JIT window is the planned mutation path,
     the break-glass identity is the disaster path. A zone never depends on
     another zone's recovery surface, because the disaster form is by
     definition unknown and the recovery guarantee must be total within the
     zone boundary; the behavioral rejection proofs of the binding live
     beside every stack.
 16. Every managed resource of the declaration plane declares the canonical
     human-readable description surface of its provider schema (the mandatory
     description duty of the mandatory resource properties convention): the
     network module exposes the description surfaces of the workload network
     origin as optional inputs (the VPC and the subnetwork as create-only
     surfaces, the egress firewall pair and the restricted-range DNS response
     policy as in-place surfaces — the DNS response policy rules expose no
     description surface in the pinned provider schema, proven through `tofu
     providers schema -json`, and are never bound), and the control stack
     binds them as required instance-supplied values, never stack defaults:
     the create-only surfaces byte-exact to the live values at the
     convergence window (a missing or diverging declaration there forces
     recreation and is a blocking defect, never a cosmetic drift), the
     in-place surfaces as the canonical forms normalized through the reviewed
     plan-gated change. The workload-identity module binds the trigger
     identity display name to its identity class name, and the control stack
     binds the pool display and description surfaces, the audit sink
     description and the per-controller display name and description —
     required and validated fail-closed. The forbidden escape forms of the
     convention stay forbidden: no `lifecycle ignore_changes` on a mandatory
     property, and no computed description form on a create-only surface. The
     behavioral proofs live beside the code: the module fixture proves the
     binding offline, and the stack fixture proves the fail-closed form in
     the governed execution window.

## Consequences

- Every module, policy binding and stack change is a governed, reviewable
  change verified by the source-quality gate: formatting, module integrity,
  tests, exact 100% statement coverage for the Go gates, race detection,
  static analysis, the OpenTofu version gate, recursive `tofu fmt` checks and
  `init` plus `validate` for every root.
- Organization and tenant instances bind the reference stacks through their
  own reviewed values and prove the binding in their own evidence.
- The core never references a concrete organization or tenant; instances
  reference only the core.
- The `release/*` and `support/*` branch families and their Rulesets are
  activated only with a complete governed release lifecycle.
