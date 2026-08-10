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
