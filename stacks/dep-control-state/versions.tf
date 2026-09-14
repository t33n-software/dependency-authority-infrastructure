terraform {
  required_version = "= 1.12.5"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "= 7.44.0"
    }
  }

  # The state home of the control zone: the gcs backend into the zone's
  # dedicated state bucket (an instance binding, created by this root's own
  # bootstrap), with the state-key grammar prefix identifying exactly this
  # root — never environment, date, operator or session segments.
  backend "gcs" {
    bucket = var.state_bucket_name
    prefix = "dep-control-state"
  }

  # The engine layer of the dual fortress state-encryption standard: every
  # state and plan artifact of this root is client-side encrypted with
  # AES-256-GCM through the organization key management, fail-closed
  # enforced, before it reaches any backend — including the local bootstrap
  # state of this root, which is encrypted from birth.
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
