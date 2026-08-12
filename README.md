# Dependency Authority Infrastructure

`dependency-authority-infrastructure` is the organization-agnostic core of
generic infrastructure modules and parameterized reference stacks for the
dependency authority trust zones: control, intake, quarantine, approved, and
evidence.

This repository never contains concrete organization, tenant, project,
identity, network, secret, or registry bindings. Instances consume these
modules only through exact version pins.

Governed changes land through ticket branches and pull requests into
`develop`. `main` is the production and control-plane truth.
