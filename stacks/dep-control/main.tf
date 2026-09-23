locals {
  workload_image_registries = {
    staging = {
      description = "Dependency authority workload images staging: the only delivery target of the governed producer channel, never a workload source"
    }
    release = {
      description = "Dependency authority workload images release: the only workload consumption source, filled exclusively through promotion of a proven staging digest"
    }
  }

  # The canonical workload job topology of the control zone: exactly one job
  # per lane operation, each bound to the existing zone workload identity of
  # its lane and invoked through its dedicated invoke-only trigger identity.
  # The image digests are instance bindings (planned until the promotion
  # read-back proof flips them to bound), never stack defaults.
  workload_jobs = {
    "dep-admission" = {
      identity_key = "admission"
      trigger_id   = "dep-admission-trigger"
    }
    "dep-promotion" = {
      identity_key = "promotion"
      trigger_id   = "dep-promotion-trigger"
    }
    "dep-revalidation" = {
      identity_key = "revalidation"
      trigger_id   = "dep-revalidation-trigger"
    }
    "dep-revocation" = {
      identity_key = "revocation"
      trigger_id   = "dep-revocation-trigger"
    }
    "dep-consumer-verification" = {
      identity_key = "consumer-verification"
      trigger_id   = "dep-consumer-verifier-trigger"
    }
  }

  # The invoke-only trigger identity of each controller lane, keyed by the
  # lane identity key: the lane federates to this identity, never to the
  # execution identity.
  controller_triggers = {
    for job, spec in local.workload_jobs : spec.identity_key => spec.trigger_id
  }

  # The canonical human-readable description surfaces of the zone identity and
  # audit export boundaries (the mandatory description duty of the mandatory
  # resource properties convention): the canonical intent texts of the zone
  # workload identity pool and the zone audit sink.
  workload_identity_pool_description = "Workload identity pool of the control zone: federates exactly the lane trigger identities of the zone, never the execution identities."
  audit_sink_description             = "Zone audit export: exports the zone project's Cloud Audit Logs into the evidence archive bucket of the evidence zone."
}

provider "google" {
  project = var.project_id
}

# The exactly pinned beta provider of the declaration plane: it declares only
# what the pinned GA provider provably lacks (the remote upstream allowance);
# everything else stays on the GA provider.
provider "google-beta" {
  project = var.project_id
}

module "policy_bindings" {
  source     = "../../policy-bindings"
  project_id = var.project_id

  disable_service_account_key_creation  = var.policy_constraints.disable_service_account_key_creation
  disable_service_account_key_upload    = var.policy_constraints.disable_service_account_key_upload
  uniform_bucket_level_access           = var.policy_constraints.uniform_bucket_level_access
  public_access_prevention              = var.policy_constraints.public_access_prevention
  cloud_run_vpc_egress_all_traffic_only = var.policy_constraints.cloud_run_vpc_egress_all_traffic_only
  cloud_run_ingress_internal_only       = var.policy_constraints.cloud_run_ingress_internal_only
}

# The zone workload network origin: exactly one VPC with one subnetwork in the
# job region carrying Private Google Access, the restricted-range DNS response
# policy and the egress firewall pair. The zone's workload jobs attach to it
# with Direct VPC egress and all-traffic routing, so their calls to the
# restricted planes originate inside the perimeter.
module "network" {
  source     = "../../modules/network"
  project_id = var.project_id

  workload_network = merge(var.workload_network, {
    region = var.location
  })
}

module "workload_image_registries" {
  source   = "../../modules/artifact-registry"
  for_each = local.workload_image_registries

  project_id    = var.project_id
  location      = var.location
  repository_id = "${each.key}-controller-images"
  description   = each.value.description
  format        = "DOCKER"
  mode          = "STANDARD_REPOSITORY"

  # The declared, platform-executed workload image lifecycle: the organization
  # instance binds the canonical convention values and the dry-run activation
  # state; the core never presets them.
  cleanup_policies       = var.workload_image_cleanup[each.key].policies
  cleanup_policy_dry_run = var.workload_image_cleanup[each.key].dry_run

