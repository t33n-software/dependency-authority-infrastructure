locals {
  boolean_constraints = {
    "iam.disableServiceAccountKeyCreation" = var.disable_service_account_key_creation
    "iam.disableServiceAccountKeyUpload"   = var.disable_service_account_key_upload
    "storage.uniformBucketLevelAccess"     = var.uniform_bucket_level_access
    "storage.publicAccessPrevention"       = var.public_access_prevention
  }
}

resource "google_org_policy_policy" "this" {
  for_each = { for constraint, enforced in local.boolean_constraints : constraint => enforced if enforced }

  name   = "projects/${var.project_id}/policies/${each.key}"
  parent = "projects/${var.project_id}"

  spec {
    rules {
      enforce = "TRUE"
    }
  }
}
