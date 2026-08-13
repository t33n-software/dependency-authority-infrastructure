variable "project_id" {
  description = "Google Cloud project ID of the trust zone receiving the policy constraints. The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "disable_service_account_key_creation" {
  description = "Enforce iam.disableServiceAccountKeyCreation on the project. Static service-account keys are never a dependency authority credential mechanism."
  type        = bool
  default     = true
}

variable "disable_service_account_key_upload" {
  description = "Enforce iam.disableServiceAccountKeyUpload on the project. Externally generated key material is never uploaded into the trust zones."
  type        = bool
  default     = true
}

variable "uniform_bucket_level_access" {
  description = "Enforce storage.uniformBucketLevelAccess on the project. Object-level ACLs never weaken the evidence archive boundary."
  type        = bool
  default     = true
}

variable "public_access_prevention" {
  description = "Enforce storage.publicAccessPrevention on the project. Evidence and archive buckets are never public."
  type        = bool
  default     = true
}
