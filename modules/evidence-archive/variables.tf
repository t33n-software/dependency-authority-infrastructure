variable "project_id" {
  description = "Google Cloud project ID of the evidence trust zone. The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "name" {
  description = "Globally unique Cloud Storage bucket name of the retention archive."
  type        = string
}

variable "location" {
  description = "Cloud Storage location (region or multi-region) of the archive bucket."
  type        = string
}

variable "retention_period_seconds" {
  description = "Retention period in seconds applied to every object in the archive. The duration is an approved instance decision; the core never carries a default."

  type = number

  validation {
    condition     = var.retention_period_seconds > 0
    error_message = "retention_period_seconds must be positive."
  }
}

variable "lock_retention_policy" {
  description = <<-EOT
    Irreversibly locks the bucket retention policy. This is a one-way
    operation: it may only be set to true after the retention and legal-hold
    duration is approved and recorded by the organization instance.
  EOT
  type        = bool
  default     = false
}

variable "kms_key_name" {
  description = "Optional CMEK key resource name used as the bucket default encryption key."
  type        = string
  default     = null
}

variable "labels" {
  description = "Bucket labels. The reference stack binds boundary and zone labels."
  type        = map(string)
  default     = {}
}
