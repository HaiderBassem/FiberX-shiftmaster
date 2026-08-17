# ShiftMaster API Reference

**Base URL:** `https://<host>/api`
**Content-Type:** `application/json`

Every response uses the same envelope:

```json
{ "success": true, "data": { }, "meta": { "count": 0 } }
```

Failures return `{ "success": false, "error": "..." }` with a 4xx or 5xx status.

> This file is maintained against `cmd/api/router.go`. The previous version had
> drifted badly — it documented logging in with `employee_code` when the API takes
> `email`, a `POST /schedules/generate` route that does not exist, and the
> `leave_type` enum that migration 017 removed.

---

## Authentication

Three token types exist and are **not interchangeable**. Each carries a `typ`
claim and is accepted on exactly one surface.

| Type | Lifetime | Accepted by |
|---|---|---|
| `access` | `JWT_ACCESS_EXPIRE_MIN` (default 15 min) | All protected routes, via `Authorization: Bearer` |
| `refresh` | `JWT_REFRESH_EXPIRE_DAYS` (default 7 days) | `POST /auth/refresh` only |
| `ws_ticket` | 30 seconds, single use | `GET /notifications/ws` only |

Presenting a refresh token as a bearer credential is rejected, as is presenting
an access token at the refresh endpoint.

Credentials are read **only** from the `Authorization` header. There is no
`?token=` query fallback; the WebSocket uses a ticket instead.

### Public

| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `POST` | `/auth/login` | Sign in | `{"email", "password"}` |
| `POST` | `/auth/refresh` | Exchange a refresh token | `{"refresh_token"}` |
| `POST` | `/auth/logout` | Clear the upload cookie | — |

`login` and `refresh` both return `{access_token, refresh_token, expires_in, employee}`
and set an `HttpOnly` cookie scoped to `/api/uploads` (see **Files**).

**Account lockout.** After `MAX_LOGIN_ATTEMPTS` consecutive failures an account is
locked for `LOCKOUT_DURATION_MIN` minutes and returns `account is locked or inactive`.
The lock expires on its own; it does not change employment status.

### Protected

| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `GET` | `/auth/me` | Current user | |
| `POST` | `/auth/change-password` | Change own password | `{"old_password", "new_password"}` |
| `POST` | `/auth/reset-password/:id` | Admin reset (manager/admin) | `{"new_password"}` |

---

## Department context

Admins and managers may act within a chosen department by sending:

```
X-Department-ID: <uuid>
```

Admins may use any department; a manager must manage the one they name; everyone
else is pinned to the department in their token.

---

## Roles

`employee` → `team_leader` → `manager` → `admin`. Route groups:

| Group | Roles |
|---|---|
| protected | any authenticated user |
| supervisor | team_leader, manager, admin |
| tlWrite | team_leader, manager, admin |
| admin | manager, admin |
| adminOnly | admin |

Some endpoints check a per-employee capability flag as well
(`can_create_tables`, `can_manage_help_docs`, `can_post_announcements`,
`can_manage_fiberx_data`, `can_manage_services`).

---

## Employees

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/employees` | List, scoped by role and department context |
| `GET` | `/employees/:id` | One employee |
| `GET` | `/employees/me/profile-stats` | Own balances and task counts |
| `POST` | `/employees/me/profile-picture` | Upload own photo (multipart, field `profile_picture`) |
| `POST` | `/employees` | Create (tlWrite) |
| `PUT` | `/employees/:id` | Update (tlWrite) |
| `PATCH` | `/employees/:id/status` | Change status (tlWrite) |
| `DELETE` | `/employees/:id` | Delete (tlWrite) |
| `PUT` | `/employees/:id/password` | Set password |
| `PUT` | `/employees/:id/preferences` | Save UI preferences |
| `PUT` | `/employees/:id/{fiberx,help,announcement,table,service}-permission` | Toggle a capability flag |

---

## Departments

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/departments` | List |
| `GET` | `/departments/my-managed` | Departments the caller manages |
| `GET` | `/departments/:id` | One department |
| `POST` `PUT` `DELETE` | `/departments[/:id]` | CRUD (**admin only**) |
| `POST` | `/departments/:id/managers` | Add a manager (admin) |
| `DELETE` | `/departments/:id/managers/:manager_id` | Remove a manager (admin) |
| `PUT` | `/departments/:id/fiberx-toggle` | Enable/disable FiberX data (admin) |

---

## Shifts and schedules

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/shifts`, `/shifts/:id` | Read |
| `POST` `PUT` `DELETE` | `/shifts[/:id]` | Manage (tlWrite) |
| `GET` | `/schedules/daily?date=YYYY-MM-DD` | One day |
| `GET` | `/schedules/department?...` | A department's week |
| `GET` | `/schedules/employee/:id?from=&to=` | One employee's range |
| `GET` | `/schedules/replacements?date=` | Candidates who were off the previous day |
| `GET` | `/schedules/pattern/:id` | An employee's weekly pattern |
| `PUT` | `/schedules/pattern/:id` | Replace the weekly pattern (tlWrite) |
| `POST` | `/schedules/shifts/set` | Set one day (tlWrite) |
| `DELETE` | `/schedules/shifts/:id` | Clear one day (tlWrite) |
| `POST` | `/schedules/shifts/:id/check-in`, `/check-out` | Attendance |
| `POST` | `/schedules/:id/publish` | Publish a week (manager/admin) |
| `POST` | `/schedules/shifts/:id/replace` | Assign a replacement (manager/admin) |

`POST /schedules/shifts/set` takes
`{"employee_id", "shift_date", "shift_id", "shift_status", "leave_reason", "permanent"}`.
`shift_status` is one of `working`, `off`, `leave`, `vacation`, `hourly`.
With `permanent: true` the weekly pattern is updated as well and future weeks are
resynchronised; the day and the pattern are written in one transaction.

Weeks are materialised lazily on read. Only rows with `source = "generated"` and
dated today or later are ever rewritten — manual edits, leave overlays and past
days are immutable.

---

## Leaves

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/leaves` | Request `{"leave_type_id", "start_date", "end_date", "reason", "start_time", "end_time"}` |
| `GET` | `/leaves/me` | Own requests |
| `GET` | `/leaves/my-balances?year=` | Own balances |
| `GET` | `/leaves/pending` | Awaiting the caller's approval |
| `POST` | `/leaves/:id/cancel` | Cancel own pending request |
| `GET` | `/leaves/pending/rich`, `/leaves/history`, `/leaves/coverage-preview` | Supervisor views |
| `POST` | `/leaves/:id/approve/team-leader` | Team-leader approval |
| `POST` | `/leaves/:id/approve/manager` | Manager approval |
| `POST` | `/leaves/:id/reject` | Reject `{"reason"}` |
| `POST` | `/leaves/:id/cancel-approval` | Undo an approval |

`status` is one of `pending`, `approved_by_team_leader`, `approved_by_manager`,
`rejected`, `cancelled`. **There is no `approved` value.**

Leave-type behaviour comes from explicit columns, never from the type's name:
`is_hourly` marks part-day leave, `bypasses_daily_limit` exempts a type from the
department's daily caps. Both are editable through `/leave-types`.

### Leave types and balances

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/leave-types`, `/leave-types/:id` | Read |
| `POST` `PUT` `DELETE` | `/leave-types[/:id]` | Manage (manager/admin) |
| `POST` | `/leave-balances/sync` | Rebuild balances (admin) |
| `GET` | `/leave-balances/employee/:id` | Read (supervisor) |
| `PUT` | `/leave-balances/employee/:id/:leave_type_id` | Adjust (admin) |

---

## Swaps

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/swaps` | Request `{"target_employee_id", "shift_date", "reason"}` |
| `GET` | `/swaps/me`, `/swaps/pending/for-me` | Own and incoming |
| `GET` | `/swaps/eligible-targets`, `/swaps/eligible-shift-targets` | Candidates |
| `POST` | `/swaps/:id/respond` | Accept or decline `{"accepted": true}` |
| `POST` | `/swaps/:id/cancel` | Withdraw |
| `GET` | `/swaps/pending/manager`, `/swaps/history` | Supervisor views |
| `POST` | `/swaps/:id/approve`, `/reject`, `/cancel-approval` | Supervisor decisions |

---

## Tasks

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/tasks/schedules`, `/tasks/boards`, `/tasks/boards/stats` | Definitions and boards |
| `GET` | `/tasks/boards/:id/view`, `/tasks/boards/:id/recurring` | Board detail |
| `GET` | `/tasks/assignments?date=`, `/tasks/assignments/me`, `/tasks/my-week` | Assignments |
| `POST` | `/tasks/executions/:id/start` | Start |
| `PATCH` | `/tasks/executions/:id/status` | `{"status": "in_progress"}` |
| `POST` | `/tasks/executions/:id/complete` | `{"completion_type", "notes"}` |
| `POST` `PUT` `PATCH` `DELETE` | `/tasks/schedules[/:id]`, `/tasks/boards[/:id]` | Manage (tlWrite) |
| `POST` | `/tasks/assign`, `/tasks/recurring-assign` | Assign (tlWrite) |
| `GET` | `/tasks/history` | Supervisor view |

---

## Notifications and real time

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/notifications`, `/notifications/unread`, `/notifications/unread/count` | Read |
| `POST` | `/notifications/:id/read`, `/notifications/read-all` | Mark read |
| `POST` | `/notifications/ws-ticket` | Mint a 30-second single-use ticket |
| `GET` | `/notifications/ws?ticket=<ticket>` | WebSocket upgrade |
| `GET` | `/push/public-key` | VAPID public key |
| `POST` | `/push/subscribe` | Register a push subscription |

