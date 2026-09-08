variable "project_id" {
  description = "Google Cloud project ID that owns the repository. The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "location" {
  description = "Artifact Registry location (region or multi-region) of the repository."
  type        = string
}

variable "repository_id" {
  description = "Repository ID (final component of the repository resource name). Canonical dependency authority names follow <ecosystem>-dependencies-<zone>."
  type        = string
}

variable "description" {
  description = "Human-readable repository description. Never carries sensitive data; descriptions are not encrypted."
  type        = string
  default     = ""
}

variable "format" {
  description = "Package format. The dependency authority operates GO, NPM and PYTHON package repositories, GENERIC evidence repositories and DOCKER workload image repositories."
  type        = string

  validation {
    condition     = contains(["GO", "NPM", "PYTHON", "GENERIC", "DOCKER"], var.format)
    error_message = "format must be one of GO, NPM, PYTHON, GENERIC, DOCKER."
  }
}

variable "mode" {
  description = "Repository mode. Intake zones use REMOTE_REPOSITORY; quarantine, approved and evidence zones use STANDARD_REPOSITORY. DOCKER workload image repositories always use STANDARD_REPOSITORY. Virtual consumer endpoints are out of scope for this module."
  type        = string
  default     = "STANDARD_REPOSITORY"

  validation {
    condition     = contains(["STANDARD_REPOSITORY", "REMOTE_REPOSITORY"], var.mode)
    error_message = "mode must be STANDARD_REPOSITORY or REMOTE_REPOSITORY."
  }

  validation {
    condition     = var.format != "DOCKER" || var.mode == "STANDARD_REPOSITORY"
    error_message = "DOCKER workload image repositories always use STANDARD_REPOSITORY mode; the workload class never binds a remote upstream."
  }
}

variable "remote_upstream" {
  description = <<-EOT
    Remote upstream binding, required exactly once for REMOTE_REPOSITORY mode and
    forbidden otherwise. Exactly one field must be set:
    - npm: public upstream enum (only NPMJS)
    - python: public upstream enum (only PYPI)
    - common_uri: common upstream URI (Go remote intake uses https://proxy.golang.org)
  EOT
  type = object({
    npm        = optional(string)
    python     = optional(string)
    common_uri = optional(string)
  })
  default = null

  validation {
    condition     = (var.mode == "REMOTE_REPOSITORY") == (var.remote_upstream != null)
    error_message = "REMOTE_REPOSITORY mode requires exactly one remote_upstream binding; STANDARD_REPOSITORY mode forbids one."
  }

  validation {
    condition     = var.remote_upstream == null || length(compact([for _, value in var.remote_upstream : value])) == 1
    error_message = "remote_upstream must set exactly one of npm, python or common_uri."
  }

  validation {
    condition = var.remote_upstream == null || (
      (var.format == "GO" && var.remote_upstream.common_uri != null) ||
      (var.format == "NPM" && var.remote_upstream.npm != null) ||
      (var.format == "PYTHON" && var.remote_upstream.python != null)
    )
    error_message = "remote_upstream must match the repository format: GO uses common_uri, NPM uses npm, PYTHON uses python."
  }

  validation {
    condition     = var.remote_upstream == null || !contains(["GENERIC", "DOCKER"], var.format)
    error_message = "GENERIC evidence repositories and DOCKER workload image repositories do not support remote mode and therefore no remote_upstream binding."
  }
}

variable "vpcsc_upstream_allowance" {
  description = <<-EOT
    Whether the repository's zone operates inside a VPC Service Controls
    perimeter and therefore declares the registry-platform upstream allowance
    (vpcsc_policy = ALLOW) for its project and location. The default is the
    fail-closed denied posture; a zone with a remote repository inside the
    perimeter opts in. The allowance is scoped to the project and location and
    covers only the configured upstreams of the remote repositories there; it
    is never a perimeter egress rule.
  EOT
  type        = bool
  default     = false

  validation {
    condition     = !var.vpcsc_upstream_allowance || var.mode == "REMOTE_REPOSITORY"
    error_message = "vpcsc_upstream_allowance requires REMOTE_REPOSITORY mode; a standard repository never carries the upstream allowance."
  }
}

variable "cleanup_policies" {
  description = <<-EOT
    Cleanup policies keyed by policy ID. Intake zones use them for short-lived
    candidate retention; quarantine, approved and evidence zones keep them empty
    by default.
  EOT
  type = map(object({
    action = string
    condition = optional(object({
      tag_state             = optional(string)
      tag_prefixes          = optional(list(string))
      version_name_prefixes = optional(list(string))
      package_name_prefixes = optional(list(string))
      older_than            = optional(string)
      newer_than            = optional(string)
    }))
    most_recent_versions = optional(object({
      package_name_prefixes = optional(list(string))
      keep_count            = optional(number)
    }))
  }))
  default = {}

  validation {
    condition     = alltrue([for _, policy in var.cleanup_policies : contains(["DELETE", "KEEP"], policy.action)])
    error_message = "cleanup policy action must be DELETE or KEEP."
  }
}

variable "cleanup_policy_dry_run" {
  description = "When true, cleanup policies never delete versions. Defaults to the fail-safe dry-run posture."
  type        = bool
  default     = true
}

variable "kms_key_name" {
  description = "Optional CMEK key resource name (projects/*/locations/*/keyRings/*/cryptoKeys/*). Immutable after repository creation; provisioning order is instance-owned."
  type        = string
  default     = null
}

variable "labels" {
  description = "Repository labels. The reference stacks bind boundary, zone and ecosystem labels; the workload image registries bind boundary and zone."
  type        = map(string)
  default     = {}
}
