# Changelog

All notable changes to the NicheCP project will be documented in this file.

## [Unreleased]

### Beginner-Friendly Testcase Generation (2026-08-25)
- Replaced the developer-facing generator modal with a guided **Generate Test Cases** experience: test count, visual input fields, ranges, seeds, reference-solution execution, and testcase saving.
- Added Simple Generator support for numbers, strings, arrays, matrices, permutations, pairs/intervals, repeated values, trees, and graphs; C++/Python authoring remains available in Advanced Generator.
- Added versioned generator configuration storage and compatibility for legacy `{code, language}` configurations. Simple definitions are compiled into trusted C++ and still execute through the unchanged Redis/Docker worker pipeline.

### Problem Setter & Judge Pipeline Architecture Upgrade (2026-08-24)
- **Queue-Delegated Problem Setter Architecture**: Replaced all host `os/exec` code execution in API handlers with Redis queue delegation (`problem_setter_queue`) to the Docker worker process, strictly enforcing ADR 011 and closing all potential host arbitrary code execution vectors.
- **Docker Sandbox API Extensions**: Extended `DockerSandboxSession` with `RunWithArgs()` for parameterizable generator/checker execution and `InjectFile()` for writing testcase input files into container tmpfs `/tmp`.
- **Relational Testcase Tracking**: Added `testcases` table tracking per-test source (`manual` vs `generated`), generator arguments, input/expected output, public sample status, and validator status, replacing monolithic JSONB storage while maintaining fallback compatibility.
- **Checker Engine Abstraction**: Implemented modular checker engine supporting `STANDARD` token matching (whitespace normalized), `FLOATING_POINT` (absolute/relative epsilon comparison), and `CUSTOM` (sandboxed C++/Python custom checker scripts).
- **Streamlined Problem Setter Workflow (Validator Removal)**: Completely removed the Validator tab and validator requirement from `admin_problems.html` and publication review checklist. The Polygon workspace sub-navigation features 6 core tabs (`Overview | Statement | Solution | Checker | Tests | Review`). The testcase workflow simplifies to `Generator / Manual Test → Input → Reference Solution → Expected Output → Checker → Ready`. Retained `validator_config` and `validation_status` database columns and API handlers for 100% backward compatibility with existing problems.
- **Polygon Problem Setter Workspace UX Upgrade**: Overhauled `admin_problems.html` layout to feature a fixed, sticky top 6-tab navigation bar that stays permanently visible at all times. Restored full contestant-grade Live Preview pane featuring real-time Markdown and KaTeX mathematical notation rendering (`$ ... $` and `$$ ... $$`), difficulty badges, tag pills, time/memory limits, formatted sample testcase boxes, resizable dragging handle, and collapse/expand toggle. Added a Markdown formatting toolbar for statement creation (**Bold**, *Italic*, Headings, Code, Code Blocks, Inline Math, Block Math, Lists, Tables, Links).

### Generator Pipeline Deep Fix (2026-08-25)
- **Missing genCode/genLang DOM Elements**: The Generator Modal was missing the generator script textarea and language selector entirely, causing `executeGeneratorPipeline()` to always send empty code → API returned HTTP 400 `"Generator code and language are required"`.
- **generator_config Never Saved**: `saveProblemConfig()` was omitting `generator_config` from the config payload, so generator code was never persisted to the database.
- **generator_config Never Loaded**: `selectProblem()` restored `solution_config` and `checker_config` from the DB but never read `generator_config`, leaving the textarea blank after every reload.
- **Worker `Stderr != ""` False Failure**: `psGenerate()` in the worker treated any stderr output from the reference solution as `SOL_FAILED`, causing all tests to fail even when C++ produced correct stdout with harmless warnings.
- **Fix — Frontend (`admin_problems.html`)**: Added `genCode` textarea with C++ example placeholder, `genLang` select, and `genReplaceAll` checkbox to the Generator Modal. `selectProblem()` now restores generator config. `saveProblemConfig()` now persists generator config. `executeGeneratorPipeline()` validates code is non-empty and displays accurate saved/failed counts. `loadTestcasesList()` now shows a loading spinner and proper error messages.
- **Fix — Worker (`worker/main.go`)**: Changed solution failure condition from `Stderr != ""` to `ExitCode != 0`. Stderr warnings no longer terminate tests.
- **Fix — API (`problem_setter.go`)**: `GenerateTests` now returns `failed_count` alongside `saved_count`.


