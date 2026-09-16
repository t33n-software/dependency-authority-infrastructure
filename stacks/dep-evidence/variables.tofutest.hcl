# The behavioral proof of the corrected state_bucket_name validation of the
# dep-evidence root: the acceptance path through an assertion and the
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
  project_id = "test-dep-evidence"
  location   = "europe-west1"

  workload_network = {
    network_name = "dep-evidence-workload"
    subnet_name  = "dep-evidence-workload-europe-west1"
    subnet_cidr  = "10.30.0.0/26"
  }

  writer = {
    provider_id         = "github-evidence-write"
    service_account_id  = "dep-evidence-writer"
    attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-evidence-write'"
    principal_value     = "example/dependency-authority"
  }

  auditor = {
    provider_id         = "github-evidence-audit"
    service_account_id  = "dep-evidence-auditor"
    attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-evidence-audit'"
    principal_value     = "example/dependency-authority"
  }

  archive_bucket_name      = "test-dep-evidence-archive"
  retention_period_seconds = 7776000

  workload_job_images = {
    "dep-evidence-write" = "europe-west1-docker.pkg.dev/test-dep-control/release-controller-images/dependency-evidence-write-controller@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    "dep-evidence-audit" = "europe-west1-docker.pkg.dev/test-dep-control/release-controller-images/dependency-evidence-audit-controller@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  }

  forensics_group = "group:dep-forensics-readers@example.com"
}

run "accepts_a_valid_state_bucket_name" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "test-dep-evidence-state"
  }

  assert {
    condition     = var.state_bucket_name == "test-dep-evidence-state"
    error_message = "The validation must accept a syntactically valid zone state bucket name."
  }
}

run "rejects_a_bucket_name_with_invalid_characters" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "Test-Dep-Evidence-State"
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
    state_bucket_name = "goog-dep-evidence-state"
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
