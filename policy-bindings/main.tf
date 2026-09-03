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

# The Cloud Run enforcement surface of the workload network origin form: the
# job-owning zones restrict the deployable VPC egress to all-traffic and the
# deployable ingress to internal, so the platform enforces the network origin
# form rather than convention alone. Both constraints are opt-in and bind only
# the job-owning zones.
resource "google_org_policy_policy" "cloud_run_vpc_egress" {
  count = var.cloud_run_vpc_egress_all_traffic_only ? 1 : 0

  name   = "projects/${var.project_id}/policies/run.allowedVPCEgress"
  parent = "projects/${var.project_id}"

  spec {
    rules {
      values {
        allowed_values = ["all-traffic"]
      }
    }
  }
}

resource "google_org_policy_policy" "cloud_run_ingress" {
  count = var.cloud_run_ingress_internal_only ? 1 : 0

  name   = "projects/${var.project_id}/policies/run.allowedIngress"
  parent = "projects/${var.project_id}"

  spec {
    rules {
      values {
        allowed_values = ["internal"]
      }
    }
  }
}
