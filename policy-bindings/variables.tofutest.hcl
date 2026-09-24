# The behavioral proofs of the policy-bindings module: the instance-bound
# project number validation and the number form of the organization policies —
# the acceptance run proves the planned policy name carries the numeric project
# number, and the rejection run proves the non-numeric form fails closed. The
# module carries no backend and no encryption block, so this proof executes
# offline; every value assigned here is synthetic test data.

variables {
  project_number = "100000000060"
}

run "accepts_a_numeric_project_number" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = google_org_policy_policy.this["iam.disableServiceAccountKeyCreation"].name == "projects/100000000060/policies/iam.disableServiceAccountKeyCreation"
    error_message = "The organization policy must bind the instance-bound numeric project number in its name."
  }
}

run "rejects_a_non_numeric_project_number" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    project_number = "test-dep-control"
  }

  expect_failures = [var.project_number]
}
