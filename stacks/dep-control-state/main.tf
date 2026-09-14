provider "google" {
  project = var.project_id
}

# The minimal state-home root of the control zone: the dedicated state
# bucket, its operator data-plane IAM and the encryption binding — declared
# once, applied with a locally encrypted state at birth and migrated into its
# own bucket immediately after the bucket exists.
module "state_home" {
  source = "../../modules/state-home"

  project_id       = var.project_id
  name             = var.state_bucket_name
  location         = var.location
  cmek_key_name    = var.state_bucket_cmek_key
  operator_members = var.operator_members

  labels = {
    boundary = "dependency-authority"
    zone     = "control"
  }
}