- **Global Navbar & Profile Avatar Consistency**: Standardized relative profile picture URL resolution (`/uploads/...`) and global `onerror` fallback handling across all HTML page headers and mobile drawers.
- **NicheCP Resources & 16-Step Visual Roadmap**: Transformed `learn.html` into **NicheCP Resources** featuring a 16-step sequential competitive programming visual roadmap (Programming Basics → Variables → ... → Advanced CP) with difficulty badges, state badges (`LOCKED`, `AVAILABLE`, `IN PROGRESS`, `COMPLETED`), interactive node modals, and resource category tabs.
- **NicheCP Journal & Problem Discovery Shortcuts**: Redesigned `blogs.html` into **NicheCP Journal** editorial technical press with category filters and article reader modal. Added Discovery Shortcuts Bar (*Beginner Collection*, *DP Collection*, *Graph Collection*, *Explore by Topic/Difficulty*) to `arena.html` without altering the Practice Hub.
- **Polygon-Inspired Admin Problem Setter Workspace**: Restructured `admin_problems.html` and backend Gin APIs (`/api/admin/problems/validate-input`, `/api/admin/problems/test-checker`, `/api/admin/problems/run-solution`) into a 7-section workspace (**Overview**, **Statement** + PDF Export, **Solution**, **Validator**, **Checker**, **Tests**, **Review**) backed by secure Docker sandbox execution.

### Security & Architecture
- **Docker Socket Isolation (ADR 005)**: Removed host Docker socket (`/var/run/docker.sock`) volume mount from Web API container. Refactored synchronous `RunCode` execution to push payloads to Redis `run_queue`, consumed asynchronously by the Go Worker process.
- **Role-Based Privilege Enforcement**: Removed all hardcoded personal email addresses (`dharunkaarthick07@gmail.com`) from authentication middleware (`RequireAdmin`, `RequireSuperAdmin`), profile handlers, admin promote/demote handlers, and frontend navbar rendering. Privileges are now strictly derived from PostgreSQL database `role` values (`admin`, `superadmin`) or an optional `SUPERADMIN_EMAILS` environment variable.
- **Contest Lifecycle & Access Control**: Enforced strict contest state machine (`CREATED` -> `UPCOMING` -> `RUNNING` -> `ENDED`) with dynamic server time calculation. Built `RequireContestLifecycle` middleware enforcing authentication, contest existence, running status, and registration before arena entry.
- **Anti-Cheat & Fullscreen Enforcement**: Fullscreen activates exclusively upon explicit user entry into active contest arenas. Implemented violation tracking (fullscreen exit, tab switch, window blur) with a 3-warning limit leading to disqualification. Disabled copy, cut, context menu, and selection on problem statement panes while retaining standard copy/paste support inside the Monaco Code Editor.

### Added
- **Algorithmic Learning Portal (`learn.html`)**: Complete Dynamic Programming track (Memoization vs Tabulation, 1D DP, 2D Grid DP, Knapsack, LIS) with interactive code templates in C++, Python, and Go, plus module tracks for Graph Algorithms, Data Structures, and Number Theory.
- **Practice Hub (`practice.html`)**: Interactive problem archive with topic filters (DP, Graphs, STL, Math, Greedy), difficulty filters (Easy, Medium, Hard), search input, and direct links to problem arenas.
- **Blogs & Technical Editorials (`blogs.html`)**: Technical knowledge base featuring contest strategy, DP state transition guides, and execution sandbox architecture articles.
- **About & Infrastructure Page (`about.html`)**: Detailed architecture breakdown of NicheCP's Nginx, Go Gin API, Redis, PostgreSQL, and Docker Sandbox worker pipeline.

### Fixed
- **Contest Leaderboard Propagation**: Fixed bug where contest ranking buttons redirected to global standings; ranking buttons now open contest-specific standings (`rankings.html?contest_id=<id>`).
- **Profile Image Relative Pathing**: Uploaded profile pictures are stored as relative URLs (`/uploads/filename`) in PostgreSQL, allowing Nginx to proxy `/uploads/` seamlessly without CORS or mixed-content domain issues.
- **Virtual Contest Removal**: Purged all Virtual Contest UI elements, cards, buttons, and references from the frontend.
- **Dead Link Elimination**: Updated all footer and navigation links across `index.html` and component templates to point to functional pages.

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
- **Next-Gen Button Design System & Magnetic Micro-Animations**: Upgraded all button components platform-wide (`.btn-primary`, `.btn-magnetic`, `.btn-ghost`, `.btn-secondary`, `.btn-enter`, `.btn-danger`). Features multi-stop dynamic gradients (`#2563EB` -> `#3B82F6` -> `#6366F1`), metallic shimmer light flares (`::before` flare slider), multi-layered ambient neon box-shadows, press scaling (`translateY(1px) scale(0.98)`), and GSAP-powered magnetic cursor tracking in `animations.js`.
- **Custom Dropdown & Select Design System**: Replaced default browser select inputs across the platform with a custom dark glassmorphic dropdown system (`appearance: none`, custom glowing blue chevron SVG indicator icon, dark popup options `#0d0f17`, glowing focus borders, and hover micro-animations). Connected live search and difficulty filtering in `arena.html`.
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
