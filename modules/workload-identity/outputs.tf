output "pool_name" {
  description = "Fully qualified Workload Identity Pool resource name (projects/<number>/locations/global/workloadIdentityPools/<pool>)."
  value       = google_iam_workload_identity_pool.this.name
}

output "provider_names" {
  description = "Fully qualified provider resource names, keyed by identity key."
  value       = { for key, provider in google_iam_workload_identity_pool_provider.this : key => provider.name }
}

output "service_account_emails" {
  description = "Service account emails, keyed by identity key."
  value       = { for key, account in google_service_account.this : key => account.email }
}

output "service_account_names" {
  description = "Service account resource names, keyed by identity key."
  value       = { for key, account in google_service_account.this : key => account.name }
}

output "principal_sets" {
  description = "Principal set strings bound to roles/iam.workloadIdentityUser, keyed by identity key."
  value       = { for key, binding in google_service_account_iam_member.workload_identity_user : key => binding.member }
}

output "trigger_service_account_emails" {
  description = "Invoke-only trigger service account emails (the lane-facing identities), keyed by identity key."
  value       = { for key, account in google_service_account.trigger : key => account.email }
}
