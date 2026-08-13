locals {
  bindings = merge(
    { for member in var.readers : "reader/${member}" => { role = "roles/artifactregistry.reader", member = member } },
    { for member in var.writers : "writer/${member}" => { role = "roles/artifactregistry.writer", member = member } },
  )
}

resource "google_artifact_registry_repository_iam_member" "this" {
  for_each = local.bindings

  project    = var.project_id
  location   = var.location
  repository = var.repository
  role       = each.value.role
  member     = each.value.member
}
