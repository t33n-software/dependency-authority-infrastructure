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
  archive, private DNS and time-bounded break-glass recovery;
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
`.terraform.lock.hcl`. The decision rationale lives in
`docs/conventions/infrastructure-as-code/`.

## Quality gates

```text
gofmt
go test ./...
go run -mod=readonly ./cmd/check-coverage
go run -mod=readonly ./cmd/build
```

Every executable Go package must reach exactly 100.0% statement coverage.
`cmd/build` additionally enforces lint (staticcheck), fail-closed
vulnerability analysis (govulncheck), Lefthook configuration validation, and
the OpenTofu gates: engine version verification, recursive format check, and
`init` plus `validate` for every module, the policy bindings and every
reference stack with enforced provider GPG validation.

The Go toolchain is pinned exactly (`toolchain go1.26.6`,
`GOTOOLCHAIN=local`); no lane downloads a toolchain at build time. Build tools
live in the pinned `tools/` module. CI re-runs the full gate daily so newly
disclosed vulnerabilities fail closed even without source changes.

## Repository layout

- `modules/` contains the seven generic modules.
- `policy-bindings/` contains the project-level policy module.
- `stacks/` contains the five trust-zone reference stacks.
- `cmd/` contains the build and coverage gates.
- `internal/packaging/` contains the same-package workflow contract tests.
- `docs/` contains architecture, conventions, and development
  documentation.

## Governance

Governed changes land through ticket branches and pull requests into
`develop`. `main` is the production and control-plane truth. Branch
governance is bound through the organization-level rule-sets; see
`docs/conventions/hosting-plattform/github/rule-sets/` for the canonical
source and the rule-set family of this repository.
