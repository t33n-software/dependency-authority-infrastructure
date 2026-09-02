output "repository_ids" {
  description = "Fully qualified evidence repository resource IDs, keyed by ecosystem."
  value       = { for ecosystem, repository in module.repositories : ecosystem => repository.id }
}

output "archive_bucket_name" {
  description = "Name of the retention archive bucket. Other trust zones receive this value to wire their audit exports."
  value       = module.evidence_archive.name
}

output "writer_service_account_email" {
  description = "Email of the evidence writer service account (dep-evidence-writer)."
  value       = module.workload_identity.service_account_emails["writer"]
}

output "auditor_service_account_email" {
  description = "Email of the evidence auditor service account (dep-evidence-auditor)."
  value       = module.workload_identity.service_account_emails["auditor"]
}

output "workload_trigger_service_account_emails" {
  description = "Invoke-only trigger service account emails of the evidence lanes, keyed by identity (writer, auditor)."
  value       = module.workload_identity.trigger_service_account_emails
}

output "workload_identity_pool_name" {
  description = "Fully qualified Workload Identity Pool resource name of the evidence zone."
  value       = module.workload_identity.pool_name
}

output "audit_sink_writer_identity" {
  description = "Writer identity of the zone audit sink."
  value       = module.audit_log_sink.writer_identities[var.audit_sink_name]
}

output "enforced_constraints" {
  description = "Organization-policy constraints enforced on the evidence project."
  value       = module.policy_bindings.enforced_constraints
}

output "workload_job_ids" {
  description = "Fully qualified Cloud Run job resource IDs of the evidence zone, keyed by canonical job name."
  value       = { for name, job in module.workload_jobs : name => job.id }
}
