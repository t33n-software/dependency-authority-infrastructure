# Verification

This document binds the local and CI verification contract for the dependency
authority infrastructure core. It exists so every instance and every reviewer
can re-run the same gates.

## Local gates

Run from the repository root with the pinned Go toolchain (`go 1.26`,
`toolchain go1.26.5`) and the pinned OpenTofu engine (`1.12.5` on `PATH`):

```text
gofmt
go test ./...
go run -mod=readonly ./cmd/check-coverage
go run -mod=readonly ./cmd/build
```

`cmd/build` is the full source-level gate: Go formatting, module checksums,
module metadata, unit tests, exact 100% statement coverage, race detector,
static analysis, Linux/AMD64 build of the gate binaries with module
provenance, and the OpenTofu gates — engine version verification, recursive
`tofu fmt -check`, and `tofu init -backend=false` plus `tofu validate` for
every module, the policy bindings and every reference stack.

The OpenTofu gates run with `OPENTOFU_ENFORCE_GPG_VALIDATION=true`,
`TF_IN_AUTOMATION=true` and `TF_INPUT=false`. Provider downloads land in the
repository-local plugin cache under `.build/`, which is never committed.
Module and policy-binding lock files stay local; every reference stack commits
its `.terraform.lock.hcl`.

## Test architecture

Every executable Go package carries same-package whitebox tests for its
invariants, branches, state transitions, errors, and cleanup paths.
`internal/packaging/` complements them with same-package workflow contract
tests that bind the CI surface, the Rulesets, the module and stack layout, the
exact OpenTofu pins and the core boundary (no concrete organization, tenant,
project, identity, network, secret or registry bindings anywhere in the
core).

## CI gates

The `Quality gates (linux-amd64)` check runs the full source-level gate with
the pinned Go and OpenTofu toolchains. The `Dependency admission review` check
blocks unreviewed dependency changes. CodeQL code scanning runs with all
alerts blocking once the shared-line Rulesets are imported.

## Instance consumption

An organization instance consumes this core through exact module version pins
(for example a git source with an immutable `?ref=` version) and supplies the
concrete project IDs, regions, OIDC bindings, members and retention values as
reviewed instance configuration. An instance never copies the module source
into its own boundary; it references the pinned version and proves the binding
in its own evidence.
