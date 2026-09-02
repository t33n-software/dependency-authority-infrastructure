# Module: cloud-run-job

One Google Cloud Run job of the dependency authority workload job topology:
the single in-perimeter execution form of one lane operation, invoked through
the external trigger seam and never called by a lane against a restricted
data plane directly.

## Boundary

- Never carries organization, tenant, project, identity, network, secret or
  registry bindings; every concrete value is an instance-supplied variable.
- The job set is canonical and declared atomically: every lane operation owns
  exactly one job named `dep-<operation>` in its own zone (intake-fetch in
  the intake zone; admission, promotion, revalidation and revocation in the
  control zone; evidence-write and evidence-audit in the evidence zone). The
  complete set is declared in one change, never incrementally per proven
  image — a partial declaration would leave a shadow topology outside
  governed control.
- The declaration binds structure only: the job, its zone, its identity and
  its input surface. The image digest is an instance binding, never an
  infrastructure default: the organization instance records it with binding
  status `planned` (a documented placeholder) until the promotion read-back
  proof exists, and flips it to `bound` only after that proof. A placeholder
  fails provisioning closed.
- The image reference is always the full immutable `@sha256:` digest of a
  release-class workload image registry — never a tag, never a mutable
  reference, never the staging class, never a dependency repository.
- The job executes as the existing zone workload identity of its operation.
  The lane identity holds invoke permission on exactly its own job and
  nothing else; neither the lane nor the job identity ever holds a write
  grant on a workload image registry.
- Operation inputs travel as validated execution parameters of the
  invocation; the workload re-validates them fail-closed before any effect.
  Static environment bindings carry no sensitive values.

## Usage

```hcl
module "dep_intake_fetch" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/cloud-run-job?ref=<exact-version>"

  project_id            = "<organization>-dep-intake"
  location              = "<region>"
  name                  = "dep-intake-fetch"
  service_account_email = "<zone-workload-identity-email>"
  image                 = "<region>-docker.pkg.dev/<organization>-dep-control/release-controller-images/dependency-intake-controller@sha256:<64 lowercase hex>"

  labels = {
    boundary = "dependency-authority"
    zone     = "intake"
  }
}
```

## Notes

- The module creates no project and enables no APIs; the organization
  instance provisions the project and its API surface first.
- The module never creates service accounts, pools or providers; the zone
  workload identity module owns them.
