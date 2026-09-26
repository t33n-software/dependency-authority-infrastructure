# The curated predefined-role set of the zone's declared surface: derived from
# the uniform module inventory of the zone stacks and proven complete by the
# infrastructure core's contract guard — every declared resource class maps to
# its covering administrative role, and a growing declaration forces the set to
# grow with it. Never a legacy basic role: the privileged-access mechanism does
# not admit owner, editor or viewer, and platforms reject IAM conditions on
# primitive roles — both forms are proven non-deployable.
locals {
  break_glass_recovery_roles = [
    "roles/artifactregistry.admin",
    "roles/cloudkms.admin",
    "roles/compute.networkAdmin",
    "roles/compute.securityAdmin",
    "roles/dns.admin",
    "roles/iam.serviceAccountAdmin",
    "roles/iam.workloadIdentityPoolAdmin",
    "roles/logging.configWriter",
    "roles/orgpolicy.policyAdmin",
    "roles/privilegedaccessmanager.admin",
    "roles/resourcemanager.projectIamAdmin",
    "roles/run.admin",
    "roles/serviceusage.serviceUsageAdmin",
    "roles/storage.admin",
  ]
}

resource "google_service_account" "this" {
  project      = var.project_id
  account_id   = var.service_account_id
  display_name = var.display_name
  description  = var.description
}

# The elevated capability of the break-glass recovery identity: no standing
# grant. The declared privileged-access entitlement binds the curated
# predefined-role set of the zone's declared surface as the privileged access
# of exactly this identity, dormant until activated; every activation is
# approval- and justification-bound, time-boxed by the platform-enforced grant
# duration and audited, and the grant auto-expires at the end of the
# activation window.
resource "google_privileged_access_manager_entitlement" "this" {
  parent               = "projects/${var.project_id}"
  location             = "global"
  entitlement_id       = var.entitlement_id
  max_request_duration = var.max_request_duration

  eligible_users {
    principals = ["serviceAccount:${google_service_account.this.email}"]
  }

  privileged_access {
    gcp_iam_access {
      resource      = "//cloudresourcemanager.googleapis.com/projects/${var.project_id}"
      resource_type = "cloudresourcemanager.googleapis.com/Project"

      dynamic "role_bindings" {
        for_each = toset(local.break_glass_recovery_roles)
        content {
          role = role_bindings.value
        }
      }
    }
  }

  requester_justification_config {
    unstructured {}
  }

  approval_workflow {
    manual_approvals {
      require_approver_justification = true

      steps {
        approvals_needed = 1

        approvers {
          principals = var.approvers
        }
      }
    }
  }
}
