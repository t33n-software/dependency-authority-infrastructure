variable "project_id" {
  description = "Google Cloud project ID of the evidence trust zone (<organization>-dep-evidence). Supplied by the organization instance; the core never carries a default."
  type        = string
}

variable "location" {
  description = "Artifact Registry location of the evidence repositories and Cloud Storage location of the retention archive."
  type        = string
}

variable "ecosystems" {
  description = "Package ecosystems receiving generic evidence repositories. Mirrors the intake projection."
  type        = set(string)
  default     = ["go"]

  validation {
    condition     = length(setsubtract(var.ecosystems, ["go", "npm", "python"])) == 0
    error_message = "ecosystems only supports go, npm and python."
  }

  validation {
    condition     = length(var.ecosystems) > 0
    error_message = "at least one ecosystem is required."
  }
}

variable "pool_id" {
  description = "Workload Identity Pool ID of the evidence zone."
  type        = string
  default     = "dep-evidence"
}

variable "writer" {
  description = <<-EOT
    Workload identity binding of the evidence writer (canonical identity class
    dep-evidence-writer). The organization instance binds the exact repository,
    protected workflow reference, environment and audience through
    attribute_condition and principal_value.
  EOT
  type = object({
    provider_id         = string
    service_account_id  = optional(string, "dep-evidence-writer")
    attribute_condition = string
    principal_attribute = optional(string, "repository")
    principal_value     = string
    roles               = optional(set(string), [])
  })
}

variable "auditor" {
  description = <<-EOT
    Workload identity binding of the evidence auditor (canonical identity class
    dep-evidence-auditor). The organization instance binds the exact repository,
    protected workflow reference, environment and audience through
    attribute_condition and principal_value.
  EOT
  type = object({
    provider_id         = string
    service_account_id  = optional(string, "dep-evidence-auditor")
    attribute_condition = string
    principal_attribute = optional(string, "repository")
    principal_value     = string
    roles               = optional(set(string), [])
  })
}

variable "archive_bucket_name" {
  description = "Globally unique Cloud Storage bucket name of the long-term immutable retention archive."
  type        = string
}

variable "retention_period_seconds" {
  description = "Retention period in seconds applied to every archived object. The duration is an approved instance decision; the core never carries a default."

  type = number

  validation {
    condition     = var.retention_period_seconds > 0
    error_message = "retention_period_seconds must be positive."
  }
}

variable "lock_retention_policy" {
  description = "Irreversibly locks the archive retention policy. Only set to true after the retention and legal-hold duration is approved and recorded by the organization instance."
  type        = bool
  default     = false
}

variable "archive_kms_key_name" {
  description = "Optional CMEK key resource name used as the archive bucket default encryption key."
  type        = string
  default     = null
}

variable "additional_auditor_members" {
  description = "Additional read-only auditor members of the evidence repositories beyond the auditor identity. Defaults to none."
  type        = set(string)
  default     = []
}

variable "workload_job_images" {
  description = <<-EOT
    Workload job image bindings of the evidence zone, keyed by the canonical
    job name (dep-evidence-write, dep-evidence-audit). The organization
    instance supplies the full immutable digest of the promoted image from
    the release-class workload image registry; a documented placeholder keeps
    the binding planned and fails provisioning closed until the promotion
    read-back proof exists.
  EOT
  type        = map(string)
}

variable "audit_sink_name" {
  description = "Name of the audit log sink of the evidence zone into its own archive."
  type        = string
  default     = "dep-evidence-audit-to-archive"
}

variable "audit_log_filter" {
  description = "Cloud Logging filter for the zone audit export. Defaults to all Cloud Audit Logs of the zone project."
  type        = string
  default     = "logName:\"logs/cloudaudit.googleapis.com\""
}

variable "policy_constraints" {
  description = "Project-level organization-policy compensation for the absent organization node. All constraints default to the enforced posture."
  type = object({
    disable_service_account_key_creation = optional(bool, true)
    disable_service_account_key_upload   = optional(bool, true)
    uniform_bucket_level_access          = optional(bool, true)
    public_access_prevention             = optional(bool, true)
  })
  default = {}
}
