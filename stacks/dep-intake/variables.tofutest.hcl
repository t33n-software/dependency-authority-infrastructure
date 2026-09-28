# The behavioral proofs of the dep-intake root: the corrected
# state_bucket_name validation, the instance-bound project number binding,
# the workload-job activation gate, the recovery binding, the workload
# network description duty, the fetcher identity display duty and the
# workload job env ownership — the acceptance
# paths through assertions and the rejection paths through expect_failures —
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
  project_id          = "test-dep-intake"
  project_number      = "100000000020"
  organization_number = "900000000001"
  location            = "europe-west1"

  workload_network = {
    network_name               = "dep-intake-workload"
    subnet_name                = "dep-intake-workload-europe-west1"
    subnet_cidr                = "10.20.0.0/26"
    network_description        = "Zone workload network origin."
    subnet_description         = "Zone workload subnetwork."
    firewall_allow_description = "Allow egress TCP 443 to the restricted range."
    firewall_deny_description  = "Deny all remaining egress."
    dns_policy_description     = "Restricted-range DNS form."
  }

  fetcher = {
    provider_id         = "github-intake-fetch"
    service_account_id  = "dep-intake-fetcher"
    display_name        = "dep-intake-fetcher"
    description         = "Execution identity of the intake fetch lane."
    attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-intake-fetch'"
    principal_value     = "example/dependency-authority"
  }

  evidence_bucket_name = "test-dep-evidence-archive"

  workload_job_images = {
    "dep-intake-fetch" = "europe-west1-docker.pkg.dev/test-dep-control/release-controller-images/dependency-intake-controller@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  }

  forensics_group = "group:dep-forensics-readers@example.com"

  enabled_workload_jobs = ["dep-intake-fetch"]

  break_glass_recovery = {
    max_request_duration = "7200s"
    approvers            = ["group:dep-break-glass-approvers@example.com"]
  }

  # The synthetic form of the canonical configuration binding: the static,
  # non-credential environment bindings of the zone's workload job, keyed by
  # the canonical job name. Every value is synthetic test data.
  workload_job_env = {
    "dep-intake-fetch" = {
      DEPENDENCY_AUTHORITY_ZONE      = "intake"
      DEPENDENCY_AUTHORITY_ECOSYSTEM = "go"
    }
  }
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

run "accepts_the_description_surface_bindings" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.workload_network.network_description == "Zone workload network origin."
    error_message = "The workload network origin must bind the canonical description surface."
  }

  assert {
    condition     = var.fetcher.description == "Execution identity of the intake fetch lane."
    error_message = "The fetcher identity must bind its canonical description surface."
  }
}

run "rejects_an_empty_network_description" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    workload_network = {
      network_name               = "dep-intake-workload"
      subnet_name                = "dep-intake-workload-europe-west1"
      subnet_cidr                = "10.20.0.0/26"
      network_description        = ""
      subnet_description         = "Zone workload subnetwork."
      firewall_allow_description = "Allow egress TCP 443 to the restricted range."
      firewall_deny_description  = "Deny all remaining egress."
      dns_policy_description     = "Restricted-range DNS form."
    }
  }

  expect_failures = [var.workload_network]
}

run "rejects_an_identity_without_the_description_surfaces" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    fetcher = {
      provider_id         = "github-intake-fetch"
      service_account_id  = "dep-intake-fetcher"
      display_name        = ""
      description         = "Execution identity of the intake fetch lane."
      attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-intake-fetch'"
      principal_value     = "example/dependency-authority"
    }
  }

  expect_failures = [var.fetcher]
}

run "accepts_the_canonical_workload_job_env_binding" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.workload_job_env["dep-intake-fetch"].DEPENDENCY_AUTHORITY_ZONE == "intake"
    error_message = "The workload job env binding must carry the declared job's static configuration."
  }
}

run "rejects_an_unknown_job_env_binding" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    workload_job_env = {
      "dep-ghost" = {
        DEPENDENCY_AUTHORITY_ZONE = "intake"
      }
    }
  }

  expect_failures = [var.workload_job_env]
}

run "rejects_a_credential_carrying_env_binding" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    workload_job_env = {
      "dep-intake-fetch" = {
        DEPENDENCY_AUTHORITY_TOKEN = "synthetic"
      }
    }
  }

  expect_failures = [var.workload_job_env]
}

run "accepts_a_numeric_project_number" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.project_number == "100000000020"
    error_message = "The validation must accept the numeric Google Cloud project number of the zone."
  }
}

run "rejects_a_non_numeric_project_number" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    project_number = "test-dep-intake"
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
    organization_number = "test-dep-intake"
  }

  expect_failures = [var.organization_number]
}
