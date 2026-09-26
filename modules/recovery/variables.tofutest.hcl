# The behavioral proofs of the recovery module: the acceptance path through
# assertions and the rejection paths through expect_failures — every run is a
# plan with refresh disabled, and no run creates infrastructure. The module
# carries no backend and no encryption block, so this proof executes offline.
# Every value assigned here is synthetic test data.

variables {
  project_id           = "test-dep-control"
  organization_number  = "900000000001"
  max_request_duration = "7200s"
  approvers            = ["group:dep-break-glass-approvers@example.com"]
}

run "accepts_the_canonical_recovery_binding" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.max_request_duration == "7200s"
    error_message = "The validation must accept the platform's seconds form of the per-activation grant duration."
  }

  assert {
    condition     = var.entitlement_id == "dep-break-glass-recovery-admin"
    error_message = "The entitlement ID must default to the canonical identity class name."
  }

  assert {
    condition     = var.organization_number == "900000000001"
    error_message = "The validation must accept the numeric organization number."
  }
}

run "rejects_a_non_numeric_organization_number" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    organization_number = "test-organization"
  }

  expect_failures = [var.organization_number]
}

run "rejects_an_invalid_entitlement_id" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    entitlement_id = "1dep-break-glass-recovery"
  }

  expect_failures = [var.entitlement_id]
}

run "rejects_an_invalid_request_duration" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    max_request_duration = "not-a-duration"
  }

  expect_failures = [var.max_request_duration]
}

run "rejects_an_empty_approver_set" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    approvers = []
  }

  expect_failures = [var.approvers]
}

run "rejects_a_malformed_approver_principal" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    approvers = ["dep-break-glass-approvers@example.com"]
  }

  expect_failures = [var.approvers]
}
