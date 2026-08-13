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
