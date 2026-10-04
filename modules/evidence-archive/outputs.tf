output "name" {
  description = "Bucket name of the retention archive."
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

output "logging_export_writer_members" {
  description = "The members carrying the standing append-focused write capability on the retention archive: the Cloud Logging service agents of the organization-plane audit export anchors, as granted by this module. The consuming window cross-checks the live bucket IAM against this set."
  value       = keys(google_storage_bucket_iam_member.logging_export_writer)
}
