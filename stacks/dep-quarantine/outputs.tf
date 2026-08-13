output "repository_ids" {
  description = "Fully qualified quarantine repository resource IDs, keyed by ecosystem."
  value       = { for ecosystem, repository in module.repositories : ecosystem => repository.id }
}

output "workload_identity_pool_name" {
  description = "Fully qualified Workload Identity Pool resource name of the quarantine zone."
  value       = module.workload_identity.pool_name
}

output "service_account_emails" {
  description = "Service account emails of the quarantine zone identities, keyed by identity key."
  value       = module.workload_identity.service_account_emails
}

output "audit_sink_writer_identity" {
  description = "Writer identity of the zone audit sink."
  value       = module.audit_log_sink.writer_identities[var.audit_sink_name]
}

output "enforced_constraints" {
  description = "Organization-policy constraints enforced on the quarantine project."
  value       = module.policy_bindings.enforced_constraints
}
