# Module: logging

Audit log export from one trust-zone project into the evidence archive: one or
more Cloud Logging sinks with unique writer identities, plus the
append-focused bucket binding for each writer.

## Boundary

- Never carries organization, tenant, identity, network, secret or registry
  bindings; the destination bucket, filters and exclusions are
  instance-supplied values.
- Sink writers receive only `roles/storage.objectCreator` by default: append,
  never rewrite or delete.
- The destination bucket is owned by the evidence trust zone; this module only
  wires the sink writers of the source zone to it.

## Usage

```hcl
module "zone_audit_export" {
  source = "git::https://github.com/<organization>/dependency-authority-infrastructure//modules/logging?ref=<exact-version>"

  project_id         = "<organization>-dep-intake"
  destination_bucket = "<evidence-archive-bucket>"

  sinks = {
    "dep-intake-audit-to-evidence" = {
      filter = "logName:\"logs/cloudaudit.googleapis.com\""
    }
  }
}
```
