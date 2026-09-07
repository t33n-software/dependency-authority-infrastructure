output "log_reader_binding_id" {
  description = "Resource ID of the forensics log-reader binding on the zone project."
  value       = google_project_iam_member.log_reader.id
}

output "execution_reader_binding_id" {
  description = "Resource ID of the forensics execution-reader binding on the zone project."
  value       = google_project_iam_member.execution_reader.id
}

output "ingress_policy_id" {
  description = "Resource ID of the forensics perimeter ingress rule; null where the rule is not declared (only the control-zone stack declares it)."
  value       = one(google_access_context_manager_service_perimeter_ingress_policy.forensics[*].id)
}
