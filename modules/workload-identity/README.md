# Module: workload-identity

The non-static workload identity foundation of one trust zone: exactly one
Workload Identity Pool plus, per lane identity, one OIDC provider, one service
account, one principal-set binding and optional project roles.

## Boundary

- One pool per trust zone; pools, providers and service accounts are never
  shared across intake, quarantine, approved, evidence or control.
- The default attribute mapping covers the GitHub OIDC claims (subject,
  actor, audience, repository, workflow reference, environment, ref); a custom
  mapping must keep `google.subject`.
- Every identity requires an instance-bound `attribute_condition` and
  `principal_value`: repository, protected workflow reference, environment and
  audience bindings are reviewed instance configuration, never core defaults.
- No service-account keys are created anywhere; identities are short-lived
  OIDC exchanges only.

## Usage

```hcl
module "zone_identity" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/workload-identity?ref=<exact-version>"

  project_id = "<organization>-dep-intake"
  pool_id    = "dep-intake"

  identities = {
    fetcher = {
      provider_id         = "dep-intake-fetcher"
      service_account_id  = "dep-intake-fetcher"
      attribute_condition = "<cel-binding: repository, workflow ref, environment>"
      principal_value     = "<repository-or-bound-attribute-value>"
    }
  }
}
```
