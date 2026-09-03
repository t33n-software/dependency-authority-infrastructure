output "controller_service_account_emails" {
  description = "Service account emails of the control-plane controllers, keyed by lane."
  value       = module.workload_identity.service_account_emails
}

output "controller_trigger_service_account_emails" {
  description = "Invoke-only trigger service account emails of the control-plane lanes, keyed by lane."
  value       = module.workload_identity.trigger_service_account_emails
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

output "workload_image_repository_ids" {
  description = "Fully qualified workload image repository resource IDs, keyed by class (staging, release)."
  value       = { for class, repository in module.workload_image_registries : class => repository.id }
}

output "workload_image_registry_uris" {
  description = "Workload image repository endpoint URIs, keyed by class (staging, release)."
  value       = { for class, repository in module.workload_image_registries : class => repository.registry_uri }
}

output "workload_job_ids" {
  description = "Fully qualified Cloud Run job resource IDs of the control zone, keyed by canonical job name."
  value       = { for name, job in module.workload_jobs : name => job.id }
}

output "workload_network_id" {
  description = "Resource ID of the zone workload VPC, bound by the zone's workload jobs."
  value       = module.network.workload_network_id
}

output "workload_subnetwork_id" {
  description = "Resource ID of the zone workload subnetwork, bound by the zone's workload jobs."
  value       = module.network.workload_subnetwork_id
}
