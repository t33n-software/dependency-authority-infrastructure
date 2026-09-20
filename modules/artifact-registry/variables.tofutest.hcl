# The behavioral proofs of the artifact-registry module's cleanup policy class
# rule: cleanup policies are the declared, platform-executed workload image
# lifecycle form and bind only to DOCKER workload image repositories — the
# dependency repositories and the evidence plane are append-only supply-chain
# records and never carry one. The module carries no backend, so this proof
# executes offline: every run is a plan with refresh disabled, and no run
# creates infrastructure. Every value assigned here is synthetic test data.

variables {
  project_id    = "test-dep-control"
  location      = "europe-west1"
  repository_id = "test-repository"
}

run "accepts_docker_cleanup_policies" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    format = "DOCKER"
    cleanup_policies = {
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

  assert {
    condition     = google_artifact_registry_repository.this.cleanup_policy_dry_run == true
    error_message = "The module must default the cleanup pipeline to the fail-safe dry-run posture."
  }
}

run "accepts_a_dependency_repository_without_cleanup_policies" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    format = "GO"
  }

  assert {
    condition     = length(google_artifact_registry_repository.this.cleanup_policies) == 0
    error_message = "A dependency repository carries no cleanup policy: it is an append-only supply-chain record."
  }
}

run "rejects_cleanup_policies_on_a_dependency_repository" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    format = "GO"
    cleanup_policies = {
      "keep-recent" = {
        action = "KEEP"
        most_recent_versions = {
          keep_count = 2
        }
      }
    }
  }

  expect_failures = [var.cleanup_policies]
}

run "rejects_cleanup_policies_on_an_evidence_repository" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    format = "GENERIC"
    cleanup_policies = {
      "keep-recent" = {
        action = "KEEP"
        most_recent_versions = {
          keep_count = 2
        }
      }
    }
  }

  expect_failures = [var.cleanup_policies]
}

run "rejects_an_unknown_cleanup_action" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    format = "DOCKER"
    cleanup_policies = {
      "expire-everything" = {
        action = "EXPIRE"
      }
    }
  }

  expect_failures = [var.cleanup_policies]
}
