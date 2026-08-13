variable "project_id" {
  description = "Google Cloud project ID whose logs are exported. The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "destination_bucket" {
  description = "Name of the Cloud Storage evidence archive bucket that receives the exported logs. Created by the evidence trust zone; other zones receive it as an instance-supplied wiring value."
  type        = string
}

variable "sinks" {
  description = <<-EOT
    Log sinks to create, keyed by sink name. Each sink carries its Cloud
    Logging filter and optional exclusions. Every sink uses a unique writer
    identity, which this module binds to the archive bucket.
  EOT
  type = map(object({
    filter      = string
    description = optional(string, "")
    exclusions = optional(list(object({
      name        = string
      filter      = string
      description = optional(string, "")
      disabled    = optional(bool, false)
    })), [])
  }))

  validation {
    condition     = length(var.sinks) > 0
    error_message = "at least one sink is required."
  }
}

variable "writer_role" {
  description = "Bucket role granted to each sink writer identity. Defaults to append-focused object creation."
  type        = string
  default     = "roles/storage.objectCreator"
}
