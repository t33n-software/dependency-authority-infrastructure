# The behavioral proofs of the dep-evidence root: the corrected
# state_bucket_name validation, the instance-bound project number binding,
# the workload-job activation gate, the recovery binding and the workload
# network description duty — the acceptance
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
  project_id            = "test-dep-evidence"
  project_number        = "100000000050"
  organization_number   = "900000000001"
  folder_number         = "400000000001"
  anchor_project_number = "100000000060"
  location              = "europe-west1"

  workload_network = {
    network_name               = "dep-evidence-workload"
    subnet_name                = "dep-evidence-workload-europe-west1"
    subnet_cidr                = "10.30.0.0/26"
    network_description        = "Zone workload network origin."
    subnet_description         = "Zone workload subnetwork."
    firewall_allow_description = "Allow egress TCP 443 to the restricted range."
    firewall_deny_description  = "Deny all remaining egress."
    dns_policy_description     = "Restricted-range DNS form."
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

  enabled_workload_jobs = ["dep-evidence-write", "dep-evidence-audit"]

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

run "accepts_the_activation_set" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = contains(var.enabled_workload_jobs, "dep-evidence-audit")
    error_message = "The activation set must carry the declared jobs of the zone topology."
  }
}

run "rejects_an_unknown_enabled_job" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    enabled_workload_jobs = ["dep-evidence-write", "dep-ghost"]
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
}

run "rejects_an_empty_network_description" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    workload_network = {
      network_name               = "dep-evidence-workload"
      subnet_name                = "dep-evidence-workload-europe-west1"
      subnet_cidr                = "10.30.0.0/26"
      network_description        = ""
      subnet_description         = "Zone workload subnetwork."
      firewall_allow_description = "Allow egress TCP 443 to the restricted range."
      firewall_deny_description  = "Deny all remaining egress."
      dns_policy_description     = "Restricted-range DNS form."
    }
  }

  expect_failures = [var.workload_network]
}

run "accepts_a_numeric_project_number" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.project_number == "100000000050"
    error_message = "The validation must accept the numeric Google Cloud project number of the zone."
  }
}

run "rejects_a_non_numeric_project_number" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    project_number = "test-dep-evidence"
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
    organization_number = "test-dep-evidence"
  }

  expect_failures = [var.organization_number]
}

run "accepts_a_numeric_folder_number" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.folder_number == "400000000001"
    error_message = "The validation must accept the numeric Google Cloud folder ID of the folder grouping layer."
  }
}

run "rejects_a_non_numeric_folder_number" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    folder_number = "dependency-authority"
  }

  expect_failures = [var.folder_number]
}

run "accepts_a_numeric_anchor_project_number" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.anchor_project_number == "100000000060"
    error_message = "The validation must accept the numeric Google Cloud project number of the organization anchor project."
  }
}

run "rejects_a_non_numeric_anchor_project_number" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    anchor_project_number = "t33n-software-org-anchor"
  }

  expect_failures = [var.anchor_project_number]
}

run "accepts_the_audit_export_writer_grammar" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = contains(module.evidence_archive.logging_export_writer_members, "serviceAccount:service-org-900000000001@gcp-sa-logging.iam.gserviceaccount.com")
    error_message = "The archive must grant the standing append-focused write capability to the organization node's Cloud Logging service agent."
  }

  assert {
    condition     = contains(module.evidence_archive.logging_export_writer_members, "serviceAccount:service-folder-400000000001@gcp-sa-logging.iam.gserviceaccount.com")
    error_message = "The archive must grant the standing append-focused write capability to the folder grouping layer's Cloud Logging service agent."
  }

  assert {
    condition     = contains(module.evidence_archive.logging_export_writer_members, "serviceAccount:service-100000000060@gcp-sa-logging.iam.gserviceaccount.com")
    error_message = "The archive must grant the standing append-focused write capability to the organization anchor project's unique Cloud Logging writer identity."
  }
}
