output "sink_names" {
  description = "Created sink resource names, keyed by sink name."
  value       = { for key, sink in google_logging_project_sink.this : key => sink.name }
}

output "writer_identities" {
  description = "Unique writer identities of the sinks, keyed by sink name."
  value       = { for key, sink in google_logging_project_sink.this : key => sink.writer_identity }
}
