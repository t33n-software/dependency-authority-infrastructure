# The behavioral proofs of the dep-quarantine root: the corrected
# state_bucket_name validation, the instance-bound project number binding and
# the recovery binding — the acceptance paths through assertions and the
# rejection paths through expect_failures — every run is a plan with refresh
# disabled, and no run creates infrastructure.
#
# The root carries the gcs backend binding and the dual-fortress encryption
# block, so its initialization resolves the state bucket and the encryption
# key material: this proof executes in the governed execution window, where
# the instance-bound variable channel (a gitignored *.tfvars) supplies
# state_bucket_name and state_encryption_key. Both are never assigned in this
# file, and every value assigned here is synthetic test data.

variables {
  project_id          = "test-dep-quarantine"
  project_number      = "100000000030"
  organization_number = "900000000001"
  location            = "europe-west1"

  writer_members = ["serviceAccount:dep-admission-controller@test-dep-control.iam.gserviceaccount.com"]

  evidence_bucket_name = "test-dep-evidence-archive"

  forensics_group = "group:dep-forensics-readers@example.com"

  break_glass_recovery = {
    max_request_duration = "7200s"
    approvers            = ["group:dep-break-glass-approvers@example.com"]
  }
}

run "accepts_a_valid_state_bucket_name" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "test-dep-quarantine-state"
  }

  assert {
    condition     = var.state_bucket_name == "test-dep-quarantine-state"
    error_message = "The validation must accept a syntactically valid zone state bucket name."
  }
}

run "rejects_a_bucket_name_with_invalid_characters" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "Test-Dep-Quarantine-State"
  }

  expect_failures = [var.state_bucket_name]
}

run "rejects_a_bucket_name_in_ip_form" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "192.168.1.1"
  }

  expect_failures = [var.state_bucket_name]
}

run "rejects_a_bucket_name_with_the_goog_prefix" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "goog-dep-quarantine-state"
  }

  expect_failures = [var.state_bucket_name]
}

run "rejects_a_bucket_name_with_the_google_substring" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "test-google-state"
  }

  expect_failures = [var.state_bucket_name]
}

run "rejects_an_invalid_recovery_duration" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    break_glass_recovery = {
      max_request_duration = "not-a-duration"
      approvers            = ["group:dep-break-glass-approvers@example.com"]
    }
  }

  expect_failures = [var.break_glass_recovery]
}

run "rejects_an_empty_recovery_approver_set" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    break_glass_recovery = {
      max_request_duration = "7200s"
      approvers            = []
    }
  }

  expect_failures = [var.break_glass_recovery]
}

run "accepts_a_numeric_project_number" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.project_number == "100000000030"
    error_message = "The validation must accept the numeric Google Cloud project number of the zone."
  }
}

run "rejects_a_non_numeric_project_number" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    project_number = "test-dep-quarantine"
  }

  expect_failures = [var.project_number]
}

run "accepts_a_numeric_organization_number" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.organization_number == "900000000001"
    error_message = "The validation must accept the numeric Google Cloud organization number."
  }
}

run "rejects_a_non_numeric_organization_number" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    organization_number = "test-dep-quarantine"
  }

  expect_failures = [var.organization_number]
}
