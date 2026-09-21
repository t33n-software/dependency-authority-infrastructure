# The behavioral proofs of the dep-approved root: the corrected
# state_bucket_name validation and the recovery binding — the acceptance path
# through an assertion and the rejection paths through expect_failures —
# every run is a plan with refresh disabled, and no run creates
# infrastructure.
#
# The root carries the gcs backend binding and the dual-fortress encryption
# block, so its initialization resolves the state bucket and the encryption
# key material: this proof executes in the governed execution window, where
# the instance-bound variable channel (a gitignored *.tfvars) supplies
# state_bucket_name and state_encryption_key. Both are never assigned in this
# file, and every value assigned here is synthetic test data.

variables {
  project_id = "test-dep-approved"
  location   = "europe-west1"

  promoter_member            = "serviceAccount:dep-approved-promoter@test-dep-control.iam.gserviceaccount.com"
  revocation_member          = "serviceAccount:dep-revocation-controller@test-dep-control.iam.gserviceaccount.com"
  revalidation_reader_member = "serviceAccount:dep-revalidation-controller@test-dep-control.iam.gserviceaccount.com"

  evidence_bucket_name = "test-dep-evidence-archive"

  forensics_group = "group:dep-forensics-readers@example.com"

  break_glass_recovery = {
    role               = "roles/resourcemanager.projectIamAdmin"
    condition_end_time = "2027-01-01T00:00:00Z"
  }
}

run "accepts_a_valid_state_bucket_name" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "test-dep-approved-state"
  }

  assert {
    condition     = var.state_bucket_name == "test-dep-approved-state"
    error_message = "The validation must accept a syntactically valid zone state bucket name."
  }
}

run "rejects_a_bucket_name_with_invalid_characters" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "Test-Dep-Approved-State"
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
    state_bucket_name = "goog-dep-approved-state"
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

run "rejects_an_invalid_recovery_end_time" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    break_glass_recovery = {
      role               = "roles/resourcemanager.projectIamAdmin"
      condition_end_time = "not-a-timestamp"
    }
  }

  expect_failures = [var.break_glass_recovery]
}
