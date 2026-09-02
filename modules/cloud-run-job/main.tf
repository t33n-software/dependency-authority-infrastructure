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

# The external trigger seam: the lane's dedicated invoke-only trigger identity
# holds invoke on exactly this job and the execution status read-back, both
# resource-scoped to this job and nothing else. roles/run.invoker carries
# run.jobs.run; roles/run.viewer carries run.executions.get and
# run.executions.list (the proven role contents; the remaining viewer
# permissions do not apply to a job resource).
resource "google_cloud_run_v2_job_iam_member" "invoker" {
  project  = var.project_id
  location = var.location
  name     = google_cloud_run_v2_job.this.name
  role     = "roles/run.invoker"
  member   = var.invoker_member
}

resource "google_cloud_run_v2_job_iam_member" "invoker_readback" {
  project  = var.project_id
  location = var.location
  name     = google_cloud_run_v2_job.this.name
  role     = "roles/run.viewer"
  member   = var.invoker_member
}
