# Stack: dep-control-state

The minimal state-home root of the control trust zone: the dedicated state
bucket of the zone (object versioning, uniform bucket-level access, enforced
public access prevention, the mandatory bucket CMEK as the provider layer of
the dual fortress state-encryption standard), exactly the
`roles/storage.objectAdmin` operator data plane on the bucket, and the
engine-layer encryption binding of the root itself (client-side AES-256-GCM
state and plan encryption through the organization key management,
fail-closed enforced from birth).

## Boundary

- The root declares the state home of the control zone exactly once; no
  other root owns a control-zone state surface, and the bucket never
  carries non-state content.
- The root is the only root that ever applies with local state: its
  encryption block precedes the first apply, so even the local bootstrap
  state is encrypted from birth; immediately after the bucket exists, the
  root migrates its own state into it (`tofu init -migrate-state`), and
  from then on no local state exists anywhere.
- The engine key (`state_encryption_key`) and the bucket CMEK key
  (`state_bucket_cmek_key`) are disjoint cryptographic boundaries, both
  instance bindings without defaults; the keys exist before any apply
  through the governed operator channel (the root-of-trust act) and
  converge into engine management with the foundation convergence.
- The backend prefix `dep-control-state` identifies exactly this root (the
  state-key grammar); the bucket name is the instance binding.
- The stack creates no project, enables no APIs and declares no zone
  workload surface; the zone stacks own those.

## Inputs

`project_id`, `location`, `state_bucket_name` (the instance-bound bucket
name), `state_encryption_key` (the instance-bound engine key reference),
`state_bucket_cmek_key` (the instance-bound bucket CMEK key reference,
proven cryptographically separate and location-coupled fail-closed) and
`operator_members` (the instance-bound operator execution identities).

## Outputs

The state bucket name, resource ID and URL, and the operator object-admin
binding IDs keyed by member.
