output "enforced_constraints" {
  description = "Organization-policy constraints enforced on the project."
  value       = sort(keys(google_org_policy_policy.this))
}

output "policy_ids" {
  description = "Resource IDs of the enforced organization policies, keyed by constraint."
  value       = { for constraint, policy in google_org_policy_policy.this : constraint => policy.id }
}
