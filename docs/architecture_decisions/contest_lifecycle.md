# Architecture Decision Record: Contest Lifecycle Enforcement

## Status
Accepted

## Context
The platform needs to support a Codeforces-style contest lifecycle consisting of specific states (CREATED, UPCOMING, RUNNING, ENDED), each with different access rules. Previously, access to the contest arena relied on frontend validations or loosely coupled middleware, allowing potential bypasses (e.g., users directly entering the arena URL before start time or submitting code after the contest ended).

## Decision
We implemented a strict backend-enforced lifecycle model for contests.

### Lifecycle States and Rules
1. **CREATED / UPCOMING**: 
   - Public can view contest details (timer, description).
   - Authenticated users can register.
   - **Restriction**: No user can enter the contest arena, view problems, or submit code.
2. **RUNNING**:
   - Registration is closed.
   - Public can view the live leaderboard (Codeforces-style).
   - Only registered users can enter the contest arena, view problems, and submit code.
   - **Restriction**: Fullscreen mode and anti-cheat tracking are enforced upon arena entry.
3. **ENDED**:
   - Anyone can enter the arena, view problems, and view the leaderboard.
   - **Restriction**: No active contest submissions are allowed (i.e. submissions linked to the `contest_id` are blocked by the backend). Practice submissions must be made outside the contest scope.

### Registration Enforcement
- **Database**: A `UNIQUE(user_id, contest_id)` constraint is enforced in PostgreSQL on the `contest_registrations` table (handled via `ON CONFLICT DO NOTHING`).
- **Endpoint**: Registration uses `POST /api/contests/:id/register`. It strictly validates the contest state, rejecting registrations if the state is `RUNNING` or `ENDED`, or if the current UTC time surpasses the start time.

### Security Boundaries & Middleware
- The middleware `RequireContestLifecycle()` acts as the primary gatekeeper for the arena. It enforces the rules described above.
- It is applied to `/problems`, `/submissions`, `/my-submissions`, and a new `/access` endpoint.
- **Frontend Validation**: The frontend (`contest_arena.html`) now performs an explicit request to `GET /api/contests/:id/access` immediately upon load. Only upon a successful `200 OK` response will it attempt to trigger fullscreen mode and initialize the arena panels. This prevents unauthorized users from rendering the arena locally.

### Clock Synchronization
- To prevent issues with desynchronized client clocks, the `GET /api/contests` endpoint returns a `server_time` (UTC). The frontend computes an offset (`serverTime - localTime`) and applies it to the countdown timer, ensuring all clients observe the contest starting at the exact same moment.

## Consequences
- **Positive**: Complete security against early access or post-contest submissions. Synchronization of countdown timers across all time zones and client devices.
- **Negative**: Adds a slight overhead to the arena initialization process (an extra HTTP check to `/access`), though this is negligible in practice.
