output "zone_ids" {
  description = "Managed zone resource IDs, keyed by zone resource name."
  value       = { for key, zone in google_dns_managed_zone.this : key => zone.id }
}

output "record_set_ids" {
  description = "Record set resource IDs, keyed by zone and record key."
  value       = { for key, record in google_dns_record_set.this : key => record.id }
}

output "workload_network_id" {
  description = "Resource ID of the zone workload VPC (projects/<project>/global/networks/<network>), bound by the zone's workload jobs; null when the zone declares no workload network."
  value       = var.workload_network != null ? google_compute_network.workload[0].id : null
}

output "workload_subnetwork_id" {
  description = "Resource ID of the zone workload subnetwork (projects/<project>/regions/<region>/subnetworks/<subnetwork>), bound by the zone's workload jobs; null when the zone declares no workload network."
  value       = var.workload_network != null ? google_compute_subnetwork.workload[0].id : null
}
