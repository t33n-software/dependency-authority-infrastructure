variable "project_id" {
  description = "Google Cloud project ID of the intake trust zone (<organization>-dep-intake). Supplied by the organization instance; the core never carries a default."
  type        = string
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
  description = "Members receiving read access on the intake repositories beyond the fetcher writer — canonically the admission controller and the approved promoter identities of the control zone, as bound by the canonical IAM target matrix. Wired by the organization instance; the fetcher and any consumer identity never appear here."
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
