variable "project_id" {
  description = "Google Cloud project ID of the trust zone that owns the state home. The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "name" {
  description = "Globally unique Cloud Storage bucket name of the zone state home. The organization instance binds it from the naming grammar family <organization>-<boundary>-<purpose>; the core never assigns one."
  type        = string

  validation {
    condition = (
      can(regex("^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$", var.name))
      && !can(regex("^[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+$", var.name))
      && !startswith(var.name, "goog")
      && !contains(var.name, "google")
    )
    error_message = "name must satisfy the Cloud Storage bucket naming rules: 3-63 characters of lowercase letters, digits, hyphens, underscores and dots, alphanumeric edges, never an IP form, never the goog prefix and never google or similar spellings."
  }
}

variable "location" {
  description = "Cloud Storage location of the state bucket. The location is coupled to the location of the bucket CMEK key ring (a hard platform rule); the consuming stack proves the coupling fail-closed."
  type        = string
}

variable "cmek_key_name" {
  description = <<-EOT
    Resource name of the customer-managed encryption key (CMEK) that
    encrypts every object of the state bucket — the provider layer of the
    dual fortress state-encryption standard. This key is the second,
    cryptographically separate key: it is never the engine key of the
    client-side state encryption. The organization instance binds the
    concrete reference; the input is mandatory, because the state home
    never exists without the provider layer.
  EOT
  type        = string

  validation {
    condition     = can(regex("^projects/[^/]+/locations/[^/]+/keyRings/[^/]+/cryptoKeys/[^/]+$", var.cmek_key_name))
    error_message = "cmek_key_name must be a full GCP KMS key resource name (projects/*/locations/*/keyRings/*/cryptoKeys/*)."
  }
}

variable "operator_members" {
  description = <<-EOT
    Fully formed member strings receiving exactly roles/storage.objectAdmin
    on the state bucket — the documented backend credential requirement of
    the engine. The organization instance binds the operator execution
    identities of the zone; the module never constructs members and never
    grants any other role.
  EOT
  type        = set(string)
  default     = []
}

variable "labels" {
  description = "Bucket labels. The reference stack binds boundary and zone labels."
  type        = map(string)
  default     = {}
}
