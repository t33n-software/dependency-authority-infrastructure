output "iam_member_ids" {
  description = "Resource IDs of the created repository IAM bindings, keyed by role and member."
  value       = { for key, binding in google_artifact_registry_repository_iam_member.this : key => binding.id }
}

output "repository" {
  description = "Repository reference the bindings apply to."
  value       = var.repository
}
