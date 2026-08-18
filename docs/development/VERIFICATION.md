# Verification

This document binds the local and CI verification contract for the dependency
authority infrastructure core. It exists so every instance and every reviewer
can re-run the same gates.

## Local gates

Run from the repository root with the pinned Go toolchain (`go 1.26`,
`toolchain go1.26.6`) and the pinned OpenTofu engine (`1.12.5` on `PATH`):

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
(govulncheck), Lefthook configuration validation, Linux/AMD64 build of the
gate binaries with module provenance, and the OpenTofu gates — engine
version verification, recursive `tofu fmt -check`, and
`tofu init -backend=false` plus `tofu validate` for every module, the policy
bindings and every reference stack.

## Build tooling

Build tools live in the separate `tools/` module and are resolved through
its own verified `go.mod` and committed `go.sum`; they never join the source
module graph. The module pins `govulncheck`, `staticcheck`, and `lefthook`
via the Go tool directive and shares the repository toolchain pin.

## Toolchain and vulnerability re-scans

The Go toolchain is pinned exactly (`toolchain go1.26.6`,
`GOTOOLCHAIN=local`); CI asserts `go env GOVERSION` before any gate runs and
no lane may download a toolchain at build time. `govulncheck` is part of the
full source gate, and CI re-runs the full gate on a daily schedule so newly
disclosed vulnerabilities in the pinned toolchain or dependency graph fail
closed even without source changes.

The OpenTofu gates run with `OPENTOFU_ENFORCE_GPG_VALIDATION=true`,
`TF_IN_AUTOMATION=true` and `TF_INPUT=false`. Provider downloads land in the
repository-local plugin cache under `.build/`, which is never committed.
Module and policy-binding lock files stay local; every reference stack commits
its `.terraform.lock.hcl`.

## Test architecture

Every executable Go package carries same-package whitebox tests for its
invariants, branches, state transitions, errors, and cleanup paths.
`internal/packaging/` complements them with same-package workflow contract
tests that bind the CI surface, the organization rule-set adoption guard, the
module and stack layout, the exact OpenTofu pins and the core boundary (no
concrete organization, tenant, project, identity, network, secret or registry
bindings anywhere in the core).

## CI gates

The `Quality gates (linux-amd64)` check runs the full source-level gate with
the pinned Go and OpenTofu toolchains on every push and pull request to the
shared lines and once per day on a schedule. The `Dependency admission review`
check blocks unreviewed dependency changes. CodeQL code scanning runs with all
alerts blocking once the organization-level shared-line rule-sets are imported
and active; the binding of this repository is documented in
`docs/conventions/hosting-plattform/github/rule-sets/`.

Lefthook provides the local `commit-msg` hook (governed commit-message
validation) and the pre-push source-quality gate.

## Instance consumption

An organization instance consumes this core through exact module version pins
(for example a git source with an immutable `?ref=` version) and supplies the
concrete project IDs, regions, OIDC bindings, members and retention values as
reviewed instance configuration. An instance never copies the module source
into its own boundary; it references the pinned version and proves the binding
in its own evidence.
