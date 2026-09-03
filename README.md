# Dependency Authority Infrastructure

`dependency-authority-infrastructure` is the organization-agnostic core of
generic infrastructure modules and parameterized reference stacks for the
dependency authority trust zones: control, intake, quarantine, approved, and
evidence.

This repository never contains concrete organization, tenant, project,
identity, network, secret, or registry bindings. Instances consume these
modules only through exact version pins.

## Core boundary

The core owns:

- the generic modules under `modules/` for artifact repositories,
  repository-scoped IAM, workload identity, audit logging, the evidence
  archive, the workload jobs of the in-perimeter execution substrate,
  private DNS and the zone workload network origin, and time-bounded
  break-glass recovery;
- the project-level organization-policy compensation under
  `policy-bindings/`;
- the parameterized reference stacks under `stacks/` for the five trust
  zones: `dep-control`, `dep-intake`, `dep-quarantine`, `dep-approved` and
  `dep-evidence`.

The core never contains:

- concrete organization or tenant values;
- credentials, tokens, private keys, or authorization headers;
- live state, plans, or variable binding files (`*.tfstate`, `*.tfvars`).

## Toolchain

Infrastructure as code is written in HCL and executed exclusively with
OpenTofu. The engine and the Google provider are exactly pinned, provider GPG
validation is enforced, and every reference stack commits its
`.terraform.lock.hcl`. The OpenTofu toolchain provisioning and gates are owned
by the `opentofu` capability pack, declared through the `extends` list of the
quality configuration seam; the pack contract lives in the shared-kernel
registry under `capabilities/infrastructure/opentofu/`.

## Quality gates

```text
gofmt
go test ./...
go tool -modfile tools/go.mod check-coverage
go tool -modfile tools/go.mod quality-gate
```

Every executable Go package must reach exactly 100.0% statement coverage.
`quality-gate` runs the canonical gate chain of the go-quality-authority
territory home through the pinned tooling module: Go formatting, module
checksums and metadata, the pinned build tool module, lint (staticcheck), unit
tests, exact 100% statement coverage, race detector, static analysis,
fail-closed vulnerability analysis (govulncheck), and Lefthook configuration
validation. The OpenTofu gates — engine version verification, recursive format
check, and `init` plus `validate` for every module, the policy bindings and
every reference stack with enforced provider GPG validation — are owned by the
`opentofu` capability pack and run in the canonical quality lane.

The CI surface is the canonical thin callers of the repository-governance
home (`ci.yml`, `codeql.yml`, `dependency-review.yml`) plus the
`canonical-conformance.yml` lane, which proves the bindings of this
repository against the home fail-closed. The canonical quality lane
provisions the declared capability packs before the gate runs.

The Go toolchain is pinned exactly (`toolchain go1.26.6`,
`GOTOOLCHAIN=local`); no lane downloads a toolchain at build time. Build tools
live in the pinned `tools/` module. CI re-runs the full gate daily so newly
disclosed vulnerabilities fail closed even without source changes.

## Repository layout

- `modules/` contains the eight generic modules.
- `policy-bindings/` contains the project-level policy module.
- `stacks/` contains the five trust-zone reference stacks.
- `internal/packaging/` contains the same-package workflow contract tests.
- `docs/` contains architecture, conventions, and development
  documentation.
- `repo-bindings.json` is the tenant binding manifest of the canonical
  adoption: the home pin, the caller hashes, the canonical file bindings, and
  the config-seam and tooling-module pins.

## Governance

Governed changes land through ticket branches and pull requests into
`develop`. `main` is the production and control-plane truth. Branch
governance is bound through the organization-level rule-sets; see
`docs/conventions/hosting-plattform/github/rule-sets/` for the canonical
source and the rule-set family of this repository.
