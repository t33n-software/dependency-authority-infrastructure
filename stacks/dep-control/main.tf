locals {
  workload_image_registries = {
    staging = {
      description = "Dependency authority workload images staging: the only delivery target of the governed producer channel, never a workload source"
    }
    release = {
      description = "Dependency authority workload images release: the only workload consumption source, filled exclusively through promotion of a proven staging digest"
    }
  }

  # The canonical workload job topology of the control zone: exactly one job
  # per lane operation, each bound to the existing zone workload identity of
  # its lane. The image digests are instance bindings (planned until the
  # promotion read-back proof flips them to bound), never stack defaults.
  workload_jobs = {
    "dep-admission" = {
      identity_key = "admission"
    }
    "dep-promotion" = {
      identity_key = "promotion"
    }
    "dep-revalidation" = {
      identity_key = "revalidation"
    }
    "dep-revocation" = {
      identity_key = "revocation"
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

module "workload_jobs" {
  source   = "../../modules/cloud-run-job"
  for_each = local.workload_jobs

  project_id            = var.project_id
  location              = var.location
  name                  = each.key
  service_account_email = module.workload_identity.service_account_emails[each.value.identity_key]
  image                 = var.workload_job_images[each.key]

  labels = {
    boundary = "dependency-authority"
    zone     = "control"
  }
}

# The canonical IAM target matrix of the workload image registries: every
# zone lane identity reads the release class (the only workload consumption
# source); no identity ever receives a writer grant on either class, and the
# staging class carries no binding at all — it is filled exclusively by the
# governed producer channel and is never a workload source.
module "workload_image_registry_iam" {
  source     = "../../modules/repository-iam"
  project_id = var.project_id
  location   = var.location
  repository = module.workload_image_registries["release"].id

  readers = concat(
    [for lane in keys(var.controllers) : "serviceAccount:${module.workload_identity.service_account_emails[lane]}"],
    tolist(var.cross_zone_workload_reader_members),
  )
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
