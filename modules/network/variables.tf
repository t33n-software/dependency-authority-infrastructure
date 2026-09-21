variable "project_id" {
  description = "Google Cloud project ID that owns the zone network surfaces. The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "zones" {
  description = <<-EOT
    Private DNS zones to create, keyed by the Cloud DNS zone resource name.
    Each zone binds its DNS name (must end with a dot), the VPC network self
    links that may resolve it, and its record sets keyed by record name.
    Defaults to none; the zone workload network origin is the separate
    workload_network surface.
  EOT
  type = map(object({
    dns_name    = string
    description = optional(string, "")
    networks    = set(string)
    records = optional(map(object({
      name    = string
      type    = string
      ttl     = number
      rrdatas = list(string)
    })), {})
  }))
  default = {}

  validation {
    condition     = alltrue([for _, zone in var.zones : endswith(zone.dns_name, ".")])
    error_message = "every zone dns_name must end with a dot."
  }

  validation {
    condition     = alltrue([for _, zone in var.zones : length(zone.networks) > 0])
    error_message = "every zone must bind at least one VPC network self link."
  }
}

variable "workload_network" {
  description = <<-EOT
    The zone workload network origin: exactly one VPC network with one
    subnetwork in the job region carrying Private Google Access, the
    restricted-range DNS response policy (every Google API call from the zone
    VPC resolves *.googleapis.com to restricted.googleapis.com, the
    199.36.153.4/30 range that serves only the restricted services, and the
    Artifact Registry data plane *.pkg.dev resolves to the same range) and the
    egress firewall pair (allow TCP 443 to the restricted range ordered before
    priority 1000, deny all egress ordered after priority 1000). Every
    workload job of the zone attaches to this network with Direct VPC egress
    and all-traffic routing; without it the job's calls to the restricted
    planes present no in-perimeter network origin and fail closed at the
    perimeter. Every managed surface of the origin binds its canonical
    human-readable description through the optional description fields (the
    mandatory description duty of the mandatory resource properties
    convention): the VPC and the subnetwork (create-only surfaces, bound
    byte-exact to the live values at convergence), the egress firewall pair
    and the restricted-range DNS response policy (in-place surfaces). All
    values are instance-supplied; the core never carries a default.
  EOT
  type = object({
    network_name               = string
    subnet_name                = string
    region                     = string
    subnet_cidr                = string
    network_description        = optional(string, null)
    subnet_description         = optional(string, null)
    firewall_allow_description = optional(string, null)
    firewall_deny_description  = optional(string, null)
    dns_policy_description     = optional(string, null)
  })
  default = null

  validation {
    condition = var.workload_network == null || (
      can(regex("^[a-z][a-z0-9-]*$", var.workload_network.network_name)) &&
      can(regex("^[a-z][a-z0-9-]*$", var.workload_network.subnet_name)) &&
      length(var.workload_network.region) > 0 &&
      can(cidrhost(var.workload_network.subnet_cidr, 0))
    )
    error_message = "workload_network must bind a valid network name, subnetwork name, region and subnet CIDR range."
  }

  # The mandatory description duty of the mandatory resource properties
  # convention: every managed surface of the workload network origin binds the
  # canonical human-readable description surface of its provider schema. A
  # bound description is never empty; an unbound surface binds null.
  validation {
    condition = var.workload_network == null || alltrue([
      for description in [
        var.workload_network.network_description,
        var.workload_network.subnet_description,
        var.workload_network.firewall_allow_description,
        var.workload_network.firewall_deny_description,
        var.workload_network.dns_policy_description,
      ] : description == null || length(description) > 0
    ])
    error_message = "every bound description surface of the workload network origin must be non-empty; an unbound surface binds null."
  }
}
