resource "google_cloud_run_v2_job" "this" {
  name     = var.name
  project  = var.project_id
  location = var.location
  labels   = var.labels

  template {
    template {
      service_account = var.service_account_email

      containers {
        image = var.image

        dynamic "env" {
          for_each = var.env
          content {
            name  = env.key
            value = env.value
          }
        }
      }
    }
  }
}
