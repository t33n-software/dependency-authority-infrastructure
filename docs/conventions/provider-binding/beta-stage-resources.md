# Convention: Beta-stage provider resources in the declaration plane

This convention binds how the infrastructure core declares provider resources
that the pinned generally available (GA) provider does not carry. It exists
because infrastructure declarations are code: a declaration that lives only in
prose is not verifiable, not guard-bound and not reproducible, and is
prohibited.

## The law

1. Every infrastructure form of the target architecture is declared as code in
   the infrastructure core. A markdown document, a runbook or an operator
   note is never the declaration of an infrastructure form; documentation
   accompanies the declaration, it never substitutes for it.
2. The engine is OpenTofu, exactly pinned, with provider GPG validation
   enforced everywhere and committed stack lockfiles. The provider supply
   chain carries the exactly pinned GA provider (`hashicorp/google`) and,
   only under this convention, the exactly pinned beta provider
   (`hashicorp/google-beta`) of the same publisher and signing trust anchor.
3. The beta provider may declare a resource only when the pinned GA provider
   provably lacks that resource. The proof is produced against the pinned
   provider's own schema surface (`tofu providers schema -json` in an
   initialized root), never from memory, and is recorded in the architecture
   decision record of the change that introduces the resource.
4. The beta provider scope is fail-closed: a packaging contract guard proves
   that the beta provider declares only the resources proven GA-absent. Any
   other use of the beta provider fails the governed suite.
5. A beta-stage resource is declared with the same discipline as every GA
   resource: null-gated opt-in surfaces, fail-closed validations, zone purity
   and instance-bound values. The beta marking never relaxes a boundary.
6. The promotion path is declared with the introduction: when the GA provider
   carries the resource, a governed change flips the provider reference to
   the GA provider, and the beta provider is removed from the core once its
   last resource is gone. Provider version movements of either provider are
   separate governed changes under the exact-pinning law.

## Current application

The Artifact Registry VPC Service Controls configuration
(`google_artifact_registry_vpcsc_config`) is the single proven case: the
pinned GA provider schema carries no resource for it while the underlying API
is generally available. The artifact-registry module declares it as the
null-gated `vpcsc_upstream_allowance` opt-in surface (default fail-closed
denied, remote mode required), and only a zone with a remote repository
inside the perimeter opts in — canonically the intake zone. The allowance is
the zone-level registry-platform singleton for the project and location,
covers only the configured upstreams of the zone's remote repositories and is
never a perimeter egress rule.
