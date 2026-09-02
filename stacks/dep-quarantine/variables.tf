variable "project_id" {
  description = "Google Cloud project ID of the quarantine trust zone (<organization>-dep-quarantine). Supplied by the organization instance; the core never carries a default."
  type        = string
}

variable "location" {
  description = "Artifact Registry location of the quarantine repositories."
  type        = string
}

variable "ecosystems" {
  description = "Package ecosystems receiving quarantine repositories. Mirrors the intake projection."
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
  description = "Workload Identity Pool ID of the quarantine zone."
  type        = string
  default     = "dep-quarantine"
}

variable "identities" {
  description = <<-EOT
    Optional workload identities of the quarantine zone (for example a future
    investigation lane). With the default empty map only the zone pool is
    created; provider and service-account bindings follow through a governed
    change, and every identity binds a dedicated invoke-only trigger service
    account through trigger_service_account_id.
  EOT
  type = map(object({
    provider_id                = string
    service_account_id         = string
    trigger_service_account_id = string
    display_name               = optional(string, "")
    description                = optional(string, "")
    issuer_uri                 = optional(string, "https://token.actions.githubusercontent.com")
    allowed_audiences          = optional(list(string), [])
    attribute_mapping          = optional(map(string))
    attribute_condition        = string
    principal_attribute        = optional(string, "repository")
    principal_value            = string
    roles                      = optional(set(string), [])
  }))
  default = {}
}

variable "writer_members" {
  description = "Members allowed to write quarantine candidates, for example the admission controller service account of the control zone. Wired by the organization instance."
  type        = set(string)
}

variable "reader_members" {
  description = "Members allowed to read quarantine candidates for investigation. Defaults to none."
  type        = set(string)
  default     = []
}

variable "evidence_bucket_name" {
  description = "Name of the evidence archive bucket provisioned by the dep-evidence stack. The quarantine zone exports its audit logs there; provisioning order is dep-evidence first."
  type        = string
}

variable "audit_sink_name" {
  description = "Name of the audit log sink into the evidence archive."
  type        = string
  default     = "dep-quarantine-audit-to-evidence"
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
