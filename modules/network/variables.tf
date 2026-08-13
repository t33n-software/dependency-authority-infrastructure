variable "project_id" {
  description = "Google Cloud project ID that owns the private DNS zones. The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "zones" {
  description = <<-EOT
    Private DNS zones to create, keyed by the Cloud DNS zone resource name.
    Each zone binds its DNS name (must end with a dot), the VPC network self
    links that may resolve it, and its record sets keyed by record name.
  EOT
  type = map(object({
    dns_name    = string
    description = optional(string, "")
    networks    = set(string)
    records = optional(map(object({
      name    = string
      type    = string
      ttl     = number
      rrdatas = list(string)
    })), {})
  }))

  validation {
    condition     = alltrue([for _, zone in var.zones : endswith(zone.dns_name, ".")])
    error_message = "every zone dns_name must end with a dot."
  }

  validation {
    condition     = alltrue([for _, zone in var.zones : length(zone.networks) > 0])
    error_message = "every zone must bind at least one VPC network self link."
  }
}
