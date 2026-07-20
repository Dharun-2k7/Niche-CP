# 03 - Backend Architecture

## Purpose
The NicheCP backend is written in Go (Golang) and utilizes the Gin web framework. It is designed to be highly concurrent, memory-safe, and capable of robustly handling untrusted user input before passing it to the execution engine.

## Folder Structure & Packages

### `cmd/server/`
- **Purpose:** The entry point for the main HTTP API server.
- **Important Files:** `main.go`
- **Responsibilities:** Initializes the Gin router, connects to PostgreSQL and Redis, applies global middleware (CORS), and mounts the API routes. Starts the HTTP listener.

### `cmd/worker/`
- **Purpose:** The entry point for the background job processing daemon.
- **Important Files:** `main.go`
- **Responsibilities:** Connects to Redis and blocks on `BRPop` for the `submissions_queue`. It spawns a bounded number of worker goroutines to compile and execute submissions, updating the PostgreSQL database with the final verdicts.

### `cmd/benchmark_workers/` & `cmd/validate_sandbox/`
- **Purpose:** Utility scripts for testing backend resilience.
- **Responsibilities:** 
  - `benchmark_workers/main.go`: Simulates massive bursts of submissions to test the semaphore limits and worker pool starvation constraints.
  - `validate_sandbox/main.go`: Provides a side-by-side A/B testing framework for comparing `DockerSandbox` against `NsJailSandbox`.

### `internal/api/`
- **Purpose:** Contains all HTTP route handlers (Controllers).
- **Important Files:** 
  - `auth.go`: Handles login, JWT generation, and OAuth validation.
  - `problems.go`: CRUD operations for problems and fetching testcases.
  - `submissions.go`: Enqueues code payloads to Redis and polls status.
- **Error Handling:** Handlers must return standard JSON structures (`{"error": "message"}`). All database errors are logged, but raw SQL errors are abstracted from the client.

### `internal/auth/`
- **Purpose:** Security utilities.
- **Responsibilities:** JWT generation (RS256/HS256) and parsing, Bcrypt password hashing and verification.

### `internal/db/`
- **Purpose:** Database and Cache initialization.
- **Important Files:** `db.go`, `redis.go`
- **Responsibilities:** Maintains the global `*sql.DB` and `*redis.Client` connection pools. The database initialization script automatically runs lightweight schema creations (tables like `users`, `roles`, `submissions`) using `CREATE TABLE IF NOT EXISTS` if they do not exist.

### `internal/judge/`
- **Purpose:** The core Execution Engine abstractions.
- **Important Files:**
  - `sandbox.go`: Defines the `SandboxProvider` interface, `DockerSandbox`, and the experimental `NsJailSandbox`. Contains the `CompileCode` function utilizing LRU caches and Singleflight to prevent redundant compilations.
  - `semaphore.go`: Implements the Redis ZSET distributed semaphore (`AcquireExecutionToken`, `ReleaseExecutionToken`, `KeepAliveToken`) to globally cap the number of active Docker containers across both the Worker and `/api/run` endpoints.

### `internal/middleware/`
- **Purpose:** Gin HTTP interception layer.
- **Important Files:** `auth.go`
- **Responsibilities:** Extracts the Bearer token, validates the JWT, and enforces Role-Based Access Control (RBAC). Identifies `superadmin` boundaries.

## Important Interfaces
- `SandboxProvider` (in `internal/judge/sandbox.go`): The abstraction that isolates the worker loop from the specific containerization technology. 

## Design Decisions
- **Why Go?** Go provides exceptional concurrency primitives (goroutines, channels) making it trivial to build a highly parallel worker pool. Its compiled nature also yields low memory footprints suitable for the OCI 12GB RAM limit.
- **Why Gin?** It is faster than net/http and provides an easy-to-use middleware routing chain perfectly suited for our RBAC system.
- **Why Separate the Worker?** Compiling C++ code and running Docker containers are CPU and Memory intensive operations. If this were done synchronously in the API handlers, a surge of submissions would instantly exhaust HTTP request threads and crash the API. Decoupling via Redis ensures the API remains responsive (returning `PENDING` instantly) while the worker chews through the backlog.
