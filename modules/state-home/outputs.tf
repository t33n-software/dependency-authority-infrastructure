output "name" {
  description = "Bucket name of the zone state home."
  value       = google_storage_bucket.this.name
}

output "id" {
  description = "Bucket resource ID."
  value       = google_storage_bucket.this.id
}

output "self_link" {
  description = "Bucket self link."
  value       = google_storage_bucket.this.self_link
}

output "url" {
  description = "Bucket URL (gs:// form)."
  value       = google_storage_bucket.this.url
}

output "operator_iam_member_ids" {
  description = "Resource IDs of the operator object-admin bindings on the state bucket, keyed by member."
  value       = { for member, binding in google_storage_bucket_iam_member.operators : member => binding.id }
}
