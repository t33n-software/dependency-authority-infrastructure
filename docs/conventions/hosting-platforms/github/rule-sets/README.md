# Hosting Platform: GitHub — Rule Sets
[INTENT: REFERENCE]

## Canonical source

The GitHub rule sets for the organization `t33n-software` are defined and
managed exactly once, centrally, in the repository
[`git-governance`](https://github.com/t33n-software/git-governance) under
`rulesets/github/`. That repository is the canonical source of truth for the
JSON definitions: it explains the architecture, applies the definitions, and
ships the versioned, importable artifacts.

A local copy, redefinition, or deviation in this repository is an anti-pattern
and forbidden (redundancy and drift prohibition). The only permitted
deviations are named, auditable repository exceptions that are stricter than
the organization baseline, never weaker.

## Family in use

This project (`dependency-authority-infrastructure`) uses the family **`quality-gates=linux-only`**:

- The quality gates run exclusively on **Linux**.
- Architectural rationale: this project ships organization-agnostic OpenTofu
  modules and reference stacks plus pure Linux/CI/CD artifacts; the gate
  binaries of the Go toolchain are built and verified exclusively for
  Linux/AMD64. No operating-system-specific binaries exist, so Windows or
  macOS gates would add no control value.

## Bound rule sets

| Rule set | Class |
|---|---|
| `push-protections: secret artifact boundary` | classless (private/internal visibility) |
| `branch-governance: ticket working branches` | classless (`~ALL`) |
| `branch-governance: develop shared line (quality-gates=linux-only)` | linux-only |
| `branch-governance: main shared line (quality-gates=linux-only)` | linux-only |
| `branch-governance: release shared lines (quality-gates=linux-only)` | linux-only |
| `branch-governance: support shared lines (quality-gates=linux-only)` | linux-only |

## Management

- Management level: the **organization** (`t33n-software`), never the
  individual repository level.
- Class membership of this repository: custom property
  `quality-gates=linux-only`.
- Changes to the rule sets happen exclusively in the canonical repository and
  are then re-imported at organization level (Organization Settings →
  Repository → Rulesets).
