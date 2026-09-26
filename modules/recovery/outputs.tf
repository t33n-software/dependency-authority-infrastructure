output "service_account_email" {
  description = "Email of the break-glass recovery service account."
  value       = google_service_account.this.email
}

output "service_account_name" {
  description = "Resource name of the break-glass recovery service account."
  value       = google_service_account.this.name
}

output "entitlement_name" {
  description = "Resource name of the break-glass recovery entitlement — the activation, approval and read-back surface of the recovery drill."
  value       = google_privileged_access_manager_entitlement.this.name
}
