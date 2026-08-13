output "id" {
  description = "Fully qualified repository resource ID (projects/*/locations/*/repositories/*)."
  value       = google_artifact_registry_repository.this.id
}

output "name" {
  description = "Repository resource name."
  value       = google_artifact_registry_repository.this.name
}

output "repository_id" {
  description = "Repository ID (final component of the resource name)."
  value       = google_artifact_registry_repository.this.repository_id
}

output "location" {
  description = "Repository location."
  value       = google_artifact_registry_repository.this.location
}

output "project" {
  description = "Owning project ID."
  value       = google_artifact_registry_repository.this.project
}

output "registry_uri" {
  description = "Repository endpoint URI."
  value       = google_artifact_registry_repository.this.registry_uri
}

output "format" {
  description = "Repository format."
  value       = google_artifact_registry_repository.this.format
}

output "mode" {
  description = "Repository mode."
  value       = google_artifact_registry_repository.this.mode
}
