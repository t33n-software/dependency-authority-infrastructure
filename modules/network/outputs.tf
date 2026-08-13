output "zone_ids" {
  description = "Managed zone resource IDs, keyed by zone resource name."
  value       = { for key, zone in google_dns_managed_zone.this : key => zone.id }
}

output "record_set_ids" {
  description = "Record set resource IDs, keyed by zone and record key."
  value       = { for key, record in google_dns_record_set.this : key => record.id }
}
