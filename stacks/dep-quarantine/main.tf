provider "google" {
  project = var.project_id
}

# The exactly pinned beta provider of the declaration plane: it declares only
# what the pinned GA provider provably lacks (the remote upstream allowance);
# everything else stays on the GA provider.
provider "google-beta" {
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

  project_id    = var.project_id
  location      = var.location
  repository_id = "${each.key}-dependencies-quarantine"
  description   = "Dependency authority ${each.key} quarantine: isolated blocked or incomplete candidates, never a consumer endpoint"
  format        = upper(each.key)
  mode          = "STANDARD_REPOSITORY"

  labels = {
    boundary  = "dependency-authority"
    zone      = "quarantine"
    ecosystem = each.key
  }
}

module "workload_identity" {
  source     = "../../modules/workload-identity"
  project_id = var.project_id
  pool_id    = var.pool_id

  identities = var.identities
}

module "repository_iam" {
  source   = "../../modules/repository-iam"
  for_each = var.ecosystems

  project_id = var.project_id
  location   = var.location
  repository = module.repositories[each.key].id
  writers    = var.writer_members
  readers    = var.reader_members
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

# The forensics reader access class: the organization-owned forensics group
# holds exactly the two read-only diagnostic roles on this zone project and no
# other grant; the group is instance-bound, never a stack literal.
module "forensics_readers" {
  source     = "../../modules/forensics-readers"
  project_id = var.project_id

  forensics_group = var.forensics_group
}
