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
