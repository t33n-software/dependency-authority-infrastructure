variable "project_id" {
  description = "Google Cloud project ID of the evidence trust zone (<organization>-dep-evidence). Supplied by the organization instance; the core never carries a default."
  type        = string
}

variable "location" {
  description = "Artifact Registry location of the evidence repositories, Cloud Storage location of the retention archive and the job region of the zone workload network."
  type        = string
}

variable "workload_network" {
  description = <<-EOT
    The zone workload network origin binding: the VPC network name, the
    subnetwork name and the subnetwork CIDR of the zone VPC. The stack
    declares exactly one VPC with one subnetwork in the job region (the stack
    location) carrying Private Google Access, the restricted-range DNS
    response policy and the egress firewall pair; the zone's workload jobs
    attach to it with Direct VPC egress and all-traffic routing. All values
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

variable "additional_writer_members" {
  description = "Members receiving write access on the evidence repositories beyond the writer identity — canonically the intake fetcher (the intake use case writes its candidate records into the evidence repository) and the admission, revalidation, revocation and promotion controllers of the control zone (the promotion writes its approved record into the evidence repository), as bound by the canonical IAM target matrix. Wired by the organization instance; evidence writes stay append-focused and never carry routine delete authority."
  type        = set(string)
  default     = []
}

variable "additional_auditor_members" {
  description = "Additional read-only auditor members of the evidence repositories beyond the auditor identity, as bound by the canonical IAM target matrix. Defaults to none."
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
