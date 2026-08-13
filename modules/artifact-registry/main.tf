resource "google_artifact_registry_repository" "this" {
  project       = var.project_id
  location      = var.location
  repository_id = var.repository_id
  description   = var.description
  format        = var.format
  mode          = var.mode

  kms_key_name           = var.kms_key_name
  cleanup_policy_dry_run = var.cleanup_policy_dry_run
  labels                 = var.labels

  dynamic "remote_repository_config" {
    for_each = var.remote_upstream == null ? [] : [var.remote_upstream]
    content {
      description = remote_repository_config.value.common_uri != null ? "controlled common upstream" : "controlled public upstream"

      dynamic "npm_repository" {
        for_each = remote_repository_config.value.npm == null ? [] : [remote_repository_config.value.npm]
        content {
          public_repository = npm_repository.value
        }
      }

      dynamic "python_repository" {
        for_each = remote_repository_config.value.python == null ? [] : [remote_repository_config.value.python]
        content {
          public_repository = python_repository.value
        }
      }

      dynamic "common_repository" {
        for_each = remote_repository_config.value.common_uri == null ? [] : [remote_repository_config.value.common_uri]
        content {
          uri = common_repository.value
        }
      }
    }
  }

  dynamic "cleanup_policies" {
    for_each = var.cleanup_policies
    content {
      id     = cleanup_policies.key
      action = cleanup_policies.value.action

      dynamic "condition" {
        for_each = cleanup_policies.value.condition == null ? [] : [cleanup_policies.value.condition]
        content {
          tag_state             = condition.value.tag_state
          tag_prefixes          = condition.value.tag_prefixes
          version_name_prefixes = condition.value.version_name_prefixes
          package_name_prefixes = condition.value.package_name_prefixes
          older_than            = condition.value.older_than
          newer_than            = condition.value.newer_than
        }
      }

      dynamic "most_recent_versions" {
        for_each = cleanup_policies.value.most_recent_versions == null ? [] : [cleanup_policies.value.most_recent_versions]
        content {
          package_name_prefixes = most_recent_versions.value.package_name_prefixes
          keep_count            = most_recent_versions.value.keep_count
        }
      }
    }
  }
}
