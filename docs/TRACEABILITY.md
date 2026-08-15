# Traceability

## Tickets

| Ticket | Change | Status |
|---|---|---|
| DAI-1 | Establish the dependency authority infrastructure core: seven generic modules (artifact-registry, evidence-archive, logging, network, repository-iam, recovery, workload-identity), project-level policy bindings, five parameterized trust-zone reference stacks (dep-control, dep-intake, dep-quarantine, dep-approved, dep-evidence), the OpenTofu engine convention, source-quality gates, CodeQL, dependency admission review, Dependabot, Lefthook, and importable Rulesets. | In progress |
| DAI-2 | Migrate the module path to the `t33n-software` organization namespace; add the LF line-ending contract (`.gitattributes`) and the push-protections Ruleset source `00-push-protections.json` in the verified GitHub export format. | In progress |

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
