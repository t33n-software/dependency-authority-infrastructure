variable "project_number" {
  description = "Google Cloud project number of the trust zone receiving the policy constraints: every organization policy is a number-addressed resource (its name and parent carry the projects/<number> form), so the module binds the numeric project number and never the project ID — binding the project ID would force a destroy-and-recreate of the live policies at the convergence window. The organization instance supplies this value; the core never presets it."
  type        = string

  validation {
    condition     = can(regex("^[0-9]+$", var.project_number))
    error_message = "project_number must be the numeric Google Cloud project number of the trust zone."
  }
}

variable "disable_service_account_key_creation" {
  description = "Enforce iam.disableServiceAccountKeyCreation on the project. Static service-account keys are never a dependency authority credential mechanism."
  type        = bool
  default     = true
}

variable "disable_service_account_key_upload" {
  description = "Enforce iam.disableServiceAccountKeyUpload on the project. Externally generated key material is never uploaded into the trust zones."
  type        = bool
  default     = true
}

variable "uniform_bucket_level_access" {
  description = "Enforce storage.uniformBucketLevelAccess on the project. Object-level ACLs never weaken the evidence archive boundary."
  type        = bool
  default     = true
}

variable "public_access_prevention" {
  description = "Enforce storage.publicAccessPrevention on the project. Evidence and archive buckets are never public."
  type        = bool
  default     = true
}

variable "cloud_run_vpc_egress_all_traffic_only" {
  description = "Restrict the deployable Cloud Run VPC egress settings on the project to all-traffic only (constraints/run.allowedVPCEgress). The workload network origin form — every job routes all outgoing traffic through its zone VPC — becomes the only deployable form, so the platform enforces it rather than convention alone. Opt-in: only the job-owning zones enable it."
  type        = bool
  default     = false
}

variable "cloud_run_ingress_internal_only" {
  description = "Restrict the deployable Cloud Run ingress settings on the project to internal only (constraints/run.allowedIngress). Workload jobs never receive external ingress; the workload ingress prohibition of the perimeter holds absolutely. Opt-in: only the job-owning zones enable it."
  type        = bool
  default     = false
}
