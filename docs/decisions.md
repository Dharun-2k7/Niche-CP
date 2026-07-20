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
