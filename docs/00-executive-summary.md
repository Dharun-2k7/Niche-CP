# 00 - Executive Summary

## Project Vision
**NicheCP** is a production-grade competitive programming platform developed originally for the Amrita Nagercoil ICPC Club. The vision is to provide an end-to-end, locally hosted, ultra-fast online judge that rivals the capabilities of platforms like Codeforces and LeetCode, tailored specifically for university environments and localized contest hosting. 

## Why This Project Exists
Existing platforms (HackerRank, Codeforces, CodeChef) are excellent but pose specific limitations for university clubs:
1. **Lack of Internal Control:** University clubs cannot easily host private, custom-tailored contests without paying enterprise licensing fees.
2. **Data Privacy & Analytics:** Institutions want to own the performance analytics of their students without relying on third-party scrapers.
3. **Execution Latency:** Third-party APIs (like Judge0) introduce network latency and rate limits that disrupt the flow of high-intensity competitive programming environments.

NicheCP solves these problems by providing a self-hosted, scalable execution engine and a sleek, modern UI designed specifically to lower the barrier to entry for competitive programming.

## Key Features
- **God-Mode Aesthetic UI:** A premium, glassmorphic Vanilla JS frontend powered by GSAP animations, offering a visually stunning user experience without the overhead of heavy frameworks.
- **Real-Time Code Execution:** An integrated Monaco Editor allowing users to write, test (`/api/run`), and submit code directly from the browser.
- **Asynchronous Judging Pipeline:** A highly resilient Go-based worker pool backed by a Redis queue ensures that submission spikes during contests do not crash the primary API.
- **Strict OS-Level Sandboxing:** User code is executed in ephemeral Docker containers (and architecturally prepared for NsJail) enforcing hard Memory, CPU, and Time Limits.
- **Role-Based Access Control (RBAC):** Distinct `user`, `admin`, and `superadmin` roles governing contest creation, problem setting, and platform management.

## Technology Stack
- **Frontend:** HTML5, CSS3, JavaScript (ES6+), GSAP (Animations), Chart.js (Analytics), Monaco Editor.
- **Backend API:** Go (Golang 1.20+), Gin Web Framework.
- **Database:** PostgreSQL (Relational persistence).
- **Caching & Message Broker:** Redis (Session caching, Rate Limiting, Job Queues, Distributed Semaphores).
- **Execution Sandbox:** Docker Engine (Current), NsJail (Future).
- **Deployment Target:** Oracle Cloud Infrastructure (OCI) Free Tier (2 OCPU, 12GB RAM) or Azure/GCP equivalent.

## Current Maturity
NicheCP is currently in the **Late Beta / Release Candidate** phase. 
- The core execution engine is functionally complete, having recently been hardened with robust distributed semaphores, singleflight compilation caching, and ultra-low latency RAM-disk caching.
- The UI is polished and fully responsive.
- The platform is structurally prepared for production deployment but currently lacks comprehensive automated testing (Unit/Integration) and observability (Prometheus/Grafana) pipelines.

## Future Roadmap
1. **Immediate (Launch):** Deploy to OCI, configure Nginx/HTTPS, and host the inaugural internal campus contest.
2. **Short-Term (Q3):** Transition the `SandboxProvider` from Docker to NsJail for microsecond kernel-level execution latency to support 100+ testcase bounds efficiently.
3. **Medium-Term (Q4):** Implement the Transactional Outbox pattern to guarantee exactly-once message delivery between PostgreSQL and Redis.
4. **Long-Term:** Horizontally scale the worker nodes across multiple VMs, shard the Redis queue, and implement WebSocket-based live leaderboards.
