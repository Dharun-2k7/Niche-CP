# NicheCP Project Context

## Current Project Status
The NicheCP platform is currently in late-stage development/stabilization. A massive, 22-point QA and UI/UX refinement pass was recently completed. The system now features a robust navigation scheme, strict Role-Based Access Control (RBAC), database-driven user roles, null-safe API interactions, and a polished frontend design utilizing glassmorphism and modern web aesthetics.

## Completed Features
*   **Authentication**: Google OAuth and standard Email/Password authentication. Support for OTP verification.
*   **User Profiles**: College email verification specifically restricted to Amrita `.edu` domains. Automatic roll number extraction upon verification.
*   **Role-Based Access Control (RBAC)**: Backend middleware dynamically querying roles from the PostgreSQL database (`admin`, `superadmin`).
*   **Problem Bank & Arena**: Users can view problems, filter by difficulty, and see solved status.
*   **Contest Management**: Admin interface to create and manage contests, including duration, penalty types, and visibility.
*   **Code Execution Engine**: Custom Docker-based sandboxing system (utilizing Redis queues) supporting multiple languages.
*   **Robust Frontend UI**: Comprehensive GSAP animations, responsive grids, and null-safe data fetching. Mobile responsive tables with horizontal scrolling.
*   **Security Architecture**: Two-step email update validation, 24-hour password reset cooldown post-email-change, and strict admin view restrictions on frontend navigation.

## Current Architecture
*   **Frontend**: Vanilla HTML/JS/CSS served statically. Dynamic interactions via `app.js` and `components.js`.
*   **Backend**: Go (Gin) API server connecting to PostgreSQL (main datastore) and Redis (caching and queues).
*   **Execution**: Asynchronous worker pool listening to Redis for code execution tasks, spinning up isolated Docker containers.

## Major Components
*   `frontend/index.html`: Landing page with dynamic hero section and 3D visual elements.
*   `frontend/js/app.js`: Global state management, navigation rendering (conditionally showing Admin tabs), and API fetch logic.
*   `backend/internal/middleware/auth.go`: JWT validation and DB-driven Admin role enforcement.
*   `backend/internal/api/auth.go`: Handles complex user registration and OAuth redirect logic (dynamically detecting frontend referer).

## Important Implementation Details
*   **No Framework**: The frontend purposely avoids React/Next.js to maintain a lightweight footprint, heavily utilizing DOM manipulation.
*   **Null Safety**: The backend ensures JSON arrays are returned as `[]` instead of `null`. The frontend wraps fetch calls in `try/catch` and validates arrays via `Array.isArray()`.
*   **CORS**: Configured dynamically to support local development ports (e.g., `localhost:3000`, `localhost:5500`).
*   **Roll Number Constraints**: Roll numbers are *not* manually entered. They are derived entirely from successful OTP verification against an official college email.

## Current Priorities
*   Monitoring system stability.
*   Preparing for production deployment on Oracle Cloud infrastructure.
*   Refining the Problem Setter UI and integrating custom test case execution.

## Known Issues
*   The connection flow between the main API server and the Docker execution workers requires ongoing monitoring to ensure high concurrency doesn't cause race conditions or memory leaks in Redis.

