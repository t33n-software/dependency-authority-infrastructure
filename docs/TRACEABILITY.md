# Traceability

## Tickets

| Ticket | Change | Status |
|---|---|---|
| DAI-1 | Establish the dependency authority infrastructure core: seven generic modules (artifact-registry, evidence-archive, logging, network, repository-iam, recovery, workload-identity), project-level policy bindings, five parameterized trust-zone reference stacks (dep-control, dep-intake, dep-quarantine, dep-approved, dep-evidence), the OpenTofu engine convention, source-quality gates, CodeQL, dependency admission review, Dependabot, Lefthook, and importable Rulesets. | In progress |
| DAI-2 | Migrate the module path to the `t33n-software` organization namespace; add the LF line-ending contract (`.gitattributes`) and the push-protections Ruleset source `00-push-protections.json` in the verified GitHub export format. | In progress |
| DAI-3 | Align the Go 1.26.6 toolchain and source gates with the supply chain fortress contract: pinned `tools/` module with govulncheck, staticcheck, and Lefthook; fail-closed vulnerability analysis; Lefthook configuration validation and commit-msg hook; daily CI re-scan. | In progress |
| DAI-5 | Adopt the canonical repo surface with the `opentofu@1` capability pack: the byte-identical canonical callers (`ci.yml`, `codeql.yml`, `dependency-review.yml`) pinned to the repository-governance home at the provision-seam stand; the thin `canonical-conformance.yml` lane running the home verifier; the canonical file family and the materialized `.github/CODEOWNERS`; the `repo-bindings.json` tenant manifest; the schema-v4 quality configuration declaring `extends: ["opentofu@1"]`; the tooling-module pins for `quality-gate` and `check-coverage` (the provision-capable go-quality-authority stand), `verify-canonical` (the extends-capable repository-governance stand), and the shared-kernel module requirement for the pack registry resolution; the conventional `--version` surface of the development tools; the OpenTofu gates moved into the pack with the repository-local setup steps and the duplicated engine convention document removed; and the packaging contract bound to the manifest. | In progress |
| DAI-6 | Reference the canonical gate chain through the tooling module pin: the repo-local chain copies `cmd/build` and `cmd/check-coverage` are removed (the `cmd/` tree carries no repository tooling anymore), the quality configuration invokes the go-quality-authority orchestrator via `go tool -modfile tools/go.mod quality-gate` with the `opentofu@1` pack declaration unchanged, the restated `defaults` block is dropped for the schema-owned `includeFamilies` default (SCG-9), and the contract guards prove the canonical invocation, the absent `defaults` block, and the absence of both copies fail-closed. | In progress |

## Scope boundaries

- DAI-1 delivers the organization-agnostic source core only. It does not
  create Google Cloud projects, enable APIs, or provision live trust-zone
  resources; the organization instance provisions those through the governed
  infrastructure change.
- Revocation download rules are runtime operations of the revocation
  controller and are intentionally not part of the static stacks.
- The core contains no concrete organization or tenant bindings, no live
  state or plans, and no variable binding files.
- The `release/*` and `support/*` branch families and their Rulesets are
  activated only with a complete governed release lifecycle.
