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

variable "identities" {
  description = <<-EOT
    Optional workload identities of the approved zone. The canonical IAM
    target matrix binds no zone-local identity here: the approved promoter,
    the revocation controller and the revalidation controller are control-zone
    identities bound through the member inputs of this stack. With the default
    empty map only the zone pool is created; any future zone-local identity is
    a governed change.
  EOT
  type = map(object({
    provider_id         = string
    service_account_id  = string
    display_name        = optional(string, "")
    description         = optional(string, "")
    issuer_uri          = optional(string, "https://token.actions.githubusercontent.com")
    allowed_audiences   = optional(list(string), [])
    attribute_mapping   = optional(map(string))
    attribute_condition = string
    principal_attribute = optional(string, "repository")
    principal_value     = string
    roles               = optional(set(string), [])
  }))
  default = {}
}

variable "promoter_member" {
  description = "Member receiving write access on the approved repositories for promotion — canonically the service account of the dep-approved-promoter identity of the control zone, wired by the organization instance as part of the canonical IAM target matrix."
  type        = string

  validation {
    condition     = can(regex("^serviceAccount:[a-z][a-z0-9-]*@[a-z][a-z0-9-]*\\.iam\\.gserviceaccount\\.com$", var.promoter_member))
    error_message = "promoter_member must be a fully formed serviceAccount member string of the control-zone promoter identity."
  }
}

variable "revocation_member" {
  description = "Member receiving write access on the approved repositories for revocation download-rule enforcement — canonically the service account of the dep-revocation-controller identity of the control zone, wired by the organization instance as part of the canonical IAM target matrix. The revocation download rules themselves stay runtime operations and are never static stack content."
  type        = string

  validation {
    condition     = can(regex("^serviceAccount:[a-z][a-z0-9-]*@[a-z][a-z0-9-]*\\.iam\\.gserviceaccount\\.com$", var.revocation_member))
    error_message = "revocation_member must be a fully formed serviceAccount member string of the control-zone revocation controller identity."
  }
}

variable "revalidation_reader_member" {
  description = "Member receiving read access on the approved repositories for periodic revalidation — canonically the service account of the dep-revalidation-controller identity of the control zone, wired by the organization instance as part of the canonical IAM target matrix."
  type        = string

  validation {
    condition     = can(regex("^serviceAccount:[a-z][a-z0-9-]*@[a-z][a-z0-9-]*\\.iam\\.gserviceaccount\\.com$", var.revalidation_reader_member))
    error_message = "revalidation_reader_member must be a fully formed serviceAccount member string of the control-zone revalidation controller identity."
  }
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
