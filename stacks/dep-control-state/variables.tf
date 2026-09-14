variable "project_id" {
  description = "Google Cloud project ID of the control trust zone (<organization>-dep-control). Supplied by the organization instance; the core never carries a default."
  type        = string
}

variable "location" {
  description = "Cloud Storage location of the control-zone state bucket. Coupled to the location of the bucket CMEK key ring; the coupling is proven fail-closed against the bound key reference."
  type        = string
}

variable "state_bucket_name" {
  description = "Globally unique Cloud Storage bucket name of the control-zone state home, bound by the organization instance from the naming grammar family <organization>-<boundary>-<purpose>. The stack never assigns one."
  type        = string

  validation {
    condition = (
      can(regex("^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$", var.state_bucket_name))
      && !can(regex("^[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+$", var.state_bucket_name))
      && !startswith(var.state_bucket_name, "goog")
      && !contains(var.state_bucket_name, "google")
    )
    error_message = "state_bucket_name must satisfy the Cloud Storage bucket naming rules: 3-63 characters of lowercase letters, digits, hyphens, underscores and dots, alphanumeric edges, never an IP form, never the goog prefix and never google or similar spellings."
  }
}

variable "state_encryption_key" {
  description = "Instance-bound GCP KMS key reference of the client-side state and plan encryption engine layer (projects/*/locations/*/keyRings/*/cryptoKeys/*). The value is mandatory and instance-bound."
  type        = string

  validation {
    condition     = can(regex("^projects/[^/]+/locations/[^/]+/keyRings/[^/]+/cryptoKeys/[^/]+$", var.state_encryption_key))
    error_message = "state_encryption_key must be a full GCP KMS key resource name."
  }
}

variable "state_bucket_cmek_key" {
  description = "Instance-bound GCP KMS key reference that encrypts every object of the state bucket as its CMEK — the provider layer of the dual fortress form. Cryptographically separate from the engine key; the value is mandatory and instance-bound."
  type        = string

  validation {
    condition     = can(regex("^projects/[^/]+/locations/[^/]+/keyRings/[^/]+/cryptoKeys/[^/]+$", var.state_bucket_cmek_key))
    error_message = "state_bucket_cmek_key must be a full GCP KMS key resource name."
  }

  validation {
    condition     = var.state_bucket_cmek_key != var.state_encryption_key
    error_message = "state_bucket_cmek_key must be cryptographically separate from state_encryption_key; the engine key and the bucket CMEK key are disjoint boundaries."
  }

  validation {
    condition     = length(split("/", var.state_bucket_cmek_key)) == 6 && element(split("/", var.state_bucket_cmek_key), 3) == var.location
    error_message = "state_bucket_cmek_key must reside in the same location as the state bucket; the CMEK location coupling is a hard platform rule."
  }
}

variable "operator_members" {
  description = "Fully formed member strings of the operator execution identities receiving exactly roles/storage.objectAdmin on the state bucket (the documented backend credential requirement). Bound by the organization instance."
  type        = set(string)
  default     = []
}
