locals {
  # The canonical workload job topology of the evidence zone: exactly one job
  # per lane operation, bound to the existing zone workload identity and
  # invoked through its dedicated invoke-only trigger identity. The image
  # digest is an instance binding (planned until the promotion read-back
  # proof flips it to bound), never a stack default.
  workload_jobs = {
    "dep-evidence-write" = {
      identity_key = "writer"
      trigger_id   = "dep-evidence-write-trigger"
    }
    "dep-evidence-audit" = {
      identity_key = "auditor"
      trigger_id   = "dep-evidence-audit-trigger"
    }
  }
}

provider "google" {
  project = var.project_id
}

module "policy_bindings" {
  source     = "../../policy-bindings"
  project_id = var.project_id

  disable_service_account_key_creation  = var.policy_constraints.disable_service_account_key_creation
  disable_service_account_key_upload    = var.policy_constraints.disable_service_account_key_upload
  uniform_bucket_level_access           = var.policy_constraints.uniform_bucket_level_access
  public_access_prevention              = var.policy_constraints.public_access_prevention
  cloud_run_vpc_egress_all_traffic_only = var.policy_constraints.cloud_run_vpc_egress_all_traffic_only
  cloud_run_ingress_internal_only       = var.policy_constraints.cloud_run_ingress_internal_only
}

# The zone workload network origin: exactly one VPC with one subnetwork in the
# job region carrying Private Google Access, the restricted-range DNS response
# policy and the egress firewall pair. The zone's workload jobs attach to it
# with Direct VPC egress and all-traffic routing, so their calls to the
# restricted planes originate inside the perimeter.
module "network" {
  source     = "../../modules/network"
  project_id = var.project_id

  workload_network = merge(var.workload_network, {
    region = var.location
  })
}

module "repositories" {
  source   = "../../modules/artifact-registry"
  for_each = var.ecosystems

  project_id    = var.project_id
  location      = var.location
  repository_id = "${each.key}-dependencies-evidence"
  description   = "Dependency authority ${each.key} evidence: immutable operational evidence, append-only"
  format        = "GENERIC"
  mode          = "STANDARD_REPOSITORY"

  labels = {
    boundary  = "dependency-authority"
    zone      = "evidence"
    ecosystem = each.key
  }
}

module "evidence_archive" {
  source = "../../modules/evidence-archive"

  project_id               = var.project_id
  name                     = var.archive_bucket_name
  location                 = var.location
  retention_period_seconds = var.retention_period_seconds
  lock_retention_policy    = var.lock_retention_policy
  kms_key_name             = var.archive_kms_key_name

  labels = {
    boundary = "dependency-authority"
    zone     = "evidence"
  }
}

module "workload_identity" {
  source     = "../../modules/workload-identity"
  project_id = var.project_id
  pool_id    = var.pool_id

  identities = {
    writer = merge(var.writer, {
      trigger_service_account_id = local.workload_jobs["dep-evidence-write"].trigger_id
    })
    auditor = merge(var.auditor, {
      trigger_service_account_id = local.workload_jobs["dep-evidence-audit"].trigger_id
    })
  }
}

module "repository_iam" {
  source   = "../../modules/repository-iam"
  for_each = var.ecosystems

  project_id = var.project_id
  location   = var.location
  repository = module.repositories[each.key].id
  writers = concat(
    ["serviceAccount:${module.workload_identity.service_account_emails["writer"]}"],
    tolist(var.additional_writer_members),
  )
  readers = concat(
    ["serviceAccount:${module.workload_identity.service_account_emails["auditor"]}"],
    tolist(var.additional_auditor_members),
  )
}

module "workload_jobs" {
  source   = "../../modules/cloud-run-job"
  for_each = local.workload_jobs

  project_id            = var.project_id
  location              = var.location
  name                  = each.key
  service_account_email = module.workload_identity.service_account_emails[each.value.identity_key]
  invoker_member        = "serviceAccount:${module.workload_identity.trigger_service_account_emails[each.value.identity_key]}"
  image                 = var.workload_job_images[each.key]
  network               = module.network.workload_network_id
  subnetwork            = module.network.workload_subnetwork_id

  labels = {
    boundary = "dependency-authority"
    zone     = "evidence"
  }
}

module "audit_log_sink" {
  source             = "../../modules/logging"
  project_id         = var.project_id
  destination_bucket = module.evidence_archive.name

  sinks = {
    (var.audit_sink_name) = {
      filter = var.audit_log_filter
    }
  }
}

# The forensics reader access class: the organization-owned forensics group
# holds exactly the two read-only diagnostic roles on this zone project and no
# other grant; the group is instance-bound, never a stack literal.
module "forensics_readers" {
  source     = "../../modules/forensics-readers"
  project_id = var.project_id

  forensics_group = var.forensics_group
}