The socket is a **signal**, not a delivery channel: on any frame the client
refetches `/notifications`. Deduplication happens there, against notification
ids, so an event arriving over both the socket and the poll is shown once.

Upgrades are refused unless the `Origin` header exactly matches an entry in
`CORS_ALLOWED_ORIGINS`.

---

## Files

Uploads are validated by content, never by filename or declared type. Only JPEG,
PNG and GIF are accepted; each is decoded and re-encoded before being stored, so
appended payloads do not survive. The stored name is generated server-side.

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/upload/image` | Upload an image (multipart) |
| `GET` | `/api/uploads/{images,profiles,documents,assignments}/<name>` | Retrieve |

Retrieval requires either the `shiftmaster_uploads` cookie (set at login and
refresh, `HttpOnly`, scoped to `/api/uploads`) or a signed URL carrying `exp` and
`sig`. Unauthorised requests return **404**, so probing cannot confirm that a
file exists.

---

## Other modules

| Area | Endpoints |
|---|---|
| Handovers | `GET POST /handovers`, `PUT /handovers/:id/{claim,unclaim,complete}`, `POST /handovers/:id/comments` |
| Tickets | `GET POST /tickets`, `POST /tickets/:id/comments`, `PUT /tickets/:id/close` |
| Announcements | `GET /announcements[/active|/active-ticker]`, `POST /announcements`, `PUT /announcements/:id/{activate,deactivate}`, `DELETE /announcements/:id` |
| Info tables | `GET POST /info-tables`, `PUT DELETE /info-tables/:id`, rows under `/:id/rows`, `/:id/export`, `/:id/import`, access under `/:id/access` |
| Help docs | `GET POST /help-docs`, `GET PUT DELETE /help-docs/:id`, `GET POST /help-docs/:id/access` |
| FiberX data | `GET POST /fiberx-data`, `GET PUT DELETE /fiberx-data/:id`, `/:id/access`, `/:id/shares` |
| Item requests | `GET /item-requests/{categories,me}`, `POST /item-requests`, `POST /item-requests/:id/cancel`, supervisor `/item-requests/pending` and `/:id/status` |
| External links | `GET /external-links/my-links`, plus management routes (tlWrite) |
| Services | `GET /services/categories`, `/categories/:id/plans`, `/plans/:id`; writes gated by `can_manage_services` |
| Provinces | `GET POST /provinces`, `PUT DELETE /provinces/:id`, shares under `/:id/shares`. Sharing, unsharing and listing shares require **ownership** of the province, not merely visibility of it. |
| Audit | `GET /activity`, `GET /audit` |
| Security | `GET /security/blocked-ips`, `DELETE /security/blocked-ips/:ip` (admin) |

---

## AI assistant

All routes require a normal access token. The assistant runs entirely over
the domain services documented above: the model can only call read tools and
stage *pending actions*; every state change requires the user to approve the
staged action through the endpoints below. Without ASSISTANT_API_KEY the
status endpoint reports `enabled: false` and chat returns 503.

| Method | Path | Notes |
|---|---|---|
| GET | `/api/assistant/status` | `{enabled}` — whether a model is configured |
| POST | `/api/assistant/chat` | Body `{message, conversation_id?}`. Streams SSE events: `status`, `text`, `tool`, `approval`, `error`, `done`. 429 on rate limit, 409 while a previous turn is running |
| GET | `/api/assistant/actions/:id` | Current state of one of the caller's staged actions |
| POST | `/api/assistant/actions/:id/approve` | Executes the exact frozen action after re-validation. Single-use: replays, double clicks and other users get 409 with the action's real state |
| POST | `/api/assistant/actions/:id/reject` | Marks the staged action rejected; nothing executes |

Staged actions expire after ASSISTANT_PENDING_ACTION_TTL_SECONDS (default
5 minutes) and are superseded when a newer action is staged in the same
conversation.

## Health

`GET /health` (outside `/api`) returns `{"status": "healthy"}` and performs a real
database ping. It is what the deploy script gates on.
