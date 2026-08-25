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
*   (2026-08-25) Beginner-Friendly Testcase Generation:
    - Replaced the developer-centric generator modal with a Simple Generator visual input builder and an explicit Advanced C++/Python mode.
    - Simple definitions are translated by the API into deterministic, trusted C++ generators and dispatched through the existing Redis → Docker worker pipeline; no sandbox/security boundary changed.
    - Added versioned generator configuration compatibility so legacy code generators and all existing testcase rows continue to work.
*   (2026-08-25) Generator Pipeline Deep Fix & Database Migration:
    - Fixed missing `genCode` / `genLang` DOM elements in Generator Modal, restoring full generator code editing capabilities.
    - Updated `saveProblemConfig()` and `selectProblem()` to save and load `generator_config` persistently in Postgres.
    - Updated worker `psGenerate()` so solution stderr output (warnings) does not cause false `SOL_FAILED` verdicts (only non-zero exit code or timeouts cause failure).
    - Executed SQL migration `001_problem_setter_pipeline.sql` on Postgres container, creating missing `testcases` table and problem setter columns (`status`, `time_limit_ms`, `memory_limit_mb`, `checker_type`, `checker_config`, `generator_config`, `solution_config`, `validator_config`).
*   (2026-08-24) Problem Setter & Judge Pipeline Architecture Upgrade:
    - Queue-Delegated Execution Architecture: Removed all host `os/exec` calls in API handlers; all problem-setter code (generators, validators, solutions, checkers) is compiled and run inside Docker sandbox containers via Redis `problem_setter_queue` consumed by the worker.
    - Docker Sandbox API Extensions: Added `RunWithArgs()` for parameterizable generator/checker execution with command-line arguments and custom timeouts, and `InjectFile()` for writing testcase input files into container tmpfs `/tmp`.
    - Relational Testcase Storage & Management: Created `testcases` database table tracking individual testcases with source attribution (`manual` vs `generated`), generator arguments, input/output, public sample status, and validation status (`valid`, `invalid`, `pending`).
    - Checker Engine Abstraction: Modularized checker engine into `STANDARD` (whitespace-normalized token matching), `FLOATING_POINT` (absolute/relative epsilon comparison), and `CUSTOM` (sandboxed C++/Python custom checker scripts).
    - Polygon Authoring Workspace UX & Workflow Simplification: Restructured `admin_problems.html` layout with a sticky top 6-tab navigation bar (`Overview | Statement | Solution | Checker | Tests | Review`) permanently visible at all times. Completely removed Validator tab and validator publication requirements. Restored contestant-grade Live Preview with KaTeX mathematical notation rendering (`$ ... $` and `$$ ... $$`), difficulty badges, tag pills, time/memory limits, formatted sample testcase boxes, resizable dragging handle, and collapse/expand toggle. Added a Markdown formatting toolbar (**Bold**, *Italic*, Headings, Code, Code Blocks, Inline Math, Block Math, Lists, Tables, Links).
*   (2026-08-20) NicheCP Targeted Product Refinement (4 Core Overhauls):
    - Change 1 (Global Navbar & Profile Avatar Consistency): Standardized profile picture relative pathing (`/uploads/...`) and fallback handling (`onerror`) across all HTML page headers and mobile drawers.
    - Change 2 (NicheCP Resources & 16-Step Visual Roadmap): Redesigned `learn.html` into a visual 16-step competitive programming roadmap (Programming Basics → ... → Advanced CP) with interactive node modals, difficulty badges, state indicators (`LOCKED`, `AVAILABLE`, `IN PROGRESS`, `COMPLETED`), and resource category tabs.
    - Change 3 (NicheCP Journal & Problem Discovery): Transformed `blogs.html` into a technical journal publication hub with editorial category filters and an article reader modal. Added Discovery Shortcuts Bar (*Beginner Collection*, *DP Collection*, *Graph Collection*, *Explore by Topic/Difficulty*) to `arena.html` without duplicating `practice.html`.
    - Change 4 (Codeforces Polygon-Inspired Admin Problem Setter): Overhauled `admin_problems.html` and backend Gin APIs (`/api/admin/problems/validate-input`, `/api/admin/problems/test-checker`, `/api/admin/problems/run-solution`) into a 7-section workspace (**Overview**, **Statement** + PDF Export, **Solution**, **Validator**, **Checker**, **Tests**, **Review**) backed by secure Docker sandbox execution.
*   (2026-08-20) NicheCP Production Readiness & Premium Product Redesign: Completed major redesign and feature upgrades across all core user journeys:
    - Navbar & Avatar Loading Fix: Resolved broken logout references and added image fallback handlers for user profile pictures (`onerror`).
    - Profile Page Refinement: Enhanced email verification OTP flow (2-step endpoints: `/api/profile/email/verify-existing` and `/api/profile/email/verify-new`), added picture upload error handling, and avatar fallback.
    - Problems vs Practice Split: Converted `arena.html` into a dense, efficient, searchable Problem Bank repository; overhauled `practice.html` into a dedicated Training Hub featuring recommended problem cards, curated topic collections (STL, DP, Graphs), difficulty progress metrics, and solved status tracking.
    - Competitive Programming Roadmap: Redesigned `learn.html` into a visual 7-stage learning journey (Fundamentals to Advanced CP) with interactive connected nodes, status badges (`COMPLETED`, `IN PROGRESS`, `AVAILABLE`, `LOCKED`), and an interactive multi-language lesson viewer modal.
    - Knowledge Base & Engineering Editorials: Redesigned `blogs.html` into a technical editorial hub with category pills and an interactive article reader modal.
    - Automated Custom Testcase Generator: Created `POST /api/admin/problems/generate-testcases` backend API executing `mainsol` (Main Solution) and `gen` (Testcase Generator script) in C++, Python, or Go to automatically generate correct testcase pairs `(Input, Output)`. Added interactive modal UI in `admin_problems.html` populating Monaco JSON editors.
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