## Recent Major Changes
*   (2026-08-11) Master Recovery & Production Hardening Audit: Completed a 22-point system recovery audit. Removed host Docker socket (`/var/run/docker.sock`) from `api` container; enqueued `RunCode` requests to Redis `run_queue` consumed by `worker`. Removed all hardcoded personal email addresses from backend and frontend auth/admin logic. Enforced strict contest lifecycle state machine (`CREATED` -> `UPCOMING` -> `RUNNING` -> `ENDED`) with dynamic time calculation. Fixed contest leaderboard propagation to `rankings.html?contest_id=<id>`. Removed all Virtual Contest UI elements. Built full Algorithmic Learning Portal (`learn.html`), Practice Hub (`practice.html`), Blogs (`blogs.html`), and About page (`about.html`). Stored relative `/uploads/filename` paths for profile pictures. Documented ADR 005 and verified security isolation.
*   (2026-08-08) Profile Avatar Smooth Dropdown Menu: Implemented an interactive hover & click dropdown menu for authenticated users attached to the navigation bar avatar trigger button. Features a dark glassmorphic container with custom entrance animations (`cubic-bezier`), user header info (name, email, role badge), quick link to `profile.html`, `admin.html` (for admin roles), and an integrated `Logout` button with red hover state and click dismiss logic.
*   (2026-08-08) Custom Dropdown & Select System: Overhauled select controls and dropdown menus platform-wide. Replaced plain browser default `<select>` elements with dark glassmorphic custom dropdowns featuring `appearance: none`, glowing blue SVG chevron arrow indicators, dark popup option lists (`#0d0f17`), and interactive focus/hover states.
*   (2026-08-08) Mobile Responsiveness & Container Alignment Overhaul: Overhauled the layout engine across all mobile breakpoints (<768px & <480px). Standardized footer containers to 1200px max-width matching `.app-container`, updated `#global-nav-container` to maintain 100% width with 16px side padding, transformed 6-column dynamic table rows (`.mobile-table-row`) into mobile flex cards, converted wide 5-column inline footers into stacked 1-2 column layouts, made `.ranking-row` and profile competitive stats grids responsive, and enforced strict `max-width: 100vw; overflow-x: hidden` to eliminate horizontal scrolling on mobile.
*   (2026-07-30) Post-Docker Functional Audit Fixes: Fixed three critical bugs introduced by the Docker migration. (1) Profile image upload fixed by creating a persistent Docker volume, ensuring `/uploads` directory creation on startup, and configuring Nginx proxy. (2) Fixed a `NOT NULL` Postgres constraint error causing HTTP 500s during contest creation by calculating and inserting `end_time = start_time + duration`. (3) Added full Admin Problem Editing capabilities (`GET /api/admin/problems/:id` and `PUT /api/admin/problems/:id`) along with a dropdown selection UI in the frontend.
*   (2026-07-20) Contest Arena Real Data & Deletion: Purged all hardcoded/simulated data from the Contest Arena (fake leaderboard, fake commentary, fake submissions, mock WebSocket). All arena panels now fetch from real backend APIs: `GET /api/contests/:id/leaderboard`, `GET /api/contests/:id/submissions`, `GET /api/contests/:id/my-submissions`. Added admin contest deletion workflow with 3 modes (contest_only, selected_problems, all_problems) using database transactions, accessible via `DELETE /api/admin/contests/:id` with a confirmation modal in the admin UI.
*   (2026-07-20) Responsive Architecture Transformation: Overhauled the platform to be fully responsive. Implemented a mobile hamburger navigation drawer, transformed data tables into vertical cards on mobile, applied CSS scroll snapping (`scroll-snap-type`) for horizontal mobile carousels (Bento overview), and optimized the Three.js particle background for mobile (capped at 3,000 particles and `devicePixelRatio` of 1).
*   (2026-07-19) Execution Engine (Phase 1): Upgraded Sandbox to "Compile Once, Execute Many" architecture. Integrated LRU caching with `singleflight` concurrency control to avoid thundering herds. Switched C/C++ and Go to statically linked binaries executing on `alpine` containers for massive latency reductions (~80% faster execution).
*   (2026-07-19) Redesigned the Admin Dashboard with a grid layout and quick actions, completely overhauled Login and Registration pages with a premium split-screen glassmorphic design, and unified the styling of module cards across the platform.
*   (2026-07-19) Implemented multi-step email update flow, added 24-hour password reset cooldown, fixed Solved Count deduplication bug, added standard navigation bar to contests page, standardized Hero CTAs, and added `favicon.svg`.
*   (2026-07-18) Completely revamped the Hero Section UI, standardized Bento grid padding, removed manual Roll Number entry in favor of strict email extraction, integrated DB-driven role checking in `RequireAdmin` middleware, and wrapped all loose `fetch` calls in `try/catch` with horizontal scroll support for data tables.
