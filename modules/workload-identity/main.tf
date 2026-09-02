locals {
  default_attribute_mapping = {
    "google.subject"          = "assertion.sub"
    "attribute.actor"         = "assertion.actor"
    "attribute.aud"           = "assertion.aud"
    "attribute.repository"    = "assertion.repository"
    "attribute.workflow_ref"  = "assertion.workflow_ref"
    "attribute.environment"   = "assertion.environment"
    "attribute.ref"           = "assertion.ref"
    "attribute.ref_type"      = "assertion.ref_type"
    "attribute.repository_id" = "assertion.repository_id"
  }

  role_bindings = length(var.identities) == 0 ? {} : merge([
    for key, identity in var.identities : {
      for role in identity.roles : "${key}/${role}" => {
        identity = key
        role     = role
      }
    }
  ]...)
}

resource "google_iam_workload_identity_pool" "this" {
  project                   = var.project_id
  workload_identity_pool_id = var.pool_id
  display_name              = var.pool_display_name != "" ? var.pool_display_name : null
  description               = var.pool_description != "" ? var.pool_description : null
}

resource "google_iam_workload_identity_pool_provider" "this" {
  for_each = var.identities

  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.this.workload_identity_pool_id
  workload_identity_pool_provider_id = each.value.provider_id
  display_name                       = each.value.display_name != "" ? each.value.display_name : null
  description                        = each.value.description != "" ? each.value.description : null
  attribute_condition                = each.value.attribute_condition
  attribute_mapping                  = coalesce(each.value.attribute_mapping, local.default_attribute_mapping)

  oidc {
    issuer_uri        = each.value.issuer_uri
    allowed_audiences = length(each.value.allowed_audiences) > 0 ? each.value.allowed_audiences : null
  }
}

resource "google_service_account" "this" {
  for_each = var.identities

  project      = var.project_id
  account_id   = each.value.service_account_id
  display_name = each.value.display_name != "" ? each.value.display_name : null
  description  = each.value.description != "" ? each.value.description : null
}

# The dedicated invoke-only trigger identity of each lane: the lane principal
# set federates to this identity, never to the execution identity, and this
# identity never receives a role anywhere.
resource "google_service_account" "trigger" {
  for_each = var.identities

  project     = var.project_id
  account_id  = each.value.trigger_service_account_id
  description = "Invoke-only trigger identity of the lane; holds invoke permission on exactly its own workload job and never a data-plane role."
}

resource "google_service_account_iam_member" "workload_identity_user" {
  for_each = var.identities

  service_account_id = google_service_account.trigger[each.key].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.this.name}/attribute.${each.value.principal_attribute}/${each.value.principal_value}"
}

resource "google_project_iam_member" "identity_roles" {
  for_each = local.role_bindings

  project = var.project_id
  role    = each.value.role
  member  = "serviceAccount:${google_service_account.this[each.value.identity].email}"
}
