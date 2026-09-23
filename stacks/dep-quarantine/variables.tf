variable "project_id" {
  description = "Google Cloud project ID of the quarantine trust zone (<organization>-dep-quarantine). Supplied by the organization instance; the core never carries a default."
  type        = string
}

variable "project_number" {
  description = "Google Cloud project number of the quarantine trust zone: the workload identity pool binds it because the provider state carries the pool's project as the numeric project number, and binding the project ID would force a destroy-and-recreate of the pool; every other surface keeps the project ID. Supplied by the organization instance; the core never presets it."
  type        = string

  validation {
    condition     = can(regex("^[0-9]+$", var.project_number))
    error_message = "project_number must be the numeric Google Cloud project number of the zone."
  }
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

variable "forensics_group" {
  description = "Member string of the organization-owned forensics group (the canonical identity class dep-forensics-readers) receiving the read-only diagnostic bindings on this zone project. Supplied by the organization instance; never carried by the core."
  type        = string

  validation {
    condition     = can(regex("^group:dep-forensics-readers@", var.forensics_group))
    error_message = "forensics_group must be the group member string of the canonical dep-forensics-readers identity class."
  }
}

variable "state_bucket_name" {
  description = "Globally unique Cloud Storage bucket name of the zone state home, provisioned by the converged foundation and bound by the organization instance from the naming grammar family <organization>-<boundary>-<purpose>. The stack never assigns one and never provisions the bucket."
  type        = string

  validation {
    condition = (
      can(regex("^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$", var.state_bucket_name))
      && !can(regex("^[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+$", var.state_bucket_name))
      && !startswith(var.state_bucket_name, "goog")
      && !can(regex("google", var.state_bucket_name))
    )
    error_message = "state_bucket_name must satisfy the Cloud Storage bucket naming rules: 3-63 characters of lowercase letters, digits, hyphens, underscores and dots, alphanumeric edges, never an IP form, never the goog prefix and never google or similar spellings."
  }
}

variable "state_encryption_key" {
  description = "Instance-bound GCP KMS key reference of the client-side state and plan encryption engine layer of this root (projects/*/locations/*/keyRings/*/cryptoKeys/*). The organization instance supplies this value as reviewed configuration; the core never presets one."
  type        = string

  validation {
    condition     = can(regex("^projects/[^/]+/locations/[^/]+/keyRings/[^/]+/cryptoKeys/[^/]+$", var.state_encryption_key))
    error_message = "state_encryption_key must be a full GCP KMS key resource name."
  }
}

variable "break_glass_recovery" {
  description = <<-EOT
    The approved recovery binding of the quarantine zone: the project-level
    role the recovery identity receives under the mandatory time-bound IAM
    condition and the RFC 3339 UTC end time after which the grant stops
    applying. Both are approved instance decisions; the core never presets
    them.
  EOT
  type = object({
    role               = string
    condition_end_time = string
  })

  validation {
    condition = (
      can(regex("^roles/[A-Za-z][A-Za-z0-9._]+$", var.break_glass_recovery.role))
      || can(regex("^projects/[a-z][a-z0-9-]*/roles/[A-Za-z][A-Za-z0-9_]*$", var.break_glass_recovery.role))
    )
    error_message = "break_glass_recovery.role must be a predefined Google Cloud role (roles/<role>) or a project-level custom role (projects/<project>/roles/<roleId>)."
  }

  validation {
    condition     = can(timecmp(var.break_glass_recovery.condition_end_time, "1970-01-01T00:00:00Z"))
    error_message = "break_glass_recovery.condition_end_time must be a valid RFC 3339 timestamp."
  }
}
