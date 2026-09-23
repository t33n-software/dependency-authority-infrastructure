variable "project_id" {
  description = "Google Cloud project ID of the trust zone owning the pool. The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "project_number" {
  description = "Google Cloud project number of the trust zone owning the pool: the pool binds it because the provider state carries the pool's project as the numeric project number (the import and read-back form projects/<number>/locations/global/workloadIdentityPools/<pool>), and binding the project ID would force a destroy-and-recreate of the pool. Every other resource of the module keeps the project ID. The organization instance supplies this value; the core never presets it."
  type        = string

  validation {
    condition     = can(regex("^[0-9]+$", var.project_number))
    error_message = "project_number must be the numeric Google Cloud project number of the trust zone."
  }
}

variable "pool_id" {
  description = "Workload Identity Pool ID of the trust zone. One pool per zone; identities of other zones never share it."
  type        = string
}

variable "pool_display_name" {
  description = "Display name of the Workload Identity Pool."
  type        = string
  default     = ""
}

variable "pool_description" {
  description = "Description of the Workload Identity Pool."
  type        = string
  default     = ""
}

variable "identities" {
  description = <<-EOT
    Workload identities of the trust zone, keyed by their lane role. Each
    identity binds one OIDC provider, one execution service account, one
    invoke-only trigger service account and one principal set: the principal
    set federates to the trigger identity, never to the execution identity,
    and the trigger identity never carries a role. The instance binds the
    exact repository, protected workflow reference, environment and audience
    through attribute_condition and principal_value. With an empty map only
    the zone pool is created.
  EOT
  type = map(object({
    provider_id                = string
    service_account_id         = string
    trigger_service_account_id = string
    display_name               = optional(string, "")
    description                = optional(string, "")
    issuer_uri                 = optional(string, "https://token.actions.githubusercontent.com")
    allowed_audiences          = optional(list(string), [])
    attribute_mapping          = optional(map(string))
    attribute_condition        = string
    principal_attribute        = optional(string, "repository")
    principal_value            = string
    roles                      = optional(set(string), [])
  }))

  validation {
    condition = alltrue([
      for _, identity in var.identities :
      identity.attribute_mapping == null || contains(keys(identity.attribute_mapping), "google.subject")
    ])
    error_message = "a custom attribute_mapping must include google.subject."
  }

  validation {
    condition = alltrue([
      for _, identity in var.identities :
      length(identity.attribute_condition) > 0 && length(identity.principal_value) > 0
    ])
    error_message = "every identity must bind an attribute_condition and a principal_value."
  }

  validation {
    condition = alltrue([
      for _, identity in var.identities :
      identity.trigger_service_account_id != identity.service_account_id
      && can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", identity.trigger_service_account_id))
    ])
    error_message = "every identity must bind a dedicated invoke-only trigger service account that is distinct from the execution identity."
  }
}
