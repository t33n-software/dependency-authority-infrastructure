variable "project_id" {
  description = "Google Cloud project ID of the trust zone receiving the break-glass identity. The organization instance supplies this value; the core never carries a default."
  type        = string
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

variable "role" {
  description = "Project-level role the break-glass identity receives under the time-bound condition. The exact role is an approved instance decision."
  type        = string
}

variable "condition_end_time" {
  description = "RFC 3339 UTC timestamp after which the break-glass grant stops applying, for example 2027-01-01T00:00:00Z. Time-boundedness is mandatory."

  type = string

  validation {
    condition     = can(timecmp(var.condition_end_time, "1970-01-01T00:00:00Z"))
    error_message = "condition_end_time must be a valid RFC 3339 timestamp."
  }
}

variable "condition_title" {
  description = "Title of the IAM condition that bounds the break-glass grant."
  type        = string
  default     = "time-bounded-break-glass"
}

variable "condition_description" {
  description = "Description of the IAM condition that bounds the break-glass grant."
  type        = string
  default     = "The grant expires automatically at the approved end time; usage is audited and reviewed."
}
