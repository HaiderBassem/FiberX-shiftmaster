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

## Technical Details

### Backend
The backend is a robust RESTful API built with Go.
* **Language**: Go 1.26
* **Web Framework**: Gin
* **Database**: PostgreSQL (using pgx/v5 driver)
* **Authentication**: JWT (JSON Web Tokens) with Bcrypt password hashing
* **Configuration**: Environment variables (.env)

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

### Prerequisites
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

## Environment Variables
The application relies heavily on environment variables for configuration. Make sure to review `.env.example` to understand all available tuning parameters, such as database connection pool settings, server timeouts, upload limits, and SMTP configurations.