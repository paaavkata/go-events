# Domain audit contract

Two tiers cover "who did what, where, when, with what result":

| Tier | What | Where | Retention |
|---|---|---|---|
| Access audit | every HTTP request through Traefik: method, path (query stripped), host, status, latency, client IP, `X-App-Id`, `X-User-Id`, `X-User-Uid`, `X-Api-Key-Uid`, `X-Is-Admin` | Traefik JSON access log, Promtail, Loki (`{namespace="infra", container="traefik"}`) | 90 days |
| Domain audit | the security- and money-sensitive state changes below, as `AuditEvent` | `PublishAudit`, NATS `audit-events.<app_id>`, event-service `event.events` | 400 days, 1095 days for the MUST list |

Services do NOT emit per-request or read-path audit events. The access log already has them.

## MUST-audit actions

A service that performs one of these actions publishes an `AuditEvent` with a
`Type` using the listed prefix. event-service keeps rows with these prefixes
(plus any `*.deleted` type and severity CRITICAL/ALERT/FATAL) for the long
security window (`AUDIT_SECURITY_RETENTION_DAYS`).

| Action | `Type` examples | Owner today |
|---|---|---|
| Login, logout, failed login, password reset | `auth.login`, `auth.login_failed`, `auth.logout` | gap: Keycloak realm events are not enabled and identity-service emits nothing; only the access log (POSTs to `/realms/*/login-actions/*`, `/api/identity/v1/auth/*` with status) |
| API key create / revoke | `apikey.created`, `apikey.revoked` | identity-service: gap, see event-service completion doc |
| Payment succeeded / failed / refunded, credits purchased | `payment.succeeded`, `payment.failed`, `payment.refunded`, `credits.purchased` | payment-service (NotificationService) |
| Subscription created / changed / canceled, customer created | `subscription.*`, `customer.created` | payment-service (NotificationService) |
| Admin actions | `admin.<resource>.<verb>`, e.g. `admin.target.authorized` | scantinel finding/scan/target, cms moderation |
| Resource deletes | `<resource>.deleted`, e.g. `content.deleted` | each owning service |
| Role / permission changes | `role.assigned`, `role.revoked`, `permission.changed` | gap: identity-service emits nothing; Keycloak admin events not enabled |
| GDPR erasure | `audit.actor_redacted` | event-service (`POST /v1/event/redact`) |

## Envelope conventions

- `AppID`: the trusted tenant (`X-App-Id` or the owning row's app). Never defaulted.
- `UID`: stable per logical event, so a retried publish dedupes (`Nats-Msg-Id` and
  the `(app_id, event_id)` key). Derive it from a provider event id when one exists.
- `Actor`: `{Type: "user", UID: <X-User-Id>}` for a user request, `{Type: "service" | "system",
  UID: <service key>}` for background work, `IP` only for user requests.
- Keycloak users: when the request carries `X-User-Uid` (Keycloak `sub`), put it in
  `Metadata` as `"user_uid"`. GDPR erasure matches it there.
- `Target`: the affected object. Use `Type: "user"` with the user id when the action is
  about a person (role change, account delete), so erasure finds it.
- `Severity`: `INFO` for normal success, `WARNING` for denied/failed security actions
  and erasures, `ERROR` for failed money actions, `CRITICAL` for suspected abuse.
- `Message`: short, no personal data. `Metadata`: ids and amounts only. Never emails,
  names, addresses, tokens, secrets or request bodies.

## Publishing

Fire-and-forget, never fail or block the request: nil-safe publisher, own goroutine or
already-async path, short timeout, log on error. Reference:
`platform-backend-services/email-service/internal/audit/audit.go`.
