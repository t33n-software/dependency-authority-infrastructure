variable "project_id" {
  description = "Google Cloud project ID of the approved trust zone (<organization>-dep-approved). Supplied by the organization instance; the core never carries a default."
  type        = string
}

variable "location" {
  description = "Artifact Registry location of the approved repositories."
  type        = string
}

variable "ecosystems" {
  description = "Package ecosystems receiving approved repositories. Mirrors the intake projection."
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
  description = "Workload Identity Pool ID of the approved zone."
  type        = string
  default     = "dep-approved"
}

variable "promoter" {
  description = <<-EOT
    Workload identity binding of the approved promoter (canonical identity
    class dep-approved-promoter). The organization instance binds the exact
    repository, protected workflow reference, environment and audience through
    attribute_condition and principal_value.
  EOT
  type = object({
    provider_id         = string
    service_account_id  = optional(string, "dep-approved-promoter")
    attribute_condition = string
    principal_attribute = optional(string, "repository")
    principal_value     = string
    roles               = optional(set(string), [])
  })
}

variable "consumer_members" {
  description = "Read-only consumer members of the approved repositories (developer, CI, builder and release identities). Wired by the organization instance; intake and quarantine identities never appear here."
  type        = set(string)
  default     = []
}

variable "evidence_bucket_name" {
  description = "Name of the evidence archive bucket provisioned by the dep-evidence stack. The approved zone exports its audit logs there; provisioning order is dep-evidence first."
  type        = string
}

variable "audit_sink_name" {
  description = "Name of the audit log sink into the evidence archive."
  type        = string
  default     = "dep-approved-audit-to-evidence"
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
