# NicheCP Task Tracking

## High Priority Tasks
- [x] **Docker Execution Integration Validation**: Verify end-to-end reliability of the Redis queue and custom Docker execution engine under load. (Phase 1 Latency/Robustness completed)
- [ ] **Execution Engine Throughput (Phase 2)**: Add Redis queue sharding and worker connection pooling.
- [x] **Problem Setter Wizard Implementation**: Transition from JSON-based problem inputs to a human-friendly UI wizard for problem setters, including ability to edit existing problems.
- [ ] **Deployment Environment**: Finalize production database schemas and Docker Compose networks for Oracle Cloud deployment.
- [x] **UI/UX Pro Max Dashboard Redesign**: Implement premium frosted glassmorphism on dashboard containers for better integration with the particle background.
- [x] **Full Responsive Architecture Reflow**: Implement adaptive layouts, mobile nav drawer, horizontal scroll snapping for bento box, and Three.js mobile performance caps across all breakpoints.

## Medium Priority Tasks
- [ ] **Profile Customization**: Finalize Codeforces handle integration and parsing metrics on the profile page.
- [x] **Leaderboard / Rankings**: Real-time contest leaderboards now fetch from `GET /api/contests/:id/leaderboard` with 15s polling.
- [x] **Admin Dashboard Permissions**: Finalize the API endpoints to dynamically update individual user roles and specific granular permissions (Create Problem, Manage Users, etc.) from the frontend modal.
- [x] **Contest Arena Real Data**: Purged all simulated/hardcoded data. All panels (problems, leaderboard, commentary, submissions) fetch from real backend APIs.
- [x] **Contest Deletion Workflow**: Admin can delete contests with 3 modes (contest only, selected problems, all problems) using database transactions.

## Low Priority Tasks
- [ ] **Blogs & Resources**: Implement the placeholder pages (`blogs.html`, `learn.html`) with actual content or database-driven resources.
- [ ] **Dark/Light Theme Persistence**: Currently defaulting to "quantum" dark theme, consider implementing a persistent toggle if requested.

## Technical Debt & Bugs
- [ ] **Concurrency Monitoring**: Ensure the Redis connection pool doesn't exhaust during high traffic.
- [ ] **Mobile Responsiveness Auditing**: Continuously verify complex data grids (like Codeforces analytics) don't overflow on small mobile screens.

## Nice-to-Have Improvements
- [ ] **WebSockets**: Introduce real-time push notifications for submission verdicts to avoid frontend polling.
- [ ] **Integrated IDE Features**: Add Vim bindings or advanced autocompletion to the Monaco Editor instance.
