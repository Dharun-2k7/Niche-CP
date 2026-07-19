# Architecture Overview

## 1. High-Level System Design

NicheCP is built on a decoupled architecture utilizing a static frontend, a lightweight REST API server, and a specialized background worker for secure code execution.

### Components
1. **Frontend Client**: Vanilla JavaScript, HTML, and CSS. No heavy frameworks. Interacts with the backend via REST endpoints.
2. **API Backend (Gin/Go)**: Serves API endpoints, handles authentication, session management, user roles, and problem/contest data retrieval.
3. **Primary Datastore (PostgreSQL)**: Stores persistent application state (users, problems, test cases, contests, permissions, past submissions).
4. **Cache & Message Broker (Redis)**: Used for ephemeral state (OTP validation) and as a robust job queue to decouple HTTP requests from heavy code execution logic.
5. **Execution Worker (Go + Docker)**: A background process that listens to Redis queues, spins up isolated Docker containers for user code, executes test cases against the code, and records execution metrics (time, memory) and verdicts.

## 2. Request Flow: Code Submission
1. **User Action**: The user submits code from the Monaco Editor in the frontend (`/api/submit`).
2. **API Reception**: The Go backend validates the payload, checks contest constraints (if applicable), and stores an initial "Pending" submission record in PostgreSQL.
3. **Queue Enqueue**: The backend serializes the submission data (code, problem ID, language) and pushes it to a Redis queue.
4. **Worker Dequeue**: A background worker (or separate process) pops the job from Redis.
5. **Execution**:
   - The worker creates a temporary directory mapping.
   - It pulls the required test cases from the database.
   - It executes a predefined `docker run` command with strict CPU, memory, and network constraints, mapping the code file inside.
   - Output from `stdout`/`stderr` is captured.
6. **Verdict Generation**: The worker compares the output against expected output, generates a verdict (AC, WA, TLE, MLE, RE, CE), and computes runtime and memory statistics.
7. **Database Update**: The worker updates the submission record in PostgreSQL with the final verdict.
8. **Client Polling**: The frontend, which has been polling `/api/submissions/:id`, receives the updated verdict and displays it to the user.

## 3. Authentication & Security
- **JWT (JSON Web Tokens)**: Used for all stateless API authentications. Stored in `localStorage` client-side.
- **RBAC**: Database-enforced role checks (Admin, Superadmin) integrated natively as Gin middleware.
- **Data Integrity**: OTP verification restricts roll number assignments securely without trusting raw user input.
- **Sandbox Security**: Docker containers are run without network access (`--network none`), with memory limits (e.g., 512MB), and aggressive timeouts to prevent infinite loops and malicious system calls.
