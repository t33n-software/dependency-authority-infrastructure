resource "google_cloud_run_v2_job" "this" {
  name     = var.name
  project  = var.project_id
  location = var.location
  labels   = var.labels

  template {
    template {
      service_account = var.service_account_email

      # The workload network origin is part of the execution contract: the job
      # attaches to its zone VPC and routes all outgoing traffic through it, so
      # its calls to the restricted planes originate inside the perimeter. A
      # job without the attachment presents no in-perimeter network origin and
      # fails closed at the perimeter.
      vpc_access {
        egress = "ALL_TRAFFIC"

        network_interfaces {
          network    = var.network
          subnetwork = var.subnetwork
        }
      }

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
# resource-scoped to this job and nothing else. The lane passes the operation
# inputs as execution-parameter overrides of the invocation, so the invoke call
# is an override execution: roles/run.jobsExecutorWithOverrides carries
# run.jobs.run, run.jobs.runWithOverrides and run.executions.cancel (the proven
# role contents), and roles/run.viewer carries run.executions.get and
# run.executions.list for the status read-back (the remaining viewer
# permissions do not apply to a job resource).
resource "google_cloud_run_v2_job_iam_member" "invoker" {
  project  = var.project_id
  location = var.location
  name     = google_cloud_run_v2_job.this.name
  role     = "roles/run.jobsExecutorWithOverrides"
  member   = var.invoker_member
}

resource "google_cloud_run_v2_job_iam_member" "invoker_readback" {
  project  = var.project_id
  location = var.location
  name     = google_cloud_run_v2_job.this.name
  role     = "roles/run.viewer"
  member   = var.invoker_member
}
