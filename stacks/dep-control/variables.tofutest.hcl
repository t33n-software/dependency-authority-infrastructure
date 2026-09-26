# The behavioral proofs of the dep-control root: the corrected
# state_bucket_name validation, the instance-bound project number binding,
# the workload-job activation gate, the recovery binding and the workload
# image cleanup lifecycle binding — the acceptance paths through assertions
# and the rejection paths through expect_failures — every run is a plan with
# refresh disabled, and no run creates infrastructure.
#
# The root carries the gcs backend binding and the dual-fortress encryption
# block, so its initialization resolves the state bucket and the encryption
# key material: this proof executes in the governed execution window, where
# the instance-bound variable channel (a gitignored *.tfvars) supplies
# state_bucket_name and state_encryption_key. Both are never assigned in this
# file, and every value assigned here is synthetic test data.

variables {
  project_id          = "test-dep-control"
  project_number      = "100000000010"
  organization_number = "900000000001"
  location            = "europe-west1"

  workload_network = {
    network_name               = "dep-control-workload"
    subnet_name                = "dep-control-workload-europe-west1"
    subnet_cidr                = "10.10.0.0/26"
    network_description        = "Zone workload network origin."
    subnet_description         = "Zone workload subnetwork."
    firewall_allow_description = "Allow egress TCP 443 to the restricted range."
    firewall_deny_description  = "Deny all remaining egress."
    dns_policy_description     = "Restricted-range DNS form."
  }

  controllers = {
    admission = {
      provider_id         = "github-admission"
      service_account_id  = "dep-admission-controller"
      display_name        = "dep-admission-controller"
      description         = "Execution identity of the admission lane."
      attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-admission'"
      principal_value     = "example/dependency-authority"
    }
    promotion = {
      provider_id         = "github-promotion"
      service_account_id  = "dep-approved-promoter"
      display_name        = "dep-approved-promoter"
      description         = "Execution identity of the promotion lane."
      attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-promotion'"
      principal_value     = "example/dependency-authority"
    }
    revalidation = {
      provider_id         = "github-revalidation"
      service_account_id  = "dep-revalidation-controller"
      display_name        = "dep-revalidation-controller"
      description         = "Execution identity of the revalidation lane."
      attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-revalidation'"
      principal_value     = "example/dependency-authority"
    }
    revocation = {
      provider_id         = "github-revocation"
      service_account_id  = "dep-revocation-controller"
      display_name        = "dep-revocation-controller"
      description         = "Execution identity of the revocation lane."
      attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-revocation'"
      principal_value     = "example/dependency-authority"
    }
    consumer-verification = {
      provider_id         = "github-consumer-verification"
      service_account_id  = "dep-consumer-verifier"
      display_name        = "dep-consumer-verifier"
      description         = "Execution identity of the consumer verification lane."
      attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-consumer-verification'"
      principal_value     = "example/dependency-authority"
    }
  }

  evidence_bucket_name = "test-dep-evidence-archive"

  workload_job_images = {
    "dep-admission"             = "europe-west1-docker.pkg.dev/test-dep-control/release-controller-images/dependency-admission-controller@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    "dep-promotion"             = "europe-west1-docker.pkg.dev/test-dep-control/release-controller-images/dependency-promotion-controller@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    "dep-revalidation"          = "europe-west1-docker.pkg.dev/test-dep-control/release-controller-images/dependency-revalidation-controller@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    "dep-revocation"            = "europe-west1-docker.pkg.dev/test-dep-control/release-controller-images/dependency-revocation-controller@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    "dep-consumer-verification" = "europe-west1-docker.pkg.dev/test-dep-control/release-controller-images/dependency-consumer-verification-controller@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  }

  forensics_group = "group:dep-forensics-readers@example.com"

  enabled_workload_jobs = [
    "dep-admission",
    "dep-promotion",
    "dep-revalidation",
    "dep-revocation",
    "dep-consumer-verification",
  ]

  break_glass_recovery = {
    max_request_duration = "7200s"
    approvers            = ["group:dep-break-glass-approvers@example.com"]
  }

  perimeter_ingress = {
    perimeter_name = "accessPolicies/100000000001/servicePerimeters/dependency_authority"
    zone_projects  = ["projects/100000000002", "projects/100000000003"]
  }

  # The synthetic form of the canonical configuration binding: the static,
  # non-credential environment bindings of the zone's workload jobs, keyed by
  # the canonical job name. Every value is synthetic test data.
  workload_job_env = {
    "dep-admission" = {
      DEPENDENCY_AUTHORITY_ZONE      = "control"
      DEPENDENCY_AUTHORITY_ECOSYSTEM = "go"
    }
  }

  # The synthetic form of the canonical lifecycle binding: the staging class
  # carries the time-based delete plus the keep floor, the release class
  # carries the keep floor only, and both carry the fail-safe dry-run
  # posture.
  workload_image_cleanup = {
    staging = {
      dry_run = true
      policies = {
        "delete-stale-staging" = {
          action = "DELETE"
          condition = {
            tag_state  = "ANY"
            older_than = "30d"
          }
        }
        "keep-recent-staging" = {
          action = "KEEP"
          most_recent_versions = {
            keep_count = 2
          }
        }
      }
    }
    release = {
      dry_run = true
      policies = {
        "keep-recent-release" = {
          action = "KEEP"
          most_recent_versions = {
            keep_count = 5
          }
        }
      }
    }
  }
}

run "accepts_a_valid_state_bucket_name" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "test-dep-control-state"
  }

  assert {
    condition     = var.state_bucket_name == "test-dep-control-state"
    error_message = "The validation must accept a syntactically valid zone state bucket name."
  }
}

