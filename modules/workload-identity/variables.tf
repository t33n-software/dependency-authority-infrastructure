variable "project_id" {
  description = "Google Cloud project ID of the trust zone owning the pool. The organization instance supplies this value; the core never carries a default."
  type        = string
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
    identity binds one OIDC provider, one service account and one principal
    set: the instance binds the exact repository, protected workflow
    reference, environment and audience through attribute_condition and
    principal_value. With an empty map only the zone pool is created.
  EOT
  type = map(object({
    provider_id         = string
    service_account_id  = string
    display_name        = optional(string, "")
    description         = optional(string, "")
    issuer_uri          = optional(string, "https://token.actions.githubusercontent.com")
    allowed_audiences   = optional(list(string), [])
    attribute_mapping   = optional(map(string))
    attribute_condition = string
    principal_attribute = optional(string, "repository")
    principal_value     = string
    roles               = optional(set(string), [])
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
}
