output "controller_service_account_emails" {
  description = "Service account emails of the control-plane controllers, keyed by lane."
  value       = module.workload_identity.service_account_emails
}

output "workload_identity_pool_name" {
  description = "Fully qualified Workload Identity Pool resource name of the control zone."
  value       = module.workload_identity.pool_name
}

output "audit_sink_writer_identity" {
  description = "Writer identity of the zone audit sink."
  value       = module.audit_log_sink.writer_identities[var.audit_sink_name]
}

output "enforced_constraints" {
  description = "Organization-policy constraints enforced on the control project."
  value       = module.policy_bindings.enforced_constraints
}
