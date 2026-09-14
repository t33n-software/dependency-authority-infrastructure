terraform {
  required_version = "= 1.12.5"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "= 7.44.0"
    }

    google-beta = {
      source  = "hashicorp/google-beta"
      version = "= 7.44.0"
    }
  }

  # The zone root consumes its state home — the dedicated state bucket of the
  # zone, provisioned by the converged foundation, never by this root and
  # never by hand — with the state-key grammar prefix identifying exactly
  # this root and nothing else.
  backend "gcs" {
    bucket = var.state_bucket_name
    prefix = "dep-approved"
  }

  # The engine layer of the dual fortress state-encryption standard: every
  # state and plan artifact of this root is client-side encrypted with
  # AES-256-GCM through the organization key management, fail-closed
  # enforced, before it reaches any backend.
  encryption {
    key_provider "gcp_kms" "main" {
      # Instance binding: the concrete engine key never appears in code.
      kms_encryption_key = var.state_encryption_key
      # 32 bytes = AES-256-GCM, the documented maximum of the method.
      key_length = 32
      # Stable metadata identity: a future rename of the provider does not
      # break the readability of already encrypted artifacts.
      encrypted_metadata_alias = "state-encryption"
    }

    method "aes_gcm" "main" {
      keys = key_provider.gcp_kms.main
    }

    state {
      method   = method.aes_gcm.main
      enforced = true # Fail-closed: never an unencrypted state write
    }

    plan {
      method   = method.aes_gcm.main
      enforced = true # Fail-closed: never an unencrypted plan write
    }

    remote_state_data_sources {
      default {
        method = method.aes_gcm.main
      }
    }
  }
}
