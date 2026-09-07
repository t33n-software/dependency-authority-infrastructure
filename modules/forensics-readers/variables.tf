variable "project_id" {
  description = "Google Cloud project ID of the trust zone receiving the forensics read-only bindings. Supplied by the organization instance; never carried by the core."
  type        = string
}

variable "forensics_group" {
  description = <<-EOT
    Member string of the organization-owned forensics group (the canonical
    identity class dep-forensics-readers), for example
    group:dep-forensics-readers@<organization-domain>. Supplied by the
    organization instance; never carried by the core. The group is created and
    membership-managed on the organization identity plane outside this core;
    this module only binds its read-only access class.
  EOT
  type        = string

  validation {
    condition     = can(regex("^group:dep-forensics-readers@", var.forensics_group))
    error_message = "forensics_group must be the group member string of the canonical dep-forensics-readers identity class."
  }
}

variable "perimeter_ingress" {
  description = <<-EOT
    The second, separate perimeter ingress rule of the forensics reader access
    class: the perimeter resource name and the zone projects the rule covers.
    Declared exactly once by the control-zone stack; every other stack leaves
    it null. All values are supplied by the organization instance.
  EOT
  type = object({
    perimeter_name = string
    zone_projects  = set(string)
  })
  default  = null
  nullable = true

  validation {
    condition = var.perimeter_ingress == null || (
      can(regex("^accessPolicies/[0-9]+/servicePerimeters/[A-Za-z0-9_]+$", var.perimeter_ingress.perimeter_name))
      && length(var.perimeter_ingress.zone_projects) > 0
      && alltrue([for project in var.perimeter_ingress.zone_projects : can(regex("^projects/[0-9]+$", project))])
    )
    error_message = "perimeter_ingress must bind the full perimeter resource name and at least one zone project in the projects/<number> form."
  }
}
