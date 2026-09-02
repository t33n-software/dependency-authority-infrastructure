locals {
  upstreams = {
    go = {
      common_uri = "https://proxy.golang.org"
    }
    npm = {
      npm = "NPMJS"
    }
    python = {
      python = "PYPI"
    }
  }

  # The canonical workload job topology of the intake zone: exactly one job
  # per lane operation, bound to the existing zone workload identity. The
  # image digest is an instance binding (planned until the promotion
  # read-back proof flips it to bound), never a stack default.
  workload_jobs = {
    "dep-intake-fetch" = {
      identity_key = "fetcher"
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

module "repositories" {
  source   = "../../modules/artifact-registry"
  for_each = var.ecosystems

  project_id      = var.project_id
  location        = var.location
  repository_id   = "${each.key}-dependencies-intake"
  description     = "Dependency authority ${each.key} intake: controlled remote upstream acquisition, never a consumer endpoint"
  format          = upper(each.key)
  mode            = "REMOTE_REPOSITORY"
  remote_upstream = local.upstreams[each.key]

  labels = {
    boundary  = "dependency-authority"
    zone      = "intake"
    ecosystem = each.key
  }
}

module "workload_identity" {
  source     = "../../modules/workload-identity"
  project_id = var.project_id
  pool_id    = var.pool_id

  identities = {
    fetcher = var.fetcher
  }
}

module "repository_iam" {
  source   = "../../modules/repository-iam"
  for_each = var.ecosystems

  project_id = var.project_id
  location   = var.location
  repository = module.repositories[each.key].id
  writers    = ["serviceAccount:${module.workload_identity.service_account_emails["fetcher"]}"]
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
    zone     = "intake"
  }
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
