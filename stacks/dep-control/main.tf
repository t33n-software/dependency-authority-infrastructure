locals {
  workload_image_registries = {
    staging = {
      description = "Dependency authority workload images staging: the only delivery target of the governed producer channel, never a workload source"
    }
    release = {
      description = "Dependency authority workload images release: the only workload consumption source, filled exclusively through promotion of a proven staging digest"
    }
  }
}

provider "google" {
  project = var.project_id
}

module "policy_bindings" {
  source     = "../../policy-bindings"
  project_id = var.project_id

  disable_service_account_key_creation = var.policy_constraints.disable_service_account_key_creation
  disable_service_account_key_upload   = var.policy_constraints.disable_service_account_key_upload
  uniform_bucket_level_access          = var.policy_constraints.uniform_bucket_level_access
  public_access_prevention             = var.policy_constraints.public_access_prevention
}

module "workload_image_registries" {
  source   = "../../modules/artifact-registry"
  for_each = local.workload_image_registries

  project_id    = var.project_id
  location      = var.location
  repository_id = "${each.key}-controller-images"
  description   = each.value.description
  format        = "DOCKER"
  mode          = "STANDARD_REPOSITORY"

  labels = {
    boundary = "dependency-authority"
    zone     = "control"
  }
}

module "workload_identity" {
  source     = "../../modules/workload-identity"
  project_id = var.project_id
  pool_id    = var.pool_id

  identities = var.controllers
}

module "audit_log_sink" {
  source             = "../../modules/logging"
  project_id         = var.project_id
  destination_bucket = var.evidence_bucket_name

  sinks = {
    (var.audit_sink_name) = {
      filter = var.audit_log_filter
    }
  }
}
