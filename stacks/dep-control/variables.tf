variable "project_id" {
  description = "Google Cloud project ID of the control trust zone (<organization>-dep-control). Supplied by the organization instance; the core never carries a default."
  type        = string
}

variable "location" {
  description = "Artifact Registry location of the control-zone workload image registries."
  type        = string
}

variable "pool_id" {
  description = "Workload Identity Pool ID of the control zone."
  type        = string
  default     = "dep-control"
}

variable "controllers" {
  description = <<-EOT
    Control-plane controller workload identities, keyed by lane (canonically
    admission, promotion, revalidation and revocation — the workload job
    topology references exactly these keys). The organization instance binds
    the exact repository, protected workflow reference, environment and
    audience through attribute_condition and principal_value, and assigns the
    canonical identity class names through service_account_id (for example
    dep-admission-controller).
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

  validation {
    condition     = length(var.controllers) > 0
    error_message = "at least one controller identity is required."
  }
}

variable "evidence_bucket_name" {
  description = "Name of the evidence archive bucket provisioned by the dep-evidence stack. The control zone exports its audit logs there; provisioning order is dep-evidence first."
  type        = string
}

variable "workload_job_images" {
  description = <<-EOT
    Workload job image bindings of the control zone, keyed by the canonical
    job name (dep-admission, dep-promotion, dep-revalidation, dep-revocation).
    The organization instance supplies the full immutable digest of the
    promoted image from the release-class workload image registry; a
    documented placeholder keeps the binding planned and fails provisioning
    closed until the promotion read-back proof exists.
  EOT
  type        = map(string)
}

variable "cross_zone_workload_reader_members" {
  description = "Workload identities of the other zones receiving read access on the release-class workload image registry across the project boundary — canonically the intake fetcher, the evidence writer and the evidence auditor, as bound by the canonical IAM target matrix. Wired by the organization instance; no identity ever receives a writer grant on either workload image registry class."
  type        = set(string)
  default     = []
}

variable "audit_sink_name" {
  description = "Name of the audit log sink into the evidence archive."
  type        = string
  default     = "dep-control-audit-to-evidence"
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
