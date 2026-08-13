output "repository_ids" {
  description = "Fully qualified approved repository resource IDs, keyed by ecosystem."
  value       = { for ecosystem, repository in module.repositories : ecosystem => repository.id }
}

output "registry_uris" {
  description = "Approved repository endpoint URIs, keyed by ecosystem. These are the only dependency consumer endpoints."
  value       = { for ecosystem, repository in module.repositories : ecosystem => repository.registry_uri }
}

output "promoter_service_account_email" {
  description = "Email of the approved promoter service account (dep-approved-promoter)."
  value       = module.workload_identity.service_account_emails["promoter"]
}

output "workload_identity_pool_name" {
  description = "Fully qualified Workload Identity Pool resource name of the approved zone."
  value       = module.workload_identity.pool_name
}

output "audit_sink_writer_identity" {
  description = "Writer identity of the zone audit sink."
  value       = module.audit_log_sink.writer_identities[var.audit_sink_name]
}

output "enforced_constraints" {
  description = "Organization-policy constraints enforced on the approved project."
  value       = module.policy_bindings.enforced_constraints
}
