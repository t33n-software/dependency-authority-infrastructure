# Module: state-home

The zone state home of the OpenTofu operating model: exactly one dedicated
Cloud Storage bucket per trust zone holding that zone's root states and
nothing else — uniform bucket-level access, enforced public access
prevention, object versioning and the mandatory bucket CMEK (the provider
layer of the dual fortress state-encryption standard), plus exactly the
`roles/storage.objectAdmin` data plane for the instance-bound operator
execution identities.

## Boundary

- Never carries organization, tenant, identity, network, secret or registry
  bindings; every concrete value is an instance-supplied variable.
- The CMEK key is mandatory and never defaults: the state home never exists
  without the provider layer, and the CMEK key is never the engine key of
  the client-side state encryption (the two keys are disjoint cryptographic
  boundaries).
- A state bucket never carries a retention policy: the state layer is the
  recovery root, not an archive; retention belongs to the evidence archive
  in `modules/evidence-archive`.
- The module grants exactly `roles/storage.objectAdmin` on the bucket — the
  documented backend credential requirement — and never constructs members;
  callers pass fully formed member strings.
- The bucket location is coupled to the CMEK key ring location (a hard
  platform rule); the consuming stack proves the coupling fail-closed.

## Usage

```hcl
module "state_home" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/state-home?ref=<exact-version>"

  project_id       = "<organization>-dep-control"
  name             = "<organization>-dep-control-state"
  location         = "<region>"
  cmek_key_name    = "projects/<project>/locations/<region>/keyRings/<keyring>/cryptoKeys/<state-bucket-cmek-key>"
  operator_members = ["user:<operator-identity>"]

  labels = {
    boundary = "dependency-authority"
    zone     = "control"
  }
}
```
