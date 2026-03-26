# ShiftMaster API Reference

**Base URL:** `http://localhost:8080/api`
**Content-Type:** `application/json`
**Authorization:** `Bearer <JWT_TOKEN>` (Include in the header for all protected routes)

---

## 🔒 Authentication (Public)
| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `POST` | `/auth/login` | Login | `{"employee_code", "password"}` |

### 🛡️ Authentication (Protected)
| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `GET`  | `/auth/me` | Current user profile | |
| `POST` | `/auth/change-password` | Change own password | `{"old_password", "new_password"}` |
| `POST` | `/auth/reset-password/:id`| Admin reset | `{"new_password"}` (Manager/Admin only) |

---

## 👥 Employees (Manager/Admin for CRUD)
| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `GET`  | `/employees` | List employees | Query: `?status=active&department=IT` |
| `GET`  | `/employees/:id` | Get employee | |
| `POST` | `/employees` | Create employee | `{"employee_code", "first_name", "last_name", "email", "gender" (male/female), "role" (employee/team_leader/manager/hr), "hire_date" (YYYY-MM-DD), "password"}` |
| `PUT`  | `/employees/:id` | Update employee | `{"first_name", "last_name", "phone", "email"}` |
| `PATCH`| `/employees/:id/status`| Change status | `{"status"}` |
| `DELETE`|`/employees/:id` | Delete employee | |

---

## 🏢 Departments (Manager/Admin)
| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `GET`  | `/departments` | List all | |
| `POST` | `/departments` | Create | `{"code", "name", "manager_id"}` |
| `PUT`  | `/departments/:id` | Update | `{"name", "manager_id"}` |

---

## ⏰ Shifts (Manager/Admin)
| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `GET`  | `/shifts` | List shifts | |
| `POST` | `/shifts` | Create shift | `{"shift_code", "name", "start_time" (15:04:05), "end_time", "color_code"}` |
| `PUT`  | `/shifts/:id` | Update shift | `{"name", "start_time", "end_time"}` |

---

## 📅 Schedules
| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `POST` | `/schedules/generate` | Generate week (Mgr) | `{"week_start_date": "YYYY-MM-DD"}` |
| `POST` | `/schedules/:id/publish`| Publish week (Mgr) | |
| `GET`  | `/schedules/daily` | Get day schedule | Query: `?date=YYYY-MM-DD` |
| `GET`  | `/schedules/employee/:id`| Get emp schedule | Query: `?from=YYYY-MM-DD&to=YYYY-MM-DD`|
| `POST` | `/schedules/shifts/:id/check-in` | Check in | |
| `POST` | `/schedules/shifts/:id/check-out`| Check out | |

---

## 🏖️ Leaves
| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `POST` | `/leaves` | Request Leave | `{"leave_type" (annual/sick), "start_date" (YYYY-MM-DD), "end_date", "reason"}` |
| `GET`  | `/leaves/me` | My Leave Req | |
| `GET`  | `/leaves/pending` | Needs Approval | Query: `?role=manager` |
| `POST` | `/leaves/:id/approve/team-leader` | TL Approve | |
| `POST` | `/leaves/:id/approve/manager` | Mgr Approve | |
| `POST` | `/leaves/:id/reject` | Reject Leave | `{"reason"}` |

---

## 🔄 Shift Swaps
| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `POST` | `/swaps` | Request Swap | `{"target_employee_id", "shift_date", "reason"}` |
| `GET`  | `/swaps/me` | My Requests | |
| `GET`  | `/swaps/pending` | Need my answer | |
| `GET`  | `/swaps/pending/manager`| Need Mgr Approval | |
| `POST` | `/swaps/:id/respond` | Emp Accept/Reject| `{"accepted": true/false}` |
| `POST` | `/swaps/:id/approve` | Mgr/TL Approve | |
| `POST` | `/swaps/:id/reject` | Mgr Reject | |

---

## ✅ Tasks
| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `POST` | `/tasks/schedules` | Create task def | `{"title", "description", "schedule_type": "daily_task", "recurrence": "daily", "shift_id"}` |
| `GET`  | `/tasks/assignments` | Daily assignees | Query: `?date=YYYY-MM-DD` |
| `POST` | `/tasks/assign` | Assign employee | `{"schedule_id", "employee_id", "assigned_date"}` |
| `GET`  | `/tasks/assignments/me` | My Tasks | Query: `?date=YYYY-MM-DD` |
| `PATCH`| `/tasks/executions/:id/status`| Change status | `{"status": "in_progress"}` |
| `POST` | `/tasks/executions/:id/complete`| Finish task | `{"notes"}` |

---

## 🔔 Notifications
| Method | Endpoint | Description | Payload |
|---|---|---|---|
| `GET`  | `/notifications` | List all | |
| `GET`  | `/notifications/unread`| List unread | |
| `POST` | `/notifications/:id/read` | Mark read | |
| `POST` | `/notifications/read-all` | Mark all read | |
