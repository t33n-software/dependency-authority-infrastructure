resource "google_logging_project_sink" "this" {
  for_each = var.sinks

  project                = var.project_id
  name                   = each.key
  destination            = "storage.googleapis.com/${var.destination_bucket}"
  filter                 = each.value.filter
  description            = each.value.description
  unique_writer_identity = true

  dynamic "exclusions" {
    for_each = each.value.exclusions
    content {
      name        = exclusions.value.name
      filter      = exclusions.value.filter
      description = exclusions.value.description
      disabled    = exclusions.value.disabled
    }
  }
}

resource "google_storage_bucket_iam_member" "sink_writer" {
  for_each = google_logging_project_sink.this

  bucket = var.destination_bucket
  role   = var.writer_role
  member = each.value.writer_identity
}
