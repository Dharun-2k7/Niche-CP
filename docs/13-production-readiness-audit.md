# 13 - Production Readiness Audit

## Overview
This audit evaluates NicheCP as if it were being deployed to production today. It highlights critical blockers and architectural debt that must be addressed for long-term scalability.

### Scorecard
- **Reliability:** 8/10 (Semaphore and Worker pool are highly resilient, but lacks retries on DB failure).
- **Scalability:** 6/10 (Bounded to a single VM currently).
- **Security:** 7/10 (Docker sandbox is secure, but lacks API rate limiting and JWTs are in localStorage).
- **Maintainability:** 8/10 (Clean Go architecture, well abstracted).
- **Performance:** 7/10 (Compiler caching is world-class, but Docker execution overhead is severe).
- **Observability:** 2/10 (Missing Prometheus/Grafana entirely).

## Top Issues

### 1. Missing API Rate Limiting
- **Severity:** Critical (Blocks Production? Yes, if public facing. No, if internal campus only).
- **Description:** There is no rate limiting on `/api/run` or `/api/submit`.
- **Impact:** A malicious user can write a loop to spam the API, instantly exhausting the global 4-token execution semaphore, causing a Denial of Service (DoS) for all other users.
- **Recommendation:** Implement Redis token-bucket rate limiting (e.g., 5 requests / min / IP).

### 2. JWT Storage in LocalStorage
- **Severity:** Medium (Blocks Production? No).
- **Description:** The JWT is stored in `localStorage`, exposing it to potential XSS attacks.
- **Impact:** If an admin injects malicious HTML into a problem description, they could steal student JWTs.
- **Recommendation:** Move JWT to an `HttpOnly`, `Secure` cookie.

### 3. Lack of Transactional Outbox
- **Severity:** Medium (Blocks Production? No, but required for scale).
- **Description:** The API currently writes to PostgreSQL and then `LPUSH`es to Redis. If the process crashes between these two steps, the database says `PENDING`, but the job is never queued.
- **Recommendation:** Implement a Transactional Outbox pattern or a CRON job to sweep stale `PENDING` submissions.

### 4. NsJail Host Provisioning
- **Severity:** High (Blocks Production? No, Docker works. Blocks Scale? Yes).
- **Description:** Processing a 100-testcase submission via Docker takes ~40 seconds.
- **Recommendation:** The OCI host must be provisioned with `nsjail` and unprivileged cgroups v2 to drop execution latency to 2ms per testcase.

---

# 14 - Testing Strategy

## Current State
- The codebase relies predominantly on manual testing and a small validation script (`cmd/validate_sandbox/main.go`).
- **Missing:** Formal Unit Tests, Integration Tests, End-to-End (E2E) UI Tests.

## Suggested Testing Roadmap

### 1. Unit Testing
- Target: `internal/auth` (JWT generation/parsing), `internal/judge` (Compilation hashing, singleflight).
- Tool: Standard `go test`.

### 2. Judge Validation (Integration)
- Ensure the `validate_sandbox` script is wired into a GitHub Action CI pipeline.
- It must test C++, Go, Java, and Python for exact boundaries (AC, TLE at 5.0s, MLE at 256m, RE on Segfault).

### 3. Load Testing
- Target: `/api/submit`
- Tool: `k6` or `Locust`.
- Goal: Fire 500 simultaneous submissions and verify that the API does not drop requests, the Redis queue handles the spike, and the Worker pool never exceeds 4 concurrent Docker containers.

### 4. End-to-End (E2E) Testing
- Target: Frontend UI
- Tool: Playwright or Cypress.
- Goal: Automate a user logging in, navigating to a problem, typing code in the Monaco editor, submitting, and waiting for the GSAP animation to reveal an `ACCEPTED` status.

---

# Final Repository Health Score
| Category | Score / 10 |
| :--- | :--- |
| Product Design | 9 |
| Architecture | 8 |
| Backend Engineering | 8 |
| Frontend Engineering | 9 |
| Database Design | 7 |
| Judge System | 8 |
| Security | 7 |
| Performance | 7 |
| Scalability | 6 |
| DevOps / Observability | 3 |
| Documentation | 10 |

## Top 3 Blockers Before Public Launch
1. **API Rate Limiting:** Must be implemented to prevent Semaphore DoS.
2. **WebSockets/SSE:** Polling the DB every 1s per user will crash Postgres at scale.
3. **HTTPS/Nginx:** Must be securely configured on the OCI host.
