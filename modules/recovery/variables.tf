variable "project_id" {
  description = "Google Cloud project ID of the trust zone receiving the break-glass identity. The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "organization_number" {
  description = "Numeric organization number of the Google Cloud organization carrying the zone: the platform embeds it in the organization-level Privileged Access Manager service agent identity (service-org-<number>@gcp-sa-pam.iam.gserviceaccount.com), which the module binds on the zone project as the standing platform setup of the privileged-access surface. The organization instance supplies this value; the core never presets it."
  type        = string

  validation {
    condition     = can(regex("^[0-9]+$", var.organization_number))
    error_message = "organization_number must be the numeric Google Cloud organization number."
  }
}

variable "service_account_id" {
  description = "Service account ID of the break-glass recovery identity."
  type        = string
  default     = "dep-break-glass-recovery"
}

variable "display_name" {
  description = "Display name of the break-glass recovery service account."
  type        = string
  default     = "Dependency Authority break-glass recovery"
}

variable "description" {
  description = "Description of the break-glass recovery service account."
  type        = string
  default     = "Time-bounded, audited break-glass identity of the dependency authority; never used in normal operation."
}

variable "entitlement_id" {
  description = "Entitlement ID of the break-glass recovery privileged-access entitlement (the platform form: 4-63 characters of lowercase letters, digits and hyphens, beginning with a letter). The identity class name is canonical; the core carries it, never the instance."
  type        = string
  default     = "dep-break-glass-recovery-admin"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{3,62}$", var.entitlement_id))
    error_message = "entitlement_id must satisfy the platform form: 4-63 characters of lowercase letters, digits and hyphens, beginning with a letter."
  }
}

variable "max_request_duration" {
  description = "The per-activation grant duration of the break-glass entitlement in the platform's seconds form (for example \"7200s\"): the platform-enforced time-box of every activation — never an IAM condition, which platforms reject on primitive roles. The organization instance binds the canonical time-box value as an approved decision; the core never presets it."
  type        = string

  validation {
    condition     = can(regex("^[1-9][0-9]*s$", var.max_request_duration))
    error_message = "max_request_duration must be a duration in the platform's seconds form (for example \"7200s\")."
  }
}

variable "approvers" {
  description = "The approver principal set of the entitlement's approval workflow in the user:, group: or serviceAccount: form: every activation is approval- and justification-bound. The organization instance binds the set as an approved decision — distinct from the eligible principal in the target form; any self-approval is a documented, expiring interim state, never the target. The core never presets it."
  type        = set(string)

  validation {
    condition     = length(var.approvers) > 0
    error_message = "approvers must carry at least one approver principal; the approval duty of every activation is mandatory."
  }

  validation {
    condition = alltrue([
      for principal in var.approvers : can(regex("^(user|group|serviceAccount):[^\\s]+$", principal))
    ])
    error_message = "every approver must be a principal in the user:, group: or serviceAccount: form."
  }
}
