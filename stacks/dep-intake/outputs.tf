output "repository_ids" {
  description = "Fully qualified intake repository resource IDs, keyed by ecosystem."
  value       = { for ecosystem, repository in module.repositories : ecosystem => repository.id }
}

output "registry_uris" {
  description = "Intake repository endpoint URIs, keyed by ecosystem."
  value       = { for ecosystem, repository in module.repositories : ecosystem => repository.registry_uri }
}

output "fetcher_service_account_email" {
  description = "Email of the intake fetcher service account (dep-intake-fetcher)."
  value       = module.workload_identity.service_account_emails["fetcher"]
}

output "workload_identity_pool_name" {
  description = "Fully qualified Workload Identity Pool resource name of the intake zone."
  value       = module.workload_identity.pool_name
}

output "audit_sink_writer_identity" {
  description = "Writer identity of the zone audit sink."
  value       = module.audit_log_sink.writer_identities[var.audit_sink_name]
}

output "enforced_constraints" {
  description = "Organization-policy constraints enforced on the intake project."
  value       = module.policy_bindings.enforced_constraints
}

output "workload_job_ids" {
  description = "Fully qualified Cloud Run job resource IDs of the intake zone, keyed by canonical job name."
  value       = { for name, job in module.workload_jobs : name => job.id }
}
