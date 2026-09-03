output "enforced_constraints" {
  description = "Organization-policy constraints enforced on the project."
  value = sort(concat(
    keys(google_org_policy_policy.this),
    [for policy in google_org_policy_policy.cloud_run_vpc_egress : "run.allowedVPCEgress"],
    [for policy in google_org_policy_policy.cloud_run_ingress : "run.allowedIngress"],
  ))
}

output "policy_ids" {
  description = "Resource IDs of the enforced organization policies, keyed by constraint."
  value = merge(
    { for constraint, policy in google_org_policy_policy.this : constraint => policy.id },
    { for policy in google_org_policy_policy.cloud_run_vpc_egress : "run.allowedVPCEgress" => policy.id },
    { for policy in google_org_policy_policy.cloud_run_ingress : "run.allowedIngress" => policy.id },
  )
}