run "rejects_a_bucket_name_with_invalid_characters" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    state_bucket_name = "Test-Dep-Control-State"
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
    state_bucket_name = "goog-dep-control-state"
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
    condition     = contains(var.enabled_workload_jobs, "dep-consumer-verification")
    error_message = "The activation set must carry the declared jobs of the zone topology."
  }
}

run "rejects_an_unknown_enabled_job" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    enabled_workload_jobs = ["dep-admission", "dep-ghost"]
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

run "accepts_the_canonical_cleanup_binding" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.workload_image_cleanup.release.dry_run == true
    error_message = "The cleanup binding must arrive with the fail-safe dry-run posture."
  }
}

run "rejects_a_delete_policy_on_the_release_class" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    workload_image_cleanup = {
      staging = {
        dry_run = true
        policies = {
          "delete-stale-staging" = {
            action = "DELETE"
            condition = {
              tag_state  = "ANY"
              older_than = "30d"
            }
          }
          "keep-recent-staging" = {
            action = "KEEP"
            most_recent_versions = {
              keep_count = 2
            }
          }
        }
      }
      release = {
        dry_run = true
        policies = {
          "delete-old-release" = {
            action = "DELETE"
            condition = {
              older_than = "365d"
            }
          }
          "keep-recent-release" = {
            action = "KEEP"
            most_recent_versions = {
              keep_count = 5
            }
          }
        }
      }
    }
  }

  expect_failures = [var.workload_image_cleanup]
}

run "rejects_a_staging_binding_without_the_time_delete" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    workload_image_cleanup = {
      staging = {
        dry_run = true
        policies = {
          "keep-recent-staging" = {
            action = "KEEP"
            most_recent_versions = {
              keep_count = 2
            }
          }
        }
      }
      release = {
        dry_run = true
        policies = {
          "keep-recent-release" = {
            action = "KEEP"
            most_recent_versions = {
              keep_count = 5
            }
          }
        }
      }
    }
  }

  expect_failures = [var.workload_image_cleanup]
}

run "rejects_a_staging_binding_without_the_keep_floor" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    workload_image_cleanup = {
      staging = {
        dry_run = true
        policies = {
          "delete-stale-staging" = {
            action = "DELETE"
            condition = {
              tag_state  = "ANY"
              older_than = "30d"
            }
          }
        }
      }
      release = {
        dry_run = true
        policies = {
          "keep-recent-release" = {
            action = "KEEP"
            most_recent_versions = {
              keep_count = 5
            }
          }
        }
      }
    }
  }

  expect_failures = [var.workload_image_cleanup]
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
    condition     = var.controllers["admission"].description == "Execution identity of the admission lane."
    error_message = "Every controller identity must bind its canonical description surface."
  }
}

run "rejects_an_empty_network_description" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    workload_network = {
      network_name               = "dep-control-workload"
      subnet_name                = "dep-control-workload-europe-west1"
      subnet_cidr                = "10.10.0.0/26"
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
    controllers = {
      admission = {
        provider_id         = "github-admission"
        service_account_id  = "dep-admission-controller"
        display_name        = ""
        description         = "Execution identity of the admission lane."
        attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-admission'"
        principal_value     = "example/dependency-authority"
      }
      promotion = {
        provider_id         = "github-promotion"
        service_account_id  = "dep-approved-promoter"
        display_name        = "dep-approved-promoter"
        description         = "Execution identity of the promotion lane."
        attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-promotion'"
        principal_value     = "example/dependency-authority"
      }
      revalidation = {
        provider_id         = "github-revalidation"
        service_account_id  = "dep-revalidation-controller"
        display_name        = "dep-revalidation-controller"
        description         = "Execution identity of the revalidation lane."
        attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-revalidation'"
        principal_value     = "example/dependency-authority"
      }
      revocation = {
        provider_id         = "github-revocation"
        service_account_id  = "dep-revocation-controller"
        display_name        = "dep-revocation-controller"
        description         = "Execution identity of the revocation lane."
        attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-revocation'"
        principal_value     = "example/dependency-authority"
      }
      consumer-verification = {
        provider_id         = "github-consumer-verification"
        service_account_id  = "dep-consumer-verifier"
        display_name        = "dep-consumer-verifier"
        description         = "Execution identity of the consumer verification lane."
        attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-consumer-verification'"
        principal_value     = "example/dependency-authority"
      }
    }
  }

  expect_failures = [var.controllers]
}

run "accepts_the_canonical_workload_job_env_binding" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.workload_job_env["dep-admission"].DEPENDENCY_AUTHORITY_ZONE == "control"
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
        DEPENDENCY_AUTHORITY_ZONE = "control"
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
      "dep-admission" = {
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
    condition     = var.project_number == "100000000010"
    error_message = "The validation must accept the numeric Google Cloud project number of the zone."
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
    organization_number = "test-dep-control"
  }

  expect_failures = [var.organization_number]
}
