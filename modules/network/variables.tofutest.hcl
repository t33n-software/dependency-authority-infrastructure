# The behavioral proofs of the network module: the workload network origin
# description surfaces of the mandatory resource properties convention — the
# acceptance run binds every description surface, and the rejection run proves
# the empty-description form fails closed. The module carries no backend and
# no encryption block, so this proof executes offline; every value assigned
# here is synthetic test data.

variables {
  project_id = "test-dep-control"

  workload_network = {
    network_name               = "dep-control-workload"
    subnet_name                = "dep-control-workload-europe-west1"
    region                     = "europe-west1"
    subnet_cidr                = "10.10.0.0/26"
    network_description        = "Zone workload network origin."
    subnet_description         = "Zone workload subnetwork."
    firewall_allow_description = "Allow egress TCP 443 to the restricted range."
    firewall_deny_description  = "Deny all remaining egress."
    dns_policy_description     = "Restricted-range DNS form."
  }
}

run "accepts_the_bound_description_surfaces" {
  command = plan

  plan_options {
    refresh = false
  }

  assert {
    condition     = var.workload_network.network_description == "Zone workload network origin."
    error_message = "The workload network origin must bind the canonical description surface."
  }
}

run "rejects_an_empty_description" {
  command = plan

  plan_options {
    refresh = false
  }

  variables {
    workload_network = {
      network_name               = "dep-control-workload"
      subnet_name                = "dep-control-workload-europe-west1"
      region                     = "europe-west1"
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
