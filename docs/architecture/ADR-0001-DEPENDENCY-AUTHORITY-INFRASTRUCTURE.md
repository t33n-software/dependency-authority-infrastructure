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
   dep-approved-promoter        control   reader on *-dependencies-intake and
                                          *-dependencies-evidence;
                                          writer on *-dependencies-approved;
                                          reader on release-controller-images
   dep-revalidation-controller  control   reader on *-dependencies-approved;
                                          writer on *-dependencies-evidence;
                                          reader on release-controller-images
   dep-revocation-controller    control   writer on *-dependencies-approved
                                          (the revocation download rules stay
                                          runtime operations) and
                                          *-dependencies-evidence;
                                          reader on release-controller-images
   dep-evidence-writer          evidence  writer on *-dependencies-evidence;
                                          reader on release-controller-images
   dep-evidence-auditor         evidence  reader on *-dependencies-evidence;
                                          reader on release-controller-images
   dep-break-glass-recovery     control   no data-plane grant: time-bounded
                                          break-glass recovery only, never
                                          federated from CI
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
