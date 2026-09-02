variable "project_id" {
  description = "Google Cloud project ID of the trust zone that owns the job (<organization>-dep-<zone>). The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "location" {
  description = "Google Cloud location of the job. The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "name" {
  description = "Canonical workload job name. Every lane operation owns exactly one job named dep-<operation> in its own zone."

  type = string

  validation {
    condition     = can(regex("^dep-[a-z0-9]+(-[a-z0-9]+)*$", var.name))
    error_message = "name must follow the canonical workload job form dep-<operation>."
  }
}

variable "service_account_email" {
  description = "Email of the existing zone workload identity that executes the job. The job never runs as a lane identity and never receives a write grant on a workload image registry."

  type = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]*@[a-z][a-z0-9-]*\\.iam\\.gserviceaccount\\.com$", var.service_account_email))
    error_message = "service_account_email must be a Google Cloud service account email of the zone workload identity."
  }
}

variable "invoker_member" {
  description = <<-EOT
    The invoke-only trigger identity of the owning lane operation as an IAM
    member string. The module binds it invoke-only on exactly this job; the
    trigger identity never holds a data-plane permission, and the execution
    identity of the job is never federated from CI.
  EOT

  type = string

  validation {
    condition     = can(regex("^serviceAccount:[a-z][a-z0-9-]*@[a-z][a-z0-9-]*\\.iam\\.gserviceaccount\\.com$", var.invoker_member))
    error_message = "invoker_member must be the serviceAccount member of the lane's dedicated invoke-only trigger identity."
  }
}

variable "image" {
  description = <<-EOT
    Workload image reference of the job. The organization instance binds the
    full immutable digest of the promoted image from the release-class
    workload image registry; the binding stays planned (documented
    placeholder) until the promotion read-back proof exists, and a
    placeholder fails provisioning closed.
  EOT

  type = string

  validation {
    condition     = can(regex("^[^@\\s]+@sha256:[0-9a-f]{64}$", var.image))
    error_message = "image must be a full immutable digest reference (<registry>/<image>@sha256:<64 lowercase hex>); tags and mutable references are forbidden."
  }

  validation {
    condition     = can(regex("/release-[^/]+/", var.image))
    error_message = "image must resolve from a release-class workload image registry; the staging class and the dependency repositories are never workload sources."
  }
}

variable "env" {
  description = "Static environment bindings of the job container. Operation inputs travel as validated execution parameters of the invocation, never as baked-in values; sensitive values are forbidden here."
  type        = map(string)
  default     = {}
}

variable "labels" {
  description = "Job labels. The reference stacks bind the boundary and zone labels."
  type        = map(string)
  default     = {}
}
