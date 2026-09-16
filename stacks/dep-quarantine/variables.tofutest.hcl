# The behavioral proof of the corrected state_bucket_name validation of the
# dep-quarantine root: the acceptance path through an assertion and the
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
  project_id = "test-dep-quarantine"
  location   = "europe-west1"

  writer_members = ["serviceAccount:dep-admission-controller@test-dep-control.iam.gserviceaccount.com"]

  evidence_bucket_name = "test-dep-evidence-archive"

  forensics_group = "group:dep-forensics-readers@example.com"
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
