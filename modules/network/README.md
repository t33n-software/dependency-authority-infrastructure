# Module: network

Private DNS zones and their record sets for controlled connectivity inside a
trust zone: private visibility bound to explicit VPC network self links.

## Boundary

- Never carries organization, tenant, identity, secret or registry bindings;
  zone names, DNS names, VPC self links and records are instance-supplied
  values.
- The module creates no VPCs, subnets, firewall rules or connectivity; those
  belong to the organization platform foundation.
- The concrete private connectivity and egress design is an approved instance
  decision; this module only expresses it once it exists.

## Usage

```hcl
module "zone_dns" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/network?ref=<exact-version>"

  project_id = "<organization>-dep-approved"

  zones = {
    "pkg-dev-private" = {
      dns_name = "pkg.dev."
      networks = ["<vpc-network-self-link>"]
      records = {
        "pkg.dev." = {
          name    = "pkg.dev."
          type    = "A"
          ttl     = 300
          rrdatas = ["<private-endpoint-ip>"]
        }
      }
    }
  }
}
```