  labels = {
    boundary = "dependency-authority"
    zone     = "control"
  }
}

module "workload_identity" {
  source         = "../../modules/workload-identity"
  project_id     = var.project_id
  project_number = var.project_number
  pool_id        = var.pool_id

  # The canonical human-readable surfaces of the zone pool (the mandatory
  # description duty): the display name binds the pool's identity class name,
  # the description the canonical intent text.
  pool_display_name = var.pool_id
  pool_description  = local.workload_identity_pool_description

  identities = {
    for lane, controller in var.controllers : lane => merge(controller, {
      trigger_service_account_id = local.controller_triggers[lane]
    })
  }
}

module "workload_jobs" {
  source = "../../modules/cloud-run-job"
  # The engine-active surface of the zone's job topology: exactly the
  # instance-bound activation set — the bound jobs plus the jobs being
  # provisioned in the current window. A declared-but-planned job never
  # enters the plan until its provisioning window activates it.
  for_each = { for job, spec in local.workload_jobs : job => spec if contains(var.enabled_workload_jobs, job) }

  project_id            = var.project_id
  location              = var.location
  name                  = each.key
  service_account_email = module.workload_identity.service_account_emails[each.value.identity_key]
  invoker_member        = "serviceAccount:${module.workload_identity.trigger_service_account_emails[each.value.identity_key]}"
  image                 = var.workload_job_images[each.key]
  network               = module.network.workload_network_id
  subnetwork            = module.network.workload_subnetwork_id

  # The declaration owns the static, non-credential configuration of every
  # workload job completely (the workload configuration ownership
  # convention): the organization instance binds the proven values as
  # reviewed configuration, operation inputs travel as validated execution
  # parameters of the invocation, and credentials never travel this surface.
  env = lookup(var.workload_job_env, each.key, {})

  labels = {
    boundary = "dependency-authority"
    zone     = "control"
  }
}

# The canonical IAM target matrix of the workload image registries: every
# zone lane identity reads the release class (the only workload consumption
# source); no identity ever receives a writer grant on either class, and the
# staging class carries no binding at all — it is filled exclusively by the
# governed producer channel and is never a workload source.
module "workload_image_registry_iam" {
  source     = "../../modules/repository-iam"
  project_id = var.project_id
  location   = var.location
  repository = module.workload_image_registries["release"].id

  readers = concat(
    [for lane in keys(var.controllers) : "serviceAccount:${module.workload_identity.service_account_emails[lane]}"],
    tolist(var.cross_zone_workload_reader_members),
  )
}

module "audit_log_sink" {
  source             = "../../modules/logging"
  project_id         = var.project_id
  destination_bucket = var.evidence_bucket_name

  sinks = {
    (var.audit_sink_name) = {
      filter      = var.audit_log_filter
      description = local.audit_sink_description
    }
  }
}

# The forensics reader access class: the organization-owned forensics group
# holds exactly the two read-only diagnostic roles on this zone project and no
# other grant. The control zone additionally declares the second, separate
# perimeter ingress rule of the class (the forensics group as the only
# identity, scoped to the read-only logging method on the zone projects): the
# boundary-level binding of the class lives exactly once, here.
module "forensics_readers" {
  source     = "../../modules/forensics-readers"
  project_id = var.project_id

  forensics_group   = var.forensics_group
  perimeter_ingress = var.perimeter_ingress
}

# The recovery identity of the control zone: the dedicated identity whose
# elevated project role exists only under the mandatory time-bound IAM
# condition. It is never used in normal operation, never federated from CI and
# holds no data-plane grant; every use is an audited incident action with a
# recorded decision. The role and the end time are approved instance
# decisions, supplied through the instance-bound input.
module "recovery" {
  source     = "../../modules/recovery"
  project_id = var.project_id

  role               = var.break_glass_recovery.role
  condition_end_time = var.break_glass_recovery.condition_end_time
}
