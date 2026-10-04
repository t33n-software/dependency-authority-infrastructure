resource "google_storage_bucket" "this" {
  project  = var.project_id
  name     = var.name
  location = var.location

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }

  retention_policy {
    is_locked        = var.lock_retention_policy
    retention_period = var.retention_period_seconds
  }

  dynamic "encryption" {
    for_each = var.kms_key_name == null ? [] : [var.kms_key_name]
    content {
      default_kms_key_name = encryption.value
    }
  }

  labels = var.labels
}

# The standing audit export writers of the organization plane: the Cloud
# Logging service agents routing the organization node's, the folder
# grouping layer's and the organization anchor project's audit trails into
# this archive append their routed log entries here. The role is the
# canonical, documentation-proven destination role for Cloud Storage log
# sink destinations (roles/storage.objectCreator): append-focused, never
# read, never delete, and bound as a fixed literal — never a configurable
# wider role and never a window grant.
resource "google_storage_bucket_iam_member" "logging_export_writer" {
  for_each = var.logging_export_writer_members

  bucket = google_storage_bucket.this.name
  role   = "roles/storage.objectCreator"
  member = each.value
}
