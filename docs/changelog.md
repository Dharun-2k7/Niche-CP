# Changelog

All notable changes to the NicheCP project will be documented in this file.

## [Unreleased]

### Added
- **Contest System Lifecycle**: Introduced formal state transitions (`CREATED`, `UPCOMING`, `RUNNING`, `ENDED`) via a background ticker. Added `description`, `status`, and `end_time` to contest configurations.
- **Contest Anti-Cheat System**: Implemented automated monitoring for fullscreen exits and tab switching during active contests. Added a 3-warning limit with automatic disqualification and visual UI alerts in the arena. Blocked copy/paste on problem statements during contests.
- **Admin Contest Management**: Added UI controls for manually updating contest descriptions and states in `admin_contests.html`. Created `admin_violations.html` with a dedicated backend API to monitor contest participant violations.
- **Docker Production Deployment**: Provided full containerization support by creating `backend/Dockerfile`, `frontend/Dockerfile`, and `docker-compose.prod.yml` to spin up PostgreSQL, Redis, the Go API server, worker, and frontend.
- **Testcase Generator ADR**: Documented architectural decisions for building a testcase generation system inside `docs/architecture_decisions/testcase_generator.md`.
- **AGENTS.md**: Persistent AI instruction manual outlining repository structures, conventions, and context rules.
- **.ai/**: Directory containing `context.md`, `tasks.md`, and `session.md` to ensure continuous AI synchronization.
- **docs/**: Comprehensive documentation directory containing architecture guidelines, setup instructions, decision logs, and this changelog.

### Changed
- **Contest Arena Real Data**: Completely purged all hardcoded/simulated data from the Contest Arena page. Removed 15 fake leaderboard entries, 5 fake commentary rows, 3 fake submission rows, and the `addMockCommentary()` mock WebSocket interval. All panels now fetch exclusively from real backend APIs with proper empty state UI.
- **Contest Arena API Expansion**: Added three new backend endpoints: `GET /api/contests/:id/leaderboard` (aggregated ranking by accepted submissions), `GET /api/contests/:id/submissions` (live commentary feed from real submissions), and `GET /api/contests/:id/my-submissions` (user's own contest submissions, auth-protected). All return `[]` for empty datasets.
- **Contest Deletion Workflow**: Implemented `DELETE /api/admin/contests/:id` with three transactional modes: `contest_only` (unlinks problems but preserves them), `selected_problems` (deletes chosen problems), and `all_problems` (deletes all attached problems). Added `GET /api/admin/contests/:id/details` for the confirmation modal. Frontend includes a delete button, radio mode selection, problem checkboxes, and destructive action confirmation.
- **Feature Patch & Bug Fixes**: Removed lingering Repovive branding across UI components and styles. Fixed critical bug where all global problems were automatically assigned to newly created contests by adding contest-specific problem fetching (`GET /api/contests/:id/problems`). Added Super Admin role management capability allowing the primary super admin to promote regular users to Admin role via the UI.
- **Dashboard UI/UX Redesign**: Applied a premium frosted glassmorphism effect (`.premium-glass-panel`) to the Upcoming Contests and Contest Categories containers. This integrates the cards with the animated particle background by using deep blur (`16px`), high saturation, and translucent base colors, mimicking a Spatial UI/Modern Dark aesthetic without affecting dense data tables elsewhere.
- **Responsive Architecture Reflow**: Overhauled `.section-container` spacing with fluid `clamp()` properties, implemented a mobile hamburger drawer navigation, converted complex grids like "Upcoming Contests" into vertically stacked cards on `< 768px`, added horizontal scroll snapping for the Bento grid, and hard-capped Three.js `PARTICLE_COUNT` to 3,000 to prevent thermal throttling on mobile.
- **Execution Engine Refactor (Phase 1)**: Separated the monolithic Sandbox execution into `CompileCode` and `RunArtifact` to achieve "Compile Once, Execute Many". Implemented an LRU Cache (`hashicorp/golang-lru/v2`) bound to 500 entries targeting physical `/dev/shm` deletion, and `singleflight.Group` to completely mitigate thundering herd concurrent compilations.
- Upgraded C, C++ and Go compiler flags to produce statically linked binaries (`-static`, `CGO_ENABLED=0`), allowing execution in ultra-lightweight `alpine` containers (cutting execution overhead by ~80%).
- Redesigned the Admin Dashboard with a grid layout and quick actions, completely overhauled Login and Registration pages with a premium split-screen glassmorphic design, and unified the styling of module cards across the platform.
- Refactored `RequireAdmin` middleware to natively query PostgreSQL for granular database roles (`admin`, `superadmin`), moving away from purely hardcoded superadmin strings.
- Updated `index.html` Hero Section for improved vertical spacing, refined typography, and standardized padding on Bento grid components.
- Modified global `fetch` calls across all dynamic pages (`arena.html`, `contests.html`, `profile.html`, `admin.html`) to enforce strict `try/catch` wrapping and null-safe array checks (`Array.isArray`).
- Updated data tables in admin and profile pages to utilize `overflow-x: auto;` for improved mobile responsiveness.
- Redesigned profile and registration roll-number extraction logic. Roll numbers are no longer input manually but strictly derived from verified `.amrita.edu` OTP responses.

### Fixed
- Fixed Google OAuth URL redirection which was previously failing on non-standard development ports. Redirection now utilizes dynamic frontend URL extraction via the `oauth_referer` cookie.
- Fixed 404 dead links on the global navigation by creating placeholder HTML files (`practice.html`, `learn.html`, `rankings.html`, `blogs.html`) using the existing `coming-soon.html` template.
- Eliminated JavaScript crashing errors inside GSAP animation logic caused by unexpected `null` payloads from API responses.
- Fixed Solved Count duplication bug by using `COUNT(DISTINCT problem_id)` in profile metrics queries.

### Security
- Implemented a secure two-step OTP validation flow for email updates (validates current email first, then new email).
- Enforced a 24-hour cooldown period on password resets following a successful email address change to prevent account hijacking.
- Restricted the frontend Admin Panel navigation logic exclusively to the verified superadmin email.
