resource "google_service_account" "this" {
  project      = var.project_id
  account_id   = var.service_account_id
  display_name = var.display_name
  description  = var.description
}

resource "google_project_iam_member" "this" {
  project = var.project_id
  role    = var.role
  member  = "serviceAccount:${google_service_account.this.email}"

  condition {
    title       = var.condition_title
    description = var.condition_description
    expression  = "request.time < timestamp('${var.condition_end_time}')"
  }
}
