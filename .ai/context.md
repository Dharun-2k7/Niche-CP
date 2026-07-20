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
*   (2026-07-20) Contest Arena Real Data & Deletion: Purged all hardcoded/simulated data from the Contest Arena (fake leaderboard, fake commentary, fake submissions, mock WebSocket). All arena panels now fetch from real backend APIs: `GET /api/contests/:id/leaderboard`, `GET /api/contests/:id/submissions`, `GET /api/contests/:id/my-submissions`. Added admin contest deletion workflow with 3 modes (contest_only, selected_problems, all_problems) using database transactions, accessible via `DELETE /api/admin/contests/:id` with a confirmation modal in the admin UI.
*   (2026-07-20) Feature Patch & Bug Fixes: Removed lingering Repovive branding across UI components and styles. Fixed critical bug where all global problems were automatically assigned to newly created contests by adding contest-specific problem fetching (`GET /api/contests/:id/problems`). Added Super Admin role management capability allowing the primary super admin to promote regular users to Admin role via the UI.
*   (2026-07-20) Dashboard UI/UX Redesign: Applied a premium frosted glassmorphism effect to the dashboard containers utilizing deep blur and low-opacity fills to seamlessly integrate with the animated particle background.
*   (2026-07-20) Responsive Architecture Transformation: Overhauled the platform to be fully responsive. Implemented a mobile hamburger navigation drawer, transformed data tables into vertical cards on mobile, applied CSS scroll snapping (`scroll-snap-type`) for horizontal mobile carousels (Bento overview), and optimized the Three.js particle background for mobile (capped at 3,000 particles and `devicePixelRatio` of 1).
*   (2026-07-19) Execution Engine (Phase 1): Upgraded Sandbox to "Compile Once, Execute Many" architecture. Integrated LRU caching with `singleflight` concurrency control to avoid thundering herds. Switched C/C++ and Go to statically linked binaries executing on `alpine` containers for massive latency reductions (~80% faster execution).
*   (2026-07-19) Redesigned the Admin Dashboard with a grid layout and quick actions, completely overhauled Login and Registration pages with a premium split-screen glassmorphic design, and unified the styling of module cards across the platform.
*   (2026-07-19) Implemented multi-step email update flow, added 24-hour password reset cooldown, fixed Solved Count deduplication bug, added standard navigation bar to contests page, standardized Hero CTAs, and added `favicon.svg`.
*   (2026-07-18) Completely revamped the Hero Section UI, standardized Bento grid padding, removed manual Roll Number entry in favor of strict email extraction, integrated DB-driven role checking in `RequireAdmin` middleware, and wrapped all loose `fetch` calls in `try/catch` with horizontal scroll support for data tables.
