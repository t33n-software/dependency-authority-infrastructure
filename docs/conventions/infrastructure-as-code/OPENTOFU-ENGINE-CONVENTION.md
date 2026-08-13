# OpenTofu as the Infrastructure-as-Code Engine

## Decision

All infrastructure as code in this organization is written in HCL and executed
with OpenTofu. Terraform is not the standard and is not used.

## Why

- **License sovereignty.** OpenTofu is MPL 2.0 under the Linux Foundation and
  the CNCF with multi-vendor governance. Terraform >= 1.6 is BUSL 1.1 under a
  single licensor that already changed the license unilaterally once
  (2023-08-10). The infrastructure engine is the root of trust of every
  infrastructure change; that position requires neutral, OSI-approved
  governance.
- **State protection.** OpenTofu provides client-side state and plan encryption
  natively; Terraform offers no client-side equivalent.
- **Verified toolchain.** OpenTofu releases are verifiable twice over (GPG and
  Sigstore/cosign keyless, bound to the release workflow through OIDC) and
  ship no telemetry.

## Conventions

- Exact pins only: `required_version` and every provider `version` are exact;
  each root module commits its `.terraform.lock.hcl`.
- Provider GPG validation is enforced everywhere through
  `OPENTOFU_ENFORCE_GPG_VALIDATION=true`.
- Secrets never persist in state or plan; ephemeral and write-only mechanisms
  are mandatory for sensitive inputs.
- A deviation from OpenTofu requires an explicit documented exception.

This convention file is intentionally identical across the organization's
infrastructure cores (`dependency-authority-infrastructure` and
`developer-platform-infrastructure`): the rationale is organization-wide, not
project-specific.
