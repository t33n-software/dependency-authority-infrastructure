# The zone state home: exactly one dedicated Cloud Storage bucket holding the
# zone's root states and nothing else. The form is the provider layer of the
# dual fortress state-encryption standard: uniform bucket-level access,
# enforced public access prevention, object versioning (the documented backend
# recommendation and the prerequisite of the key-version destruction
# discipline) and the mandatory bucket CMEK with the second, cryptographically
# separate key. A state bucket never carries a retention policy: the state
# layer is the recovery root, not an archive.
resource "google_storage_bucket" "this" {
  project  = var.project_id
  name     = var.name
  location = var.location

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }

  encryption {
    default_kms_key_name = var.cmek_key_name
  }

  labels = var.labels
}

# The bucket data plane of the engine: exactly the storage object-admin role,
# resource-sharp on this bucket, for the instance-bound operator execution
# identities. No other role is ever granted through this module.
resource "google_storage_bucket_iam_member" "operators" {
  for_each = var.operator_members

  bucket = google_storage_bucket.this.name
  role   = "roles/storage.objectAdmin"
  member = each.value
}
