output "state_bucket_name" {
  description = "Bucket name of the control-zone state home."
  value       = module.state_home.name
}

output "state_bucket_id" {
  description = "Bucket resource ID of the control-zone state home."
  value       = module.state_home.id
}

output "state_bucket_url" {
  description = "Bucket URL (gs:// form) of the control-zone state home."
  value       = module.state_home.url
}

output "state_bucket_operator_iam_member_ids" {
  description = "Resource IDs of the operator object-admin bindings on the state bucket, keyed by member."
  value       = module.state_home.operator_iam_member_ids
}
