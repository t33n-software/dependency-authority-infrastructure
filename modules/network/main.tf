resource "google_dns_managed_zone" "this" {
  for_each = var.zones

  project     = var.project_id
  name        = each.key
  dns_name    = each.value.dns_name
  description = each.value.description
  visibility  = "private"

  private_visibility_config {
    dynamic "networks" {
      for_each = each.value.networks
      content {
        network_url = networks.value
      }
    }
  }
}

locals {
  records = length(var.zones) == 0 ? {} : merge([
    for zone_key, zone in var.zones : {
      for record_key, record in zone.records : "${zone_key}/${record_key}" => merge(record, { zone = zone_key })
    }
  ]...)
}

resource "google_dns_record_set" "this" {
  for_each = local.records

  project      = var.project_id
  managed_zone = google_dns_managed_zone.this[each.value.zone].name
  name         = each.value.name
  type         = each.value.type
  ttl          = each.value.ttl
  rrdatas      = each.value.rrdatas
}

# The zone workload network origin: exactly one VPC with one subnetwork in the
# job region carrying Private Google Access. Every workload job of the zone
# attaches to this network with Direct VPC egress and routes all outgoing
# traffic through it, so its calls to the restricted planes originate inside
# the perimeter; a workload without the attachment is evaluated as external to
# the perimeter and fails closed.
resource "google_compute_network" "workload" {
  count = var.workload_network != null ? 1 : 0

  project                 = var.project_id
  name                    = var.workload_network.network_name
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "workload" {
  count = var.workload_network != null ? 1 : 0

  project                  = var.project_id
  name                     = var.workload_network.subnet_name
  region                   = var.workload_network.region
  network                  = google_compute_network.workload[0].id
  ip_cidr_range            = var.workload_network.subnet_cidr
  private_ip_google_access = true
}

# The restricted-range DNS form: every Google API call from the zone VPC
# resolves to restricted.googleapis.com (199.36.153.4/30), the virtual IP range
# that serves only the VPC Service Controls restricted services.
resource "google_dns_response_policy" "workload" {
  count = var.workload_network != null ? 1 : 0

  project              = var.project_id
  response_policy_name = "${var.workload_network.network_name}-restricted-googleapis"

  networks {
    network_url = google_compute_network.workload[0].id
  }
}

resource "google_dns_response_policy_rule" "restricted_googleapis" {
  count = var.workload_network != null ? 1 : 0

  project         = var.project_id
  response_policy = google_dns_response_policy.workload[0].response_policy_name
  rule_name       = "restricted-googleapis"
  dns_name        = "*.googleapis.com."

  local_data {
    local_datas {
      name    = "*.googleapis.com."
      type    = "A"
      ttl     = 300
      rrdatas = ["199.36.153.4", "199.36.153.5", "199.36.153.6", "199.36.153.7"]
    }
  }
}

# The egress firewall pair: nothing leaves the zone VPC except TCP 443 toward
# the restricted range. The allow rule orders before priority 1000, the
# deny-all rule after it.
resource "google_compute_firewall" "allow_restricted_googleapis_egress" {
  count = var.workload_network != null ? 1 : 0

  project   = var.project_id
  name      = "${var.workload_network.network_name}-allow-restricted-googleapis"
  network   = google_compute_network.workload[0].id
  direction = "EGRESS"
  priority  = 999

  destination_ranges = ["199.36.153.4/30"]

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }
}

resource "google_compute_firewall" "deny_all_egress" {
  count = var.workload_network != null ? 1 : 0

  project   = var.project_id
  name      = "${var.workload_network.network_name}-deny-all-egress"
  network   = google_compute_network.workload[0].id
  direction = "EGRESS"
  priority  = 1001

  destination_ranges = ["0.0.0.0/0"]

  deny {
    protocol = "all"
  }
}
