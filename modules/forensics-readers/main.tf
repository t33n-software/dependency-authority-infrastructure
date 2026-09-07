# The forensics reader access class: the organization-owned forensics group
# holds exactly the two read-only diagnostic roles on each zone project —
# project-scoped, never organization-scoped — and no other grant on any plane.
# The read-only form is enforced by the granted roles, never by convention.

resource "google_project_iam_member" "log_reader" {
  project = var.project_id
  role    = "roles/logging.viewer"
  member  = var.forensics_group
}

resource "google_project_iam_member" "execution_reader" {
  project = var.project_id
  role    = "roles/run.viewer"
  member  = var.forensics_group
}

# The second, separate perimeter ingress rule: the forensics group is the only
# identity, scoped to the read-only logging method logging.logEntries.list with
# the zone projects as resources, entering through the same identity-bound
# channel as the administration rule. The administration ingress rule never
# carries the forensics identity, and every additional read method is a
# governed change to this rule. The execution status read-back travels the
# deliberately non-restricted compute control plane and needs no perimeter
# rule. Declared exactly once, where the instance binds the perimeter (the
# control-zone stack); every other stack leaves it undeclared.
resource "google_access_context_manager_service_perimeter_ingress_policy" "forensics" {
  count = var.perimeter_ingress == null ? 0 : 1

  perimeter = var.perimeter_ingress.perimeter_name
  title     = "forensics-read-only-log-access"

  ingress_from {
    identities = [var.forensics_group]

    sources {
      access_level = "*"
    }
  }

  ingress_to {
    resources = tolist(var.perimeter_ingress.zone_projects)

    operations {
      service_name = "logging.googleapis.com"

      method_selectors {
        permission = "logging.logEntries.list"
      }
    }
  }
}
