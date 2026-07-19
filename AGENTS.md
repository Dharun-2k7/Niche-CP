# AGENTS.md

## 1. Project Overview
**NicheCP** is a production-grade competitive programming platform developed by the Amrita Nagercoil ICPC Club. It provides the software infrastructure necessary to host automated algorithmic contests, manage problem archives, execute code securely in a sandbox, and track user ratings and problem-solving analytics across campus.

## 2. Tech Stack
*   **Frontend**: Vanilla HTML5, CSS3, JavaScript (ES6+). Uses GSAP for animations, Monaco Editor for in-browser coding, and Chart.js for analytics. No heavy framework (like React or Vue) is used by default.
*   **Backend**: Go (Golang) using the Gin web framework.
*   **Database**: PostgreSQL for persistent data (users, problems, submissions, contests).
*   **Caching & Queues**: Redis (used for OTP rate-limiting, session caching, and background job queues).
*   **Execution Engine**: Custom Docker-based sandbox environment that executes C++, Go, Python, and Java code securely with millisecond precision.
*   **Deployment**: Intended for deployment on scalable Linux environments (e.g., Oracle Cloud).

## 3. Folder Structure
*   `frontend/`: Contains all client-side assets (HTML, CSS, JS, images).
    *   `frontend/js/`: Contains modular JavaScript (`app.js`, `animations.js`, `components.js`).
    *   `frontend/css/`: Contains styling (`style.css`).
*   `backend/`: Contains the Go backend application.
    *   `backend/cmd/server/`: The entry point for the backend (`main.go`).
    *   `backend/internal/api/`: API route handlers (Auth, Problems, Submissions, Profile, Admin).
    *   `backend/internal/db/`: PostgreSQL and Redis connection setup and schema execution.
    *   `backend/internal/middleware/`: Custom Gin middleware (CORS, JWT Auth, Admin RBAC).
    *   `backend/internal/auth/`: JWT generation and validation utilities.
*   `worker/` (or related backend modules): Logic for the Docker execution engine consuming Redis queues.
*   `docs/`: Project documentation (Architecture, Setup, Decisions, Changelog).
*   `.ai/`: Persistent AI context and memory tracking files.

## 4. Architecture Overview
*   **Client-Server Model**: The vanilla JS frontend communicates with the Go backend via RESTful APIs over HTTP.
*   **Authentication**: JWT-based authentication (Bearer tokens) stored in `localStorage` on the client.
*   **Role-Based Access Control (RBAC)**: Enforced via middleware and database roles (`admin`, `superadmin`). The superadmin is hardcoded as `dharunkaarthick07@gmail.com`.
*   **Asynchronous Code Execution**: When a user submits code, the backend pushes a job to a Redis queue. A separate worker process (or goroutine) pulls the job, spins up a Docker container, injects the code and test cases, measures execution time/memory, and writes the result back. The frontend polls for submission status.

## 5. Coding Conventions
*   **Go**: Follow standard Go idioms. Use clear, descriptive variable names. Always return and handle errors.
*   **JavaScript**: Use modern ES6+ syntax (`async/await`, `const`/`let`). Avoid generic error swallowing.
*   **HTML/CSS**: Use semantic HTML. Favor CSS Grid and Flexbox. Adhere to the existing design system (glassmorphism, dark theme, specific accent colors like `#ff4081`).
*   **Null Safety**: The backend must handle `nil` slices by initializing them (`make([]Type, 0)`) so they marshal to `[]` instead of `null`. The frontend must always use `try/catch` on fetches and verify `Array.isArray()` before iterating.

## 6. Naming Conventions
*   Go structs and functions: PascalCase for exported, camelCase for internal.
*   Database Columns: `snake_case`.
*   CSS Classes: `kebab-case`.
*   JS Variables: `camelCase`.

## 7. Design Principles
*   **Performance & Reliability**: Zero frontend runtime errors. Network resiliency is critical.
*   **Premium Aesthetic**: The UI must feel state-of-the-art and "wow" the user (glassmorphism, smooth micro-interactions, dark mode, high-contrast accents).
*   **Strict Validation**: Data integrity must be enforced on the backend (e.g., extracting roll numbers strictly from verified `.amrita.edu` college emails, validating JWTs).

## 8. Common Development Commands
*   **Run Backend**: `cd backend && go run ./cmd/server` (or `go build -o server ./cmd/server && ./server`)
*   **Run Frontend**: Serve the `frontend/` directory using any local web server (e.g., Live Server on VS Code, `python3 -m http.server`).

## 9. Documentation & Memory Workflow

We use a layered memory system to preserve long-term project history while keeping the current session context lightweight.

### Startup Behavior
Before starting any new work, automatically read the following to build your context:
1. `AGENTS.md` (This file)
2. `.ai/context.md`
3. `.ai/tasks.md`
4. The most recent archived session from `.ai/sessions/`
5. `docs/decisions.md`

### Continuous Workflow
While working, adhere to these rules:
1.  **`session.md`**: Treat `.ai/session.md` as temporary working memory for the *current session only*. Continuously update it while working. Do not rely on it for long-term project history.
2.  **`context.md`**: Maintain `.ai/context.md` with the current state of the project. Update it *only* when the actual project state changes (completed features, architecture, tech stack, current priorities). Keep it concise and up to date.
3.  **`tasks.md`**: Maintain `.ai/tasks.md` with active tasks, completed tasks, blockers, and next steps. Remove completed items or move them to a completed section.
4.  **`docs/changelog.md`**: Append major milestones and implemented features. *Never* rewrite previous entries.
5.  **`docs/decisions.md`**: Record important architectural or technical decisions along with the reasoning behind them. *Never* delete previous decisions.

### End of Session / Major Milestones
At the end of every session (or after major milestones), archive the current `session.md` as:
`.ai/sessions/YYYY-MM-DD-HHMM.md`
*Never* overwrite previous session archives. Reset `session.md` for the next session.

## 10. Rules for Modifying Existing Code
*   **Understand First**: Read `.ai/context.md` and `.ai/tasks.md` before coding.
*   **Preserve Styling**: Do not break the visual consistency or CSS grid layouts when editing HTML.
*   **Error Handling**: Never leave loose `fetch` calls. Never ignore Go errors.
*   **No Placeholders**: Implement actual logic or provide working demonstrations.
*   **Sync Documentation**: Code and documentation must be perfectly synchronized at the end of every task.

## 11. Files/Directories Requiring Extra Care
*   `frontend/index.html` & `frontend/js/animations.js`: The GSAP logic is tightly coupled to the DOM elements. Do not remove or rename IDs/classes carelessly.
*   `backend/internal/middleware/auth.go`: Critical security pathway.
*   `backend/internal/api/auth.go`: Handles sensitive operations (OAuth, login, JWT issuance, OTPs).
