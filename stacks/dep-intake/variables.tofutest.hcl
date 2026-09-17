# The behavioral proofs of the dep-intake root: the corrected
# state_bucket_name validation and the workload-job activation gate — the
# acceptance paths through assertions and the rejection paths through
# expect_failures — every run is a plan with refresh disabled, and no run
# creates infrastructure.
#
# The root carries the gcs backend binding and the dual-fortress encryption
# block, so its initialization resolves the state bucket and the encryption
# key material: this proof executes in the governed execution window, where
# the instance-bound variable channel (a gitignored *.tfvars) supplies
# state_bucket_name and state_encryption_key. Both are never assigned in this
# file, and every value assigned here is synthetic test data.

variables {
  project_id = "test-dep-intake"
  location   = "europe-west1"

  workload_network = {
    network_name = "dep-intake-workload"
    subnet_name  = "dep-intake-workload-europe-west1"
    subnet_cidr  = "10.20.0.0/26"
  }

  fetcher = {
    provider_id         = "github-intake-fetch"
    service_account_id  = "dep-intake-fetcher"
    attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-intake-fetch'"
    principal_value     = "example/dependency-authority"
  }

  evidence_bucket_name = "test-dep-evidence-archive"

  workload_job_images = {
    "dep-intake-fetch" = "europe-west1-docker.pkg.dev/test-dep-control/release-controller-images/dependency-intake-controller@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  }

  forensics_group = "group:dep-forensics-readers@example.com"

  enabled_workload_jobs = ["dep-intake-fetch"]
}

run "accepts_a_valid_state_bucket_name" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "test-dep-intake-state"
  }

  assert {
    condition     = var.state_bucket_name == "test-dep-intake-state"
    error_message = "The validation must accept a syntactically valid zone state bucket name."
  }
}

run "rejects_a_bucket_name_with_invalid_characters" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "Test-Dep-Intake-State"
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
    state_bucket_name = "goog-dep-intake-state"
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

run "accepts_the_activation_set" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = contains(var.enabled_workload_jobs, "dep-intake-fetch")
    error_message = "The activation set must carry the declared jobs of the zone topology."
  }
}

run "rejects_an_unknown_enabled_job" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    enabled_workload_jobs = ["dep-intake-fetch", "dep-ghost"]
  }

  expect_failures = [var.enabled_workload_jobs]
}
