# Break-glass recovery: activation and drill

The activation and drill runbook of the break-glass recovery identity of a
trust zone. The identity holds no standing grant: its elevated capability is
carried exclusively by the declared Privileged Access Manager entitlement of
its zone project, dormant until activated. Every activation is approval- and
justification-bound, time-boxed by the platform-enforced grant duration and
audited, and the grant auto-expires at the end of the activation window.

## Boundary

- The entitlement is provisioned exclusively by the engine (the pinned
  provider resource), never by the CLI; the CLI surface below is the operator
  and drill channel only.
- The operator and drill surface is the GA `gcloud pam` command group of the
  bound Google Cloud SDK; the production carrier is the pinned provider
  resource, never the CLI.
- The eligible principal of the entitlement is the zone's own break-glass
  recovery identity, never the platform operator's personal identity.
- The approver set is an organization-instance binding, distinct from the
  eligible principal by default; any self-approval is a documented, expiring
  interim state, never the target.
- Every step of an activation is an audited incident action: the grant
  request, the approval, the elevated calls and the expiry land in the zone
  audit export of the evidence plane.

## Prerequisites

- The entitlement of the zone is provisioned and proven by read-back:

  ```powershell
  gcloud pam entitlements describe "<ENTITLEMENT_ID>" --project="<PROJECT_ID>" --location=global
  ```

- The operator holds the standing organization-plane granting capability (the
  mutation channel), and the zone's break-glass recovery service account
  exists.
- The drill or incident is recorded as a reviewed operational event before it
  begins.

## Activation

1. Grant the impersonation capability on the break-glass identity through the
   operator channel (a time-boxed window grant), and prove the binding by
   read-back:

   ```powershell
   gcloud iam service-accounts add-iam-policy-binding "<BREAK_GLASS_SA_EMAIL>" --member="user:<OPERATOR>" --role="roles/iam.serviceAccountUser" --format=none
   gcloud iam service-accounts get-iam-policy "<BREAK_GLASS_SA_EMAIL>"
   ```

2. Request the grant as the break-glass identity (impersonated), with the
   justification recorded:

   ```powershell
   gcloud pam grants create --entitlement="<ENTITLEMENT_ID>" --project="<PROJECT_ID>" --location=global --requested-duration="<DURATION>" --justification="<INCIDENT_OR_DRILL_REFERENCE>" --impersonate-service-account="<BREAK_GLASS_SA_EMAIL>"
   ```

   The requested duration never exceeds the entitlement's bound
   `max_request_duration`; the platform enforces the time-box.

3. The approver approves the grant with a recorded reason:

   ```powershell
   gcloud pam grants approve "<GRANT_NAME>" --reason="<APPROVAL_REFERENCE>"
   ```

4. Prove the elevated capability through the impersonated identity (a call
   that fails closed without the grant), and read the grant state back:

   ```powershell
   gcloud pam grants describe "<GRANT_NAME>"
   ```

5. Let the grant expire at the end of its activation window — or revoke it
   early with a recorded reason when the incident ends sooner:

   ```powershell
   gcloud pam grants revoke "<GRANT_NAME>" --reason="<CLOSURE_REFERENCE>"
   ```

6. Harden the impersonation binding and prove the hardened state by
   read-back:

   ```powershell
   gcloud iam service-accounts remove-iam-policy-binding "<BREAK_GLASS_SA_EMAIL>" --member="user:<OPERATOR>" --role="roles/iam.serviceAccountUser" --format=none
   gcloud iam service-accounts get-iam-policy "<BREAK_GLASS_SA_EMAIL>"
   ```

## Drill

The drill is the same procedure executed as a reviewed operational event on a
standing zone: activate, prove the elevated capability, let the grant expire,
harden, and land the evidence in the evidence plane. A recovery form that has
never been activated is unproven — the drill is the proof of the activation
path, never the existence of the entitlement alone.

## Evidence

Every activation and every drill lands in the evidence plane: the grant
request, the approval decision, the elevated calls and the expiry are recorded
in the zone audit export, and the drill record references them.
