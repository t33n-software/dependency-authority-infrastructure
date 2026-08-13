output "service_account_email" {
  description = "Email of the break-glass recovery service account."
  value       = google_service_account.this.email
}

output "service_account_name" {
  description = "Resource name of the break-glass recovery service account."
  value       = google_service_account.this.name
}

output "condition_end_time" {
  description = "RFC 3339 timestamp after which the break-glass grant stops applying."
  value       = var.condition_end_time
}
