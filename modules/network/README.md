# Module: network

The network surfaces of one trust zone: private DNS zones and their record
sets bound to explicit VPC network self links, and the zone workload network
origin — the VPC substrate every workload job of the zone attaches to.

## Boundary

- Never carries organization, tenant, identity, secret or registry bindings;
  zone names, DNS names, VPC self links, records and the workload network
  names, region and CIDR are instance-supplied values.
- The workload network origin surface creates exactly one VPC with one
  subnetwork in the job region carrying Private Google Access, the
  restricted-range DNS response policy (`*.googleapis.com` resolves to
  `restricted.googleapis.com`, the `199.36.153.4/30` range that serves only
  the VPC Service Controls restricted services, and the Artifact Registry
  data plane `*.pkg.dev` resolves to the same range — the registry domains
  are served by the restricted VIP, and without the mapping the workload's
  registry calls resolve to public addresses and fail against the deny-all
  egress rule) and the egress firewall pair (allow TCP 443 to the restricted
  range ordered before priority 1000, deny all egress ordered after priority
  1000; the allow rule targets the restricted range, not a domain, so it
  already covers the data plane). The restricted range and the DNS names are
  provider constants of the VPC Service Controls contract, never instance
  values.
- Every workload job of the zone attaches to this network and routes all
  outgoing traffic through it as Direct VPC egress with all-traffic routing
  (bound by the cloud-run-job module): a serverless job without the zone
  network attachment presents no in-perimeter network origin, its calls to
  the restricted planes are evaluated as external to the perimeter, and they
  fail closed — the zone-project membership of the workload does not by
  itself place its calls inside.
- The concrete private connectivity and egress design is an approved instance
  decision; this module only expresses it once it exists.

## Usage

```hcl
module "zone_network" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/network?ref=<exact-version>"

  project_id = "<organization>-dep-intake"

  workload_network = {
    network_name = "<zone-vpc>"
    subnet_name  = "<zone-subnet>"
    region       = "<job-region>"
    subnet_cidr  = "<subnet-cidr>"
  }

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

## Notes

- The module creates no project and enables no APIs; the organization
  instance provisions the project and its API surface first (including the
  compute and DNS APIs).
- The two surfaces are independent: a zone may bind private DNS zones, the
  workload network origin, or both.
