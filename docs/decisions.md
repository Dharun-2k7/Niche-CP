# Architecture & Implementation Decisions

This document logs significant technical choices made during the development of NicheCP.

## 1. Vanilla JavaScript vs. React/Vue
- **Decision**: Build the frontend using pure HTML, CSS, and Vanilla JavaScript.
- **Why**: To keep the platform extremely lightweight, minimize build steps, and deeply integrate specialized libraries like GSAP for high-end micro-animations and DOM manipulations without fighting virtual DOM diffing.
- **Alternatives Considered**: React (Next.js) or Vue. These were rejected to prioritize raw performance and lower the learning curve for club members contributing purely UI/UX elements.

## 2. Docker Sandbox execution vs. Judge0 API
- **Decision**: Develop a custom Code Execution Engine using Docker and Redis queues instead of relying on external services like Judge0 or Sphere Engine.
- **Why**: 
  - Absolute control over the execution environment.
  - Zero reliance on external rate limits.
  - Ability to seamlessly integrate custom problem setter constraints and custom validators in the future.
  - Reduced long-term operational costs.
- **Alternatives Considered**: Judge0 (open-source but heavy to self-host), Piston API.

## 3. Redis as a Job Queue
- **Decision**: Utilize Redis to broker execution jobs between the Gin API server and the execution worker.
- **Why**: 
  - Redis provides extremely fast in-memory queues (via lists or pub/sub) with persistence options.
  - Safely decouples the HTTP request lifecycle from the slow process of compiling/executing code, preventing the API from hanging.
- **Alternatives Considered**: RabbitMQ (overkill for simple pub/sub), PostgreSQL LISTEN/NOTIFY (can become a bottleneck under high throughput).

## 4. Roll Number Extraction
- **Decision**: Extract roll numbers dynamically upon verified OTP validation rather than allowing manual input.
- **Why**: Prevents impersonation and enforces a trusted data flow for analytics and leaderboards. By binding the Roll Number strictly to a verified `@xx.amrita.edu` email domain, the platform ensures data integrity.

## 5. JWT for Authentication
- **Decision**: Use stateless JSON Web Tokens (Bearer) stored in `localStorage` rather than server-side sessions.
- **Why**: Stateless architecture scales effortlessly across multiple API instances. It simplifies API interactions and keeps the Go backend fully RESTful and decoupled from the static HTML frontend.

## 6. Explicit Delete Modes vs. CASCADE for Contest-Problem Relationships
- **Decision**: Use explicit transactional delete modes (`contest_only`, `selected_problems`, `all_problems`) rather than relying on PostgreSQL `ON DELETE CASCADE`.
- **Why**: 
  - Contest-Problem is a many-to-many relationship. A problem may belong to multiple contests or exist as a standalone practice problem.
  - CASCADE would blindly destroy problems when a contest is deleted, even if the admin only intended to remove the contest structure.
  - Explicit modes give the admin full control over data preservation, with database transactions ensuring atomicity and rollback on failure.
- **Alternatives Considered**: `ON DELETE CASCADE` (rejected: too destructive), soft deletes with `deleted_at` column (deferred for future consideration).

## 2026-07-22: Single-Node Optimization (Semaphore & Sandbox)
**Context:** NicheCP is deployed on a single Oracle Cloud VM (2 OCPU, 12GB RAM). The previous architecture used Redis for a distributed semaphore and spawned a new Docker container for *every* testcase to guarantee isolation.
**Decision:** 
1. **Local Go Semaphore:** Replaced the Redis distributed semaphore with a local Go buffered channel (`chan struct{}`). Redis was unnecessary for concurrency control on a single VM, and the network/I/O latency of Lua scripts was wasted.
2. **Session-based Execution:** We shifted from "one container per testcase" to "one container per submission". `StartSession` creates a persistent container using `tail -f /dev/null`, and `RunTestcase` uses `docker exec` to run code within it. We run cleanup (`kill $(ps -o pid | tail -n +2 | grep -v '^ *1$')` and `rm -rf /tmp/*`) between testcases to prevent state leakage.
**Reasoning:** Removing the Docker container lifecycle overhead (creation and destruction) per testcase saves ~350-500ms *per testcase*, dramatically increasing throughput. Switching from Redis to a native Go channel eliminates network hops and simplifies the architecture without sacrificing the strict concurrency limit required for a 2-OCPU machine.

## 9. Mobile Container Alignment & Responsive Card Engine (2026-08-08)
- **Decision**: Enforce a strict single max-width container baseline (1200px) across `.app-container`, `.site-footer`, `.footer-grid`, and `.footer-bottom`, paired with CSS-level flex card transformation for dynamic table rows (`.mobile-table-row`).
- **Why**: Hardcoded inline CSS styles (such as `max-width: 1400px` on footers and `display: grid; grid-template-columns: 2fr repeat(4, 1fr)` or `2fr 1fr 1fr 1fr 1fr 1fr`) caused page body elements to expand far past `100vw` on mobile screens. Fixed floating elements like `#global-nav-container` only spanned the initial viewport width, creating a jarring visual mismatch where the floating navbar ended prematurely while the rest of the page overflowed out to 700px+.
- **How**:
  1. Standardized root container width constraints to `max-width: 1200px` with fluid 16px side padding on mobile breakpoints (`@media screen and (max-width: 768px)`).
  2. Overhauled floating navbar (`#global-nav-container` / `.glass-nav`) padding to `0 16px` with `width: 100%`, guaranteeing exact visual alignment with main page content padding on mobile.
  3. Replaced 5-column inline footers with `.site-footer`, `.footer-grid`, and `.footer-bottom` classes that automatically break down into 2-column or 1-column vertically centered stacked sections on mobile.
  4. Transformed multi-column table rows (`.mobile-table-row`, `.contest-row`) into mobile flex cards (`display: flex; flex-direction: column`) with key-value pairings per row item.
  5. Enforced `overflow-x: hidden !important` on `html, body` and added `overflow-x: auto` wrappers for data tables and activity heatmaps.

