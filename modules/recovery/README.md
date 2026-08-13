# Module: recovery

The break-glass recovery identity of a trust zone: a dedicated service account
whose elevated project role exists only under a mandatory time-bound IAM
condition.

## Boundary

- Never used in normal operation; every use is an audited incident action with
  a recorded decision.
- The grant is always time-bounded (`condition_end_time`); an unbounded
  break-glass grant is a contract violation.
- The exact role and the end time are approved instance decisions; the core
  carries no default role.

## Usage

```hcl
module "break_glass" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/recovery?ref=<exact-version>"

  project_id         = "<organization>-dep-evidence"
  role               = "<approved-break-glass-role>"
  condition_end_time = "<approved-rfc3339-end-time>"
}
```
