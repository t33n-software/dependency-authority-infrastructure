variable "project_id" {
  description = "Google Cloud project ID of the control trust zone (<organization>-dep-control). Supplied by the organization instance; the core never carries a default."
  type        = string
}

variable "project_number" {
  description = "Google Cloud project number of the control trust zone: the number-addressed classes bind it — the workload identity pool and its providers (the provider state carries them under the numeric project number; the ID form would force their destroy-and-recreate) and the organization policies (their name and parent are number-addressed); every other surface keeps the project ID. Supplied by the organization instance; the core never presets it."
  type        = string

  validation {
    condition     = can(regex("^[0-9]+$", var.project_number))
    error_message = "project_number must be the numeric Google Cloud project number of the zone."
  }
}

variable "location" {
  description = "Artifact Registry location of the control-zone workload image registries and the job region of the zone workload network."
  type        = string
}

variable "workload_network" {
  description = <<-EOT
    The zone workload network origin binding: the VPC network name, the
    subnetwork name, the subnetwork CIDR and the canonical description of
    every managed surface of the zone VPC. The stack declares exactly one VPC
    with one subnetwork in the job region (the stack location) carrying
    Private Google Access, the restricted-range DNS response policy and the
    egress firewall pair; the zone's workload jobs attach to it with Direct
    VPC egress and all-traffic routing. The mandatory description duty of the
    mandatory resource properties convention binds the description surfaces:
    the VPC and subnetwork descriptions are create-only surfaces, bound
    byte-exact to the live values at the convergence window; the egress
    firewall pair and the restricted-range DNS response policy descriptions
    are in-place surfaces. All values are instance-supplied.
  EOT
  type = object({
    network_name               = string
    subnet_name                = string
    subnet_cidr                = string
    network_description        = string
    subnet_description         = string
    firewall_allow_description = string
    firewall_deny_description  = string
    dns_policy_description     = string
  })

  validation {
    condition     = can(cidrhost(var.workload_network.subnet_cidr, 0))
    error_message = "workload_network.subnet_cidr must be a valid CIDR range."
  }

  validation {
    condition = (
      length(var.workload_network.network_description) > 0 &&
      length(var.workload_network.subnet_description) > 0 &&
      length(var.workload_network.firewall_allow_description) > 0 &&
      length(var.workload_network.firewall_deny_description) > 0 &&
      length(var.workload_network.dns_policy_description) > 0
    )
    error_message = "workload_network must bind the non-empty canonical description of every managed network surface: the create-only VPC and subnetwork descriptions byte-exact to the live values, and the in-place firewall pair and DNS policy descriptions."
  }
}

variable "pool_id" {
  description = "Workload Identity Pool ID of the control zone."
  type        = string
  default     = "dep-control"
}