## 10. Admin UI Problem/Contest Overhaul (2026-08-10)
- **Decision**: Shifted the Problem Setter (`admin_problems.html`) to a Split-Pane Workspace (Monaco Editor for JSON testcases + Markdown Live Preview) and transitioned the Contest Manager (`admin_contests.html`) to a high-density 2-column layout.
- **Why**: The previous single-column wizard layouts lacked the professional workflow expected on platforms like CodeChef and Polygon.
- **How**:
  1. Replaced raw testcase textareas with embedded `monaco-editor` instances for full JSON syntax highlighting.
  2. Implemented a resizable split-pane layout for side-by-side editing and real-time Markdown rendering using `marked.js`.
  3. Reorganized the Contest setup into a logical 2-column flow (Core Config vs Status/Deletion) with premium glassmorphic styling.

## 11. Docker Socket Security Isolation & Database-Backed RBAC (2026-08-11)
- **Decision**: Removed host Docker socket (`/var/run/docker.sock`) volume mount from the Go API container. Enforced queue-based code execution delegation via Redis (`run_queue`) where only the Go Worker process mounts the Docker socket. Removed all hardcoded personal email addresses (`dharunkaarthick07@gmail.com`) from authentication and authorization middleware; privileges are now strictly derived from database `role` values (`admin`, `superadmin`) or an optional `SUPERADMIN_EMAILS` environment variable.
- **Why**:
  1. Prevents potential host-level compromise through the Web API container.
  2. Adheres to least privilege and zero trust principles.
  3. Guarantees clean multi-environment deployment capability without baked-in personal identities.

## 12. Automated Testcase Generator Pipeline & Dedicated Curriculum Structure (2026-08-20)
- **Decision**:
  1. Built a dedicated backend endpoint `POST /api/admin/problems/generate-testcases` that compiles and executes reference Main Solution (`mainsol`) and Generator (`gen`) scripts written in C++, Python, or Go.
  2. The server executes `gen` to produce raw input streams, feeds the input to `mainsol` via stdin to produce expected output, and formats the output into structured testcase JSON pairs `[{"input": "...", "output": "..."}]`.
  3. Separated Problem Bank (`arena.html`) and Training Hub (`practice.html`): `arena.html` acts as a dense, efficient problem bank, while `practice.html` provides curated problem collections (STL, DP, Graphs), difficulty progression metrics, and recommended next problem cards based on solved state.
  4. Transformed `learn.html` into a visual 7-stage Competitive Programming Roadmap (Fundamentals to Advanced CP) with interactive connected nodes, status badges, and multi-language lesson modals.
  5. Converted `blogs.html` into a technical editorial platform featuring category filters and an interactive full-article reader modal.
- **Why**:
  - Replaces manual, error-prone testcase creation with automated generation using reference AC solutions and random generator scripts.
  - Establishes a clear separation between raw problem repository browsing (`arena.html`) and structured algorithmic skill building (`practice.html`).
  - Provides a gamified, visual learning path for students moving from basic syntax to advanced competitive programming algorithms.

## 13. Problem Setter & Judge Pipeline Architecture Upgrade

* **Status**: Accepted
* **Date**: 2026-08-24
* **Context**: 
  NicheCP previously executed problem-setter scripts (generators, validators, solutions, checkers) directly on the API server host via insecure `os/exec` calls. This violated ADR 011 (Docker Socket Security Isolation) and posed severe arbitrary code execution risks. Furthermore, testcases were stored as monolithic JSONB arrays, making individual testcase tracking, validation status, and custom checker integration impossible.

* **Decision**:
  1. **Security Isolation & Queue Delegation**: All problem-setter execution (generators, validators, reference solutions, custom checkers) is strictly delegated from the API container to the Docker worker process via a dedicated Redis queue (`problem_setter_queue`). Zero host `os/exec` calls exist in the API layer.
  2. **Sandbox API Extensions**: Extended `DockerSandboxSession` with `RunWithArgs()` (passing command-line arguments and custom timeouts to generators/checkers) and `InjectFile()` (injecting `input.txt`, `expected.txt`, and `actual.txt` into writable `/tmp` inside read-only container rootfs).
  3. **Individual Testcase Tracking**: Replaced monolithic JSONB arrays with a dedicated `testcases` table tracking source attribution (`manual` vs `generated`), generator arguments, input/expected output, public sample flag, and validation status (`valid`, `invalid`, `pending`). Retained legacy JSONB fallback for zero-downtime backward compatibility.
  4. **Checker Abstraction**: Implemented modular checker engine supporting `STANDARD` (whitespace-normalized token matching), `FLOATING_POINT` (absolute and relative epsilon tolerance), and `CUSTOM` (sandboxed C++/Python custom checker scripts communicating via exit codes and stderr feedback).
  5. **Problem Lifecycle Enforcement**: Enforced `DRAFT` → `READY_FOR_REVIEW` → `PUBLISHED` state transitions with an automated 10-point audit checklist (`/api/admin/problems/:id/review`). Non-published problems are hidden from participants.

* **Consequences**:
  * Complete security containment of user-submitted problem-setter code within isolated Docker containers.
  * Full Polygon-grade problem authoring workflow with automated test generation and validation.
  * Backward compatibility maintained for existing problems while providing scalable relational storage for new problem archives.
