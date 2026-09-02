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
   retention archive, private DNS and time-bounded break-glass recovery. It
   owns the project-level organization-policy compensation under
   `policy-bindings/` and the parameterized reference stacks under `stacks/`
   for the five trust zones.
2. Infrastructure as code is written in HCL and executed exclusively with
   OpenTofu. The engine and the Google provider are exactly pinned, provider
   GPG validation is enforced everywhere, and every reference stack commits
   its `.terraform.lock.hcl`. The decision rationale lives in
   `docs/conventions/infrastructure-as-code/`.
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
   dep-intake-fetcher           intake    writer on *-dependencies-intake;
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
   `roles/run.invoker` (carrying the invoke permission `run.jobs.run`) and
   `roles/run.viewer` (carrying the execution status read-back
   `run.executions.get` and `run.executions.list`) resource-scoped to exactly
   its own job and no other grant anywhere — no data-plane role, no
   project-level invoke permission and no shared trigger identity across
   lanes. The execution identity is never federated from CI and keeps the
   data-plane matrix of item 7. The role contents are proven against the
   provider (`gcloud iam roles describe`), never assumed; the remaining
   viewer permissions do not apply to a job resource.

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
