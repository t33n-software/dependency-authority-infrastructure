# The behavioral proof of the corrected state_bucket_name validation of the
# dep-control root: the acceptance path through an assertion and the rejection
# paths through expect_failures — every run is a plan with refresh disabled,
# and no run creates infrastructure.
#
# The root carries the gcs backend binding and the dual-fortress encryption
# block, so its initialization resolves the state bucket and the encryption
# key material: this proof executes in the governed execution window, where
# the instance-bound variable channel (a gitignored *.tfvars) supplies
# state_bucket_name and state_encryption_key. Both are never assigned in this
# file, and every value assigned here is synthetic test data.

variables {
  project_id = "test-dep-control"
  location   = "europe-west1"

  workload_network = {
    network_name = "dep-control-workload"
    subnet_name  = "dep-control-workload-europe-west1"
    subnet_cidr  = "10.10.0.0/26"
  }

  controllers = {
    admission = {
      provider_id         = "github-admission"
      service_account_id  = "dep-admission-controller"
      attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-admission'"
      principal_value     = "example/dependency-authority"
    }
    promotion = {
      provider_id         = "github-promotion"
      service_account_id  = "dep-approved-promoter"
      attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-promotion'"
      principal_value     = "example/dependency-authority"
    }
    revalidation = {
      provider_id         = "github-revalidation"
      service_account_id  = "dep-revalidation-controller"
      attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-revalidation'"
      principal_value     = "example/dependency-authority"
    }
    revocation = {
      provider_id         = "github-revocation"
      service_account_id  = "dep-revocation-controller"
      attribute_condition = "assertion.repository == 'example/dependency-authority' && assertion.environment == 'dep-revocation'"
      principal_value     = "example/dependency-authority"
    }
    consumer-verification = {
      provider_id         = "github-consumer-verification"
      service_account_id  = "dep-consumer-verifier"
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

  perimeter_ingress = {
    perimeter_name = "accessPolicies/100000000001/servicePerimeters/dependency_authority"
    zone_projects  = ["projects/100000000002", "projects/100000000003"]
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