variable "controllers" {
  description = <<-EOT
    Control-plane controller workload identities, keyed by lane (canonically
    admission, promotion, revalidation, revocation and consumer-verification —
    the workload job topology references exactly these keys). The organization instance binds
    the exact repository, protected workflow reference, environment and
    audience     through attribute_condition and principal_value, and assigns the
    canonical identity class names through service_account_id (for example
    dep-admission-controller). Every controller identity binds its canonical
    display name and description surfaces (the mandatory description duty of
    the mandatory resource properties convention) as instance-bound values.
  EOT
  type = map(object({
    provider_id         = string
    service_account_id  = string
    display_name        = string
    description         = string
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

  validation {
    condition = alltrue([
      for _, controller in var.controllers :
      length(controller.display_name) > 0 && length(controller.description) > 0
    ])
    error_message = "every controller identity must bind its non-empty canonical display name and description surfaces (the mandatory description duty of the mandatory resource properties convention)."
  }
}

variable "evidence_bucket_name" {
  description = "Name of the evidence archive bucket provisioned by the dep-evidence stack. The control zone exports its audit logs there; provisioning order is dep-evidence first."
  type        = string
}

variable "workload_job_images" {
  description = <<-EOT
    Workload job image bindings of the control zone, keyed by the canonical
    job name (dep-admission, dep-promotion, dep-revalidation, dep-revocation,
    dep-consumer-verification). The organization instance supplies the full immutable digest of the
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

variable "perimeter_ingress" {
  description = <<-EOT
    The second, separate perimeter ingress rule of the forensics reader access
    class: the perimeter resource name and the five zone projects the rule
    covers. The control zone declares this boundary-level binding exactly once
    for the whole perimeter; every other stack never carries it. All values
    are supplied by the organization instance.
  EOT
  type = object({
    perimeter_name = string
    zone_projects  = set(string)
  })

  validation {
    condition = (
      can(regex("^accessPolicies/[0-9]+/servicePerimeters/[A-Za-z0-9_]+$", var.perimeter_ingress.perimeter_name))
      && length(var.perimeter_ingress.zone_projects) > 0
      && alltrue([for project in var.perimeter_ingress.zone_projects : can(regex("^projects/[0-9]+$", project))])
    )
    error_message = "perimeter_ingress must bind the full perimeter resource name and at least one zone project in the projects/<number> form."
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
    The approved recovery binding of the control zone: the project-level role
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

variable "workload_image_cleanup" {
  description = <<-EOT
    The instance-bound workload image lifecycle binding of the control zone:
    the declared, platform-executed cleanup policies of the two workload image
    registries and their dry-run activation state, keyed by the canonical
    class. The organization instance binds the canonical convention values —
    the staging class deletes image versions older than 30 days and always
    keeps the most recent 2 per package; the release class always keeps the
    most recent 5 per package and never carries a time-based deletion — with
    binding status planned until the list-cleanup-policies read-back proof
    flips them to bound. The dry-run state starts true (the fail-safe
    posture) and flips to false only through the governed activation window
    after the dry-run proof. The core never presets it.
  EOT
  type = object({
    staging = object({
      dry_run = bool
      policies = map(object({
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
    })
    release = object({
      dry_run = bool
      policies = map(object({
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
    })
  })

  validation {
    condition     = alltrue([for _, policy in var.workload_image_cleanup.release.policies : policy.action == "KEEP"])
    error_message = "the release class carries a depth retention only: keep policies, never a delete policy and never a time-based expiry."
  }

  validation {
    condition     = anytrue([for _, policy in var.workload_image_cleanup.staging.policies : policy.action == "DELETE" && policy.condition != null && policy.condition.older_than != null])
    error_message = "the staging class carries the short transit retention: a time-based delete policy plus the keep floor."
  }

  validation {
    condition     = anytrue([for _, policy in var.workload_image_cleanup.staging.policies : policy.action == "KEEP" && policy.most_recent_versions != null && policy.most_recent_versions.keep_count != null])
    error_message = "the staging class carries the keep floor: a most-recent-versions keep policy."
  }
}

variable "workload_job_env" {
  description = <<-EOT
    The instance-bound static environment bindings of the zone's workload
    jobs, keyed by the canonical job name. The declaration owns every static,
    non-credential configuration value of a workload completely (the workload
    configuration ownership convention): the organization instance binds the
    proven live values as reviewed configuration, and every value referencing
    another bound surface is a proven projection the instance verifier
    cross-binds fail-closed against its canonical source. Operation inputs
    travel as validated execution parameters of the invocation, never as
    baked-in values, and credentials never travel this surface. The core
    never presets it.
  EOT
  type        = map(map(string))

  validation {
    condition     = length(setsubtract(keys(var.workload_job_env), keys(local.workload_jobs))) == 0
    error_message = "workload_job_env must reference only declared workload jobs of the zone topology."
  }

  validation {
    condition = alltrue([
      for _, bindings in var.workload_job_env :
      alltrue([for key in keys(bindings) : can(regex("^[A-Z][A-Z0-9_]*$", key))])
    ])
    error_message = "workload_job_env must carry only well-formed environment variable names (UPPER_SNAKE_CASE)."
  }

  validation {
    condition = alltrue([
      for _, bindings in var.workload_job_env :
      alltrue([
        for key, value in bindings :
        !can(regex("(?i)(password|secret|token|credential|api[_-]?key|private[_-]?key)", key))
        && !can(regex("(?i)(password|secret|token|credential|api[_-]?key|private[_-]?key)", value))
      ])
    ])
    error_message = "workload_job_env never carries credentials: no key and no value may carry a credential marker."
  }
}
