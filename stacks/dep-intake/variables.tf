variable "project_id" {
  description = "Google Cloud project ID of the intake trust zone (<organization>-dep-intake). Supplied by the organization instance; the core never carries a default."
  type        = string
}

variable "project_number" {
  description = "Google Cloud project number of the intake trust zone: the workload identity pool binds it because the provider state carries the pool's project as the numeric project number, and binding the project ID would force a destroy-and-recreate of the pool; every other surface keeps the project ID. Supplied by the organization instance; the core never presets it."
  type        = string

  validation {
    condition     = can(regex("^[0-9]+$", var.project_number))
    error_message = "project_number must be the numeric Google Cloud project number of the zone."
  }
}

variable "location" {
  description = "Artifact Registry location of the intake repositories and the job region of the zone workload network."
  type        = string
}

variable "workload_network" {
  description = <<-EOT
    The zone workload network origin binding: the VPC network name, the
    subnetwork name and the subnetwork CIDR of the zone VPC. The stack
    declares exactly one VPC with one subnetwork in the job region (the stack
    location) carrying Private Google Access, the restricted-range DNS
    response policy and the egress firewall pair; the zone's workload job
    attaches to it with Direct VPC egress and all-traffic routing. All values
    are instance-supplied.
  EOT
  type = object({
    network_name = string
    subnet_name  = string
    subnet_cidr  = string
  })

  validation {
    condition     = can(cidrhost(var.workload_network.subnet_cidr, 0))
    error_message = "workload_network.subnet_cidr must be a valid CIDR range."
  }
}

variable "ecosystems" {
  description = "Package ecosystems receiving remote intake repositories. Go is the default first ecosystem projection; npm and python follow after their acceptance evidence exists."
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
  description = "Workload Identity Pool ID of the intake zone."
  type        = string
  default     = "dep-intake"
}

variable "fetcher" {
  description = <<-EOT
    Workload identity binding of the intake fetcher (canonical identity class
    dep-intake-fetcher). The organization instance binds the exact repository,
    protected workflow reference, environment and audience through
    attribute_condition and principal_value.
  EOT
  type = object({
    provider_id         = string
    service_account_id  = optional(string, "dep-intake-fetcher")
    attribute_condition = string
    principal_attribute = optional(string, "repository")
    principal_value     = string
    roles               = optional(set(string), [])
  })
}

variable "additional_reader_members" {
  description = "Members receiving read access on the intake repositories beyond the fetcher writer — canonically the admission controller, the revalidation controller and the approved promoter identities of the control zone, as bound by the canonical IAM target matrix (the admission and revalidation lanes materialize the candidate content from the controlled intake boundary). Wired by the organization instance; the fetcher and any consumer identity never appear here."
  type        = set(string)
  default     = []
}

variable "evidence_bucket_name" {
  description = "Name of the evidence archive bucket provisioned by the dep-evidence stack. The intake zone exports its audit logs there; provisioning order is dep-evidence first."
  type        = string
}

variable "workload_job_images" {
  description = <<-EOT
    Workload job image bindings of the intake zone, keyed by the canonical job
    name (dep-intake-fetch). The organization instance supplies the full
    immutable digest of the promoted image from the release-class workload
    image registry; a documented placeholder keeps the binding planned and
    fails provisioning closed until the promotion read-back proof exists.
  EOT
  type        = map(string)
}

variable "audit_sink_name" {
  description = "Name of the audit log sink into the evidence archive."
  type        = string
  default     = "dep-intake-audit-to-evidence"
}

variable "audit_log_filter" {
  description = "Cloud Logging filter for the zone audit export. Defaults to all Cloud Audit Logs of the zone project."
  type        = string
  default     = "logName:\"logs/cloudaudit.googleapis.com\""
}

variable "policy_constraints" {
  description = "Project-level organization-policy compensation for the absent organization node. All constraints default to the enforced posture; the job-owning zones additionally enforce the Cloud Run workload network origin form by default."
  type = object({
    disable_service_account_key_creation  = optional(bool, true)
    disable_service_account_key_upload    = optional(bool, true)
    uniform_bucket_level_access           = optional(bool, true)
    public_access_prevention              = optional(bool, true)
    cloud_run_vpc_egress_all_traffic_only = optional(bool, true)
    cloud_run_ingress_internal_only       = optional(bool, true)
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

variable "enabled_workload_jobs" {
  description = <<-EOT
    The instance-bound activation set of the zone's workload jobs: exactly the
    bound jobs plus the jobs being provisioned in the current window. A
    declared-but-planned job is never engine-active and never enters the plan
    until its provisioning window activates it through this set; the set
    always carries every bound job, because a bound job dropped from the
    active set would plan its own destruction. The organization instance
    supplies this value as reviewed configuration; the core never presets it.
  EOT
  type        = set(string)

  validation {
    condition     = length(setsubtract(var.enabled_workload_jobs, keys(local.workload_jobs))) == 0
    error_message = "enabled_workload_jobs must reference only declared workload jobs of the zone topology."
  }
}

variable "break_glass_recovery" {
  description = <<-EOT
    The approved recovery binding of the intake zone: the project-level role
    the recovery identity receives under the mandatory time-bound IAM
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
