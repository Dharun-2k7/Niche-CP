# 07 - Security Audit

## Audit Overview
This document evaluates the security posture of NicheCP. Given that the core functionality involves executing arbitrary, untrusted, and potentially malicious code on the host server, security is the paramount concern of the architecture.

## 1. Code Execution Sandbox Security
- **Isolation Mechanism:** Docker containers (moving to NsJail).
- **Network Isolation:** `--network none` is enforced. Malicious payloads cannot download malware, execute DDoS attacks, or exfiltrate environment variables to a webhook.
- **Filesystem Constraints:** The workspace (`/workspace`) is mounted as `--read-only` and `--tmpfs /tmp` is used. A payload cannot modify the host or write persistent malware to disk.
- **Privilege Escalation:** `--security-opt no-new-privileges` prevents the use of `setuid` binaries (like `sudo`). The user is strictly forced to UID `1000:1000`.
- **Resource Exhaustion (Denial of Service):** 
  - *Fork Bombs:* Prevented via `--pids-limit 50`.
  - *Memory Exhaustion:* Prevented via `--memory 256m` (cgroups trigger SIGKILL).
  - *CPU Hogs:* Prevented via `--cpus 1.0` and a strict 5.0-second `context.WithTimeout`.
- **Vulnerability Status:** Highly Secure. Container escapes require a 0-day Linux kernel vulnerability.

## 2. Authentication & Authorization
- **JWT Storage:** Tokens are currently stored in `localStorage`. 
  - *Severity: Medium.* `localStorage` is vulnerable to Cross-Site Scripting (XSS). 
  - *Recommendation:* Migrate JWT storage to an `HttpOnly`, `Secure` cookie.
- **Passwords:** Hashed securely using `golang.org/x/crypto/bcrypt`.
- **Authorization:** `internal/middleware/auth.go` robustly validates JWT signatures and strictly enforces RBAC (`admin`, `superadmin`). Path protection is secure.

## 3. Web Vulnerabilities
- **SQL Injection:** Mitigated. The Go backend exclusively uses parameterized queries (e.g., `db.DB.Exec("UPDATE ... WHERE id = $1", id)`).
- **XSS (Cross-Site Scripting):** Mitigated in code display via Monaco editor escaping. However, if admins can inject raw HTML into problem descriptions without sanitization, Stored XSS is possible.
  - *Severity: Low/Medium.* 
  - *Recommendation:* Implement a markdown sanitizer (e.g., DOMPurify) on the frontend when rendering problem descriptions.
- **CSRF (Cross-Site Request Forgery):** Because the API relies on `Authorization: Bearer <token>` rather than cookies, it is inherently immune to standard CSRF attacks.

## 4. API Abuse & Rate Limiting
- **Rate Limiting:** Currently, there is no strict IP-based rate limiting on the `/api/run` or `/api/submit` endpoints.
  - *Severity: High.* A malicious user could write a script to hammer `/api/run`, exhausting the 4-token execution semaphore and preventing legitimate users from submitting code (Denial of Service).
  - *Recommendation:* Implement a Redis-based sliding window rate limiter middleware (e.g., max 5 submissions per minute per user).

## 5. Secrets Management
- **Environment Variables:** All secrets (Database credentials, JWT keys, Redis URLs) are correctly abstracted into a `.env` file and excluded from version control via `.gitignore`.
- **Vulnerability Status:** Secure.

## Vulnerability Summary & Priorities

| Vulnerability | Severity | Impact | Recommendation |
| :--- | :--- | :--- | :--- |
| **Missing API Rate Limiting** | High | Semaphore exhaustion (DoS) | Implement Redis-based IP/User token bucket rate limiting on execution endpoints. |
| **JWT in localStorage** | Medium | Token theft via XSS | Migrate to `HttpOnly` cookies. |
| **Problem Description XSS** | Low | Admin-initiated XSS | Sanitize problem description HTML before rendering. |
