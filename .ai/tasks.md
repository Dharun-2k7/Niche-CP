# NicheCP Task Tracking

- [x] **Problem Setter & Judge Pipeline Architecture Upgrade (2026-08-24)**:
  - Security Isolation: Enqueued all problem-setter execution to Redis `problem_setter_queue` consumed by Docker worker process (zero host `os/exec`).
  - Docker Sandbox Extensions: Implemented `RunWithArgs()` for generator args/timeouts and `InjectFile()` for tmpfs input files.
  - Relational Testcase Tracking: Created `testcases` database table with per-test source, generator args, validation status (`valid`, `invalid`, `pending`).
  - Checker Engine Abstraction: Implemented `STANDARD`, `FLOATING_POINT`, and `CUSTOM` checker types with sandbox execution.
  - Polygon Workspace & Lifecycle: Overhauled `admin_problems.html` with 7-section workspace, 10-point review checklist, and `DRAFT` → `READY` → `PUBLISHED` state machine.
- [x] **Targeted Product Refinement (4 Core Overhauls)**:
  - CHANGE 1 (Navbar & Profile Avatar Consistency): Global path resolution (`/uploads/...`) & fallback `onerror` handling.
  - CHANGE 2 (NicheCP Resources & 16-Step Visual Roadmap): 16-node sequential progression roadmap in `learn.html` with difficulty badges, prerequisites, interactive modals, and resource tabs.
  - CHANGE 3 (NicheCP Journal & Problem Discovery): Editorial publication hub in `blogs.html` + Discovery Shortcuts bar (*Beginner*, *DP*, *Graph*, *Topics*, *Difficulty*) in `arena.html`.
  - CHANGE 4 (Codeforces Polygon-Inspired Problem Setter): 7-section workspace (**Overview**, **Statement** + PDF Export, **Solution**, **Validator**, **Checker**, **Tests**, **Review**) in `admin_problems.html` + Go Gin sandbox endpoints (`/api/admin/problems/validate-input`, `/api/admin/problems/test-checker`, `/api/admin/problems/run-solution`).
- [x] **Master System Recovery & Production Hardening**: 22-point audit completed. Removed Docker socket from API, enqueued RunCode jobs to Redis run_queue, eliminated hardcoded emails, enforced strict contest state machine, built Learn/Practice/Blogs/About portals, fixed profile uploads relative paths, documented ADR 005.
- [x] **Master Redesign & Product Readiness Pass**: Completed 25-section overhaul:
  - Fixed Navbar logout and image `onerror` fallback handling platform-wide.
  - Fixed Profile page picture error rendering, avatar fallbacks, and 2-step email OTP verification endpoints.
  - Problems vs Practice Split: Converted `arena.html` into a dense Problem Bank repository and `practice.html` into a Training Hub with recommendations, curated topic collections, and solved stats metrics.
  - CP Learning Roadmap: Redesigned `learn.html` into a visual 7-stage roadmap with interactive status badges and multi-language lesson modals.
  - Knowledge Base & Engineering Editorials: Redesigned `blogs.html` into a technical editorial hub with category filters and full-article reader modal.
  - Automated Testcase Generator Pipeline: Implemented `POST /api/admin/problems/generate-testcases` backend API executing `mainsol` and `gen` scripts to automatically create correct testcase pairs `(Input, Output)` in `admin_problems.html`.
- [x] **Docker Execution Integration Validation**: Verify end-to-end reliability of the Redis queue and custom Docker execution engine under load. (Phase 1 Latency/Robustness completed)
- [x] **Problem Setter Wizard Implementation**: Transition from JSON-based problem inputs to a human-friendly UI wizard for problem setters, including ability to edit existing problems and auto-generate testcases.
- [ ] **Deployment Environment**: Finalize production database schemas and Docker Compose networks for Oracle Cloud deployment.
- [x] **UI/UX Pro Max Dashboard Redesign**: Implement premium frosted glassmorphism on dashboard containers for better integration with the particle background.
- [x] **Full Responsive Architecture Reflow**: Implement adaptive layouts, mobile nav drawer, horizontal scroll snapping for bento box, and Three.js mobile performance caps across all breakpoints.
- [x] **Mobile Container & Navbar/Footer Responsive Alignment Overhaul**: Standardized container widths to 1200px, converted wide inline footers and table rows to adaptive mobile cards, aligned floating glass nav padding, and eliminated all horizontal overflow on small mobile screens.
- [x] **Custom Dropdown & Select Control Engine**: Replaced default browser select inputs across the application with custom dark glassmorphic dropdowns.
- [x] **Profile Avatar Smooth Dropdown Menu**: Built an interactive hover/click glassmorphic dropdown menu for logged-in users.

## Medium Priority Tasks
- [x] **Profile Customization**: Password resets, email verification, 2-step OTP flows, avatar picture upload, and Codeforces handle linking.
- [x] **Leaderboard / Rankings**: Real-time contest leaderboards fetch from `GET /api/contests/:id/leaderboard` with polling.
- [x] **Admin Dashboard Permissions**: Finalize API endpoints to dynamically update user roles and permissions.
- [x] **Contest Arena Real Data**: Purged all simulated/hardcoded data. All panels fetch from real backend APIs.
- [x] **Contest Deletion Workflow**: Admin can delete contests with 3 modes using database transactions.

## Low Priority Tasks
- [x] **Blogs & Resources**: Implemented full Algorithmic Learning Portal (`learn.html`), Practice Hub (`practice.html`), Blogs (`blogs.html`), and About page (`about.html`).

## Technical Debt & Bugs
- [ ] **Concurrency Monitoring**: Ensure the Redis connection pool doesn't exhaust during high traffic.
- [ ] **Mobile Responsiveness Auditing**: Continuously verify complex data grids don't overflow on small mobile screens.

