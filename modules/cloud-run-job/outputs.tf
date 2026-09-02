output "id" {
  description = "Fully qualified job resource ID (projects/*/locations/*/jobs/*)."
  value       = google_cloud_run_v2_job.this.id
}

output "name" {
  description = "Job name (the canonical dep-<operation> form)."
  value       = google_cloud_run_v2_job.this.name
}

output "location" {
  description = "Job location."
  value       = google_cloud_run_v2_job.this.location
}
