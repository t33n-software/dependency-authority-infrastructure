# Verification

This document binds the local and CI verification contract for the dependency
authority infrastructure core. It exists so every instance and every reviewer
can re-run the same gates.

## Local gates

Run from the repository root with the pinned Go toolchain (`go 1.26`,
`toolchain go1.26.6`):

```text
gofmt
go test ./...
go run -mod=readonly ./cmd/check-coverage
go run -mod=readonly ./cmd/build
```

`cmd/build` is the full source-level gate: Go formatting, module checksums,
module metadata, build tool download, build tool checksums, build tool
metadata, lint (staticcheck), unit tests, exact 100% statement coverage,
race detector, static analysis, fail-closed vulnerability analysis
(govulncheck), Lefthook configuration validation, and the Linux/AMD64 build
of the gate binaries with module provenance.

The OpenTofu gates are pack-owned: the canonical quality lane resolves the
declared `opentofu@1` pack, provisions the pinned engine through its digest-
and signature-bound recipe (`quality-gate provision`), and runs the engine
version assertion, the recursive `tofu fmt -check`, and
`tofu init -backend=false` plus `tofu validate` for every discovered HCL
root — the modules, the policy bindings, and every reference stack. A local
pack-gate run uses the same two commands and requires the cosign verifier on
the lane for the signature proof.

## Build tooling

Build tools live in the separate `tools/` module and are resolved through
its own verified `go.mod` and committed `go.sum`; they never join the source
module graph. The module pins `govulncheck`, `staticcheck`, and `lefthook`
via the Go tool directive and shares the repository toolchain pin. It also
pins the governed quality tools — `quality-gate` and `check-coverage` from
the go-quality-authority home and `verify-canonical` from the
repository-governance home — and requires the supply-chain-governance shared
kernel, so the capability-pack registry resolves at the pinned stand.

## Toolchain and vulnerability re-scans

The Go toolchain is pinned exactly (`toolchain go1.26.6`,
`GOTOOLCHAIN=local`); CI asserts `go env GOVERSION` before any gate runs and
no lane may download a toolchain at build time. `govulncheck` is part of the
full source gate, and CI re-runs the full gate on a daily schedule so newly
disclosed vulnerabilities in the pinned toolchain or dependency graph fail
closed even without source changes.

The OpenTofu gates run with the pack-bound environment
(`OPENTOFU_ENFORCE_GPG_VALIDATION=true`, `TF_IN_AUTOMATION=true`,
`TF_INPUT=false`). Provider downloads land in the repository-local plugin
cache under `.build/`, which is never committed. Module and policy-binding
lock files stay local; every reference stack commits its
`.terraform.lock.hcl`.

## Test architecture

Every executable Go package carries same-package whitebox tests for its
invariants, branches, state transitions, errors, and cleanup paths.
`internal/packaging/` complements them with same-package workflow contract
tests that bind the canonical caller byte-identity and pins to the tenant
binding manifest, the canonical file family, the conformance lane, the
capability-pack declaration, the organization rule-set adoption guard, the
module and stack layout, the exact OpenTofu pins and the core boundary (no
concrete organization, tenant, project, identity, network, secret or
registry bindings anywhere in the core).

## CI gates

The `Quality gates / linux-amd64` check runs the canonical quality gate of
the repository-governance home with the pinned Go toolchain and the
provisioned `opentofu@1` pack on every push and pull request to the shared
lines and once per day on a schedule. The `Canonical conformance` check
proves the canonical bindings of this repository fail-closed. The
`Dependency review / Dependency admission review` check blocks unreviewed
dependency changes. CodeQL code scanning runs with all alerts blocking; the
binding of this repository is documented in
`docs/conventions/hosting-plattform/github/rule-sets/`.

Lefthook provides the local `commit-msg` hook (governed commit-message
validation) and the canonical pre-push validation.

## Instance consumption

An organization instance consumes this core through exact module version pins
(for example a git source with an immutable `?ref=` version) and supplies the
concrete project IDs, regions, OIDC bindings, members and retention values as
reviewed instance configuration. An instance never copies the module source
into its own boundary; it references the pinned version and proves the binding
in its own evidence.
