# FiberX Shiftmaster

## Overview
Shiftmaster is a comprehensive shift and workforce management system designed to handle employee schedules, daily handovers, leave requests, task assignments, and internal communications. It provides an all-in-one platform for employees, team leaders, managers, and administrators to orchestrate their daily operations efficiently.

## Core Features
* **Authentication & Authorization**: Role-based access control (Admin, Manager, Team Leader, Employee) with JWT-based security.
* **Employee Management**: Manage profiles, departments, permissions, and shift schedules.
* **Shift Handovers**: Seamlessly transfer responsibilities between shifts with detailed handover boards.
* **Task Management**: Create, assign, and track one-off and recurring tasks.
* **Leave Management**: Request, approve, and track hourly or daily leaves and balances.
* **Calendar & Scheduling**: Visual representation of shifts, assignments, and time-offs.
* **Dynamic Information Tables (References)**: Customizable data tables for tracking assets, contacts, and custom records with granular access control.
* **Info Bank**: Centralized knowledge base for sharing documents and procedures.
* **Announcements**: Broadcast important messages across departments or globally.
* **External Modules**: Integration with external tools via managed external links.
* **Audit Logging**: Tracking sensitive actions for security and compliance.
* **AI Assistant**: A natural-language assistant on the Home page that understands Iraqi Arabic, formal Arabic, English and any mix of them, answers from live ShiftMaster data through permission-checked tools, and stages any change for the person to approve. It runs a language model **on the ShiftMaster server itself** — no cloud AI service, no API key, no per-message cost, and no employee data leaving the premises.

## Technical Details

### Backend
The backend is a robust RESTful API built with Go.
* **Language**: Go 1.26
* **Web Framework**: Gin
* **Database**: PostgreSQL (using pgx/v5 driver)
* **Authentication**: JWT (JSON Web Tokens) with Bcrypt password hashing
* **Configuration**: Environment variables (.env)

### Local AI
The assistant's language model runs on the same machine as the API.
* **Runtime**: llama.cpp (`llama-server`) on loopback or a Unix socket
* **Model**: a quantised GGUF chosen to fit the server, installed once by `deploy/provision-ai.sh`
* **Grounding**: the model may only call the assistant's tools; it has no SQL access and cannot execute a change without human approval
* **Network**: the API refuses to start if the runtime address is not local, so there is no configuration that sends a prompt off the machine

### Frontend
The frontend is a modern, responsive Single Page Application (SPA).
* **Library**: React 18+
* **Language**: TypeScript
* **Build Tool**: Vite
* **Styling**: Tailwind CSS (with custom HSL theme variables and dark mode support)
* **State Management**: Zustand (Global state) & React Query (Server state)
* **Routing**: React Router DOM
* **Drag & Drop**: @dnd-kit (used for dynamic layouts like the References grid)

## Directory Structure
* `/cmd/api/` - Application entry point (main.go) and HTTP handlers.
* `/internal/` - Core backend logic including models, repository (database access), services, and middleware.
* `/internal/database/migrations/` - SQL migration files for setting up the PostgreSQL database schema.
* `/pkg/` - Reusable backend packages (e.g., database connection, token generation).
* `/frontend/` - Contains the entire React application.
  * `/src/components/` - Reusable UI components.
  * `/src/features/` - Domain-specific modules (handovers, tasks, calendar, etc.).
  * `/src/hooks/` - Custom React hooks.
  * `/src/store/` - Zustand state stores.
  * `/src/services/` - API client configurations and endpoints.

## How to Run

There are two ways in. Docker gives you the whole system — database, migrations,
API and web tier — from one command, and is what CI exercises on every push. The
manual setup below stays the reference for developing against the code directly.

### Run with Docker

```bash
cp deploy/docker.env.example .env.docker
# Fill in JWT_SECRET; the API refuses to start on a placeholder.
openssl rand -hex 32

docker compose up -d --build
```

The app is then on <http://localhost:8080> (set `WEB_PORT` to publish elsewhere).
Only the web container is published: the API and PostgreSQL stay on the compose
network, reachable through Caddy at `/api`, which also means the browser makes no
cross-origin request and needs no CORS configuration.

Migrations are a separate one-shot service that must finish before the API
starts, so the API never meets a schema it does not understand. Application data
lives in the `pgdata` volume and uploads in `uploads`; `docker compose down`
leaves both alone, and `down -v` deletes them.

The assistant is not included by default — it needs a multi-gigabyte model that
does not belong in an image, and without one it simply reports itself
unavailable. To add it, provision a model (see below) and layer the overlay:

```bash
AI_MODEL_FILE=$HOME/.shiftmaster/models/model.gguf \
  docker compose -f docker-compose.yml -f docker-compose.ai.yml up -d --build
```

The model server runs beside the API in the same container rather than in one of
its own. That is deliberate: the API refuses to start when `AI_BASE_URL` is
anything but loopback or a Unix socket, so that no deployment can quietly point
the assistant at a hosted inference service.

### Prerequisites (manual setup)
* Go 1.26 or higher
* Node.js 18 or higher (with npm)
* PostgreSQL 14 or higher

### 1. Database Setup
1. Create a new PostgreSQL database (e.g., `shiftmaster`).
2. Run the SQL migration files located in `internal/database/migrations/` in ascending order (from 001 to the latest) to build the schema.

### 2. Backend Setup
1. Navigate to the root of the project.
2. Copy the example environment file to `.env`:
   ```bash
   cp .env.example .env
   ```
3. Update the `.env` file with your database credentials (`DB_USER`, `DB_PASSWORD`, `DB_NAME`, etc.) and a secure `JWT_SECRET`.
4. Install Go dependencies:
   ```bash
   go mod download
   ```
5. Start the backend server:
   ```bash
   go run cmd/api/main.go
   ```
   The backend will start running on the port specified in `.env` (default is 8080).

### 3. Frontend Setup
1. Open a new terminal and navigate to the frontend directory:
   ```bash
   cd frontend
   ```
2. Install Node dependencies:
   ```bash
   npm install
   ```
3. Start the development server:
   ```bash
   npm run dev
   ```
   The application will be accessible at `http://localhost:3000` (or the port specified by Vite).

### 4. AI Assistant Setup (optional, one-time)
The assistant is enabled by default but needs a model before it can answer. On the
server:

```bash
sudo ./deploy/provision-ai.sh
```

It inspects the CPU, RAM and GPU, picks the strongest model that machine can run
reliably, downloads it once, proves it loads, and prints the `AI_*` settings to
paste into `.env`. Restart the API afterwards.

Until then — and any time the model is down — the Home page shows the assistant
with an honest "starting up" or "unavailable" state, and every other part of
ShiftMaster works exactly as before.

To evaluate the model on the machine it will run on (real prompts, real database,
Iraqi Arabic, adversarial and prompt-injection cases):

```bash
AI_EVAL=1 SHIFTMASTER_TEST_DB=shiftmaster_test go test ./internal/assistant -run TestEval -v
```

## Environment Variables
The application relies heavily on environment variables for configuration. Make sure to review `.env.example` to understand all available tuning parameters, such as database connection pool settings, server timeouts, upload limits, and SMTP configurations.