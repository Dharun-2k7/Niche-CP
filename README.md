# NicheCP 🚀

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?style=flat&logo=postgresql)](https://www.postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis)](https://redis.io)
[![Docker](https://img.shields.io/badge/Docker-Containerized-2496ED?style=flat&logo=docker)](https://www.docker.com)
[![Nginx](https://img.shields.io/badge/Nginx-Reverse%20Proxy-009639?style=flat&logo=nginx)](https://nginx.org)

**NicheCP** is a production-grade, highly modular, and scalable Competitive Programming Platform developed for the **Amrita Nagercoil ICPC Club**. It provides the complete infrastructure required to host automated algorithmic programming contests, archive problem banks, execute code securely inside containerized sandboxes with millisecond precision, and track user ratings and problem-solving analytics campus-wide.

---

## 🏛 System Architecture

The platform follows a decoupled, cloud-native microservices architecture designed for deployment on Linux cloud environments (e.g., Oracle Cloud Infrastructure):

```
                       ┌─────────────────────────┐
                       │   Client Web Browser    │
                       └───────────┬─────────────┘
                                   │ HTTPS / WSS
                                   ▼
                       ┌─────────────────────────┐
                       │  Nginx Reverse Proxy    │
                       │  (frontend / SSL / CORS)│
                       └─────┬──────────────┬────┘
                             │              │
                    /uploads │              │ /api/*
                             ▼              ▼
                     ┌──────────────┬──────────────┐
                     │ Static Files │  Go REST API │
                     │ (Uploads/DP) │ (Gin Server) │
                     └──────────────┴───────┬──────┘
                                            │
                     ┌──────────────────────┼──────────────────────┐
                     │                      │                      │
                     ▼                      ▼                      ▼
           ┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐
           │ PostgreSQL 15 DB │   │  Redis 7 Cache   │   │ Docker / NsJail  │
           │ (Users, Problems,│   │ (Jobs, OTP Limit,│   │ Execution Worker │
           │  Submissions)    │   │  Session Cache)  │   │  (Judge Engine)  │
           └──────────────────┘   └──────────────────┘   └──────────────────┘
```

---

## ✨ Key Features

### 🔒 Authentication & Identity Security
* **Multi-Provider Auth**: Support for traditional credentials, **Google OAuth 2.0**, and **Discord OAuth**.
* **Redis OTP System**: Rate-limited email OTP verification for password resets and email modifications.
* **University Verification**: Automated roll-number extraction and verification for `@amrita.edu` institutional emails.
* **Codeforces Handle Verification**: Automated verification of Codeforces handles via custom string matching against the Codeforces API.
* **Role-Based Access Control (RBAC)**: Fine-grained permissions (`student`, `problem-setter`, `admin`, `superadmin`).

### ⚡ Secure Code Execution Engine (The Judge)
* **Multi-Language Support**: Complete compilation and execution pipelines for **C++ (GCC 12)**, **C**, **Python 3.9**, **Go 1.20**, and **Java 17 (Temurin)**.
* **High-Performance Caching**:
  * **RAM Disk LRU Cache**: Pre-compiled binaries are cached in `tmpfs` (`/tmp/nichecp-cache-v3`) with singleflight coalescing to eliminate thundering-herd compile requests.
  * **Container Session Reuse**: Spins up a single hardened container per submission and executes test cases via `docker exec` to minimize startup overhead.
* **Strict Isolation**: Containers run `--network none`, `--read-only` root filesystem, memory ceilings, PID limits (`--pids-limit 50`), and non-root UID enforcement.
* **Experimental NsJail Provider**: Built-in support for ultra-low latency sub-millisecond execution (`SANDBOX_PROVIDER=nsjail`).

### 🏆 Contest & Problem Engine
* **Dynamic Lifecycle States**: Automatic transitions between `Upcoming`, `Live`, and `Past` contest states.
* **Scoring Rules**: Supports **ICPC Rules** (Time penalty + 20-min submission penalties), **IOI Rules** (Partial testcase scoring), and **Custom scoring**.
* **Anti-Cheat Monitoring**: Tracks and logs real-time user violation events (tab switching, window blur, fullscreen escapes) during live contests.
* **Standalone Problem Bank**: Problems exist independently from contests, supporting markdown descriptions, sample/hidden testcase suites, and tag-based discovery.

---

## 🛠 Tech Stack

| Layer | Technologies Used |
|---|---|
| **Frontend** | Vanilla HTML5, CSS3 (Glassmorphism design system), ES6+ JavaScript, Monaco Editor, GSAP Animations |
| **Backend API** | Go (Golang 1.20+), Gin Web Framework, JWT Auth, OAuth2 |
| **Database** | PostgreSQL 15 (Relational storage, JSONB permissions) |
| **Caching & Queues** | Redis 7 (OTP rate limiting, session storage, task queues) |
| **Execution Sandbox** | Docker Engine API, `tmpfs` RAM disk, optional NsJail kernel namespaces |
| **Reverse Proxy** | Nginx (SSL termination, rate limiting, static asset proxy) |

---

## 🚀 Quick Start (Production Setup)

### 1. Prerequisites
Ensure Docker and Docker Compose are installed on your server:
```bash
docker --version
docker compose version
```

### 2. Environment Configuration
Create a `.env` file in the `backend/` directory based on your environment requirements:
```env
PORT=8080
DATABASE_URL=postgres://user:password@postgres:5432/niche_cp?sslmode=disable
REDIS_URL=redis://redis:6379
BACKEND_URL=https://api.nichecp.app
FRONTEND_URL=https://nichecp.app
JWT_SECRET=your_production_jwt_secret_key
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URL=https://api.nichecp.app/api/auth/google/callback
DISCORD_CLIENT_ID=your_discord_client_id
DISCORD_CLIENT_SECRET=your_discord_client_secret
DISCORD_REDIRECT_URL=https://api.nichecp.app/api/auth/discord/callback
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your_email@gmail.com
SMTP_PASS=your_app_password
```

### 3. Launching with Docker Compose
To deploy the entire production stack (Postgres, Redis, API, Worker, and Nginx):
```bash
docker compose -f docker-compose.prod.yml up -d --build
```

To verify running services:
```bash
docker compose -f docker-compose.prod.yml ps
```

---

## 🧪 Local Development Setup

If you wish to run components locally for development:

1. **Start Database & Caching Services**:
   ```bash
   docker compose -f docker-compose.prod.yml up -d postgres redis
   ```

2. **Run Backend API Server**:
   ```bash
   cd backend
   go run ./cmd/server
   ```

3. **Run Execution Worker**:
   ```bash
   cd backend
   go run ./cmd/worker
   ```

4. **Test Sandbox Engine**:
   Run the standalone sandbox validation suite to verify language compilers and container security:
   ```bash
   cd backend
   go run ./cmd/validate_sandbox
   ```

5. **Serve Frontend**:
   Serve the `frontend/` directory using any HTTP server:
   ```bash
   cd frontend
   python3 -m http.server 3000
   ```

---

## 📡 API Architecture Overview

| Endpoint | Method | Auth Required | Description |
|---|---|---|---|
| `/api/auth/login` | `POST` | Public | Authenticates user and returns JWT Bearer token |
| `/api/auth/register` | `POST` | Public | Registers a new user account |
| `/api/auth/google/login` | `GET` | Public | Initiates Google OAuth 2.0 authentication flow |
| `/api/auth/discord/login` | `GET` | Public | Initiates Discord OAuth authentication flow |
| `/api/problems` | `GET` | Public | Lists all problems in the public Problem Bank |
| `/api/contests` | `GET` | Public | Lists all active, upcoming, and past contests |
| `/api/submit` | `POST` | `User` | Submits code for execution and evaluation |
| `/api/profile/upload-dp` | `POST` | `User` | Uploads profile picture with security MIME checks |
| `/api/profile/cf/verify` | `POST` | `User` | Verifies linked Codeforces handle |
| `/api/contests/:id/log` | `POST` | `User` | Logs anti-cheat focus/tab violation during contest |
| `/api/admin/problems` | `POST` | `Admin` | Authors new problems and hidden testcases |
| `/api/admin/contests` | `POST` | `Admin` | Configures and schedules a new contest |

---

## 📂 Project Structure

```
├── backend/
│   ├── cmd/
│   │   ├── server/           # Backend REST API entry point (main.go)
│   │   ├── worker/           # Background execution judge worker entry point
│   │   └── validate_sandbox/ # Standalone sandbox verification diagnostic tool
│   ├── internal/
│   │   ├── api/              # HTTP Route Handlers (Auth, Profile, Problems, Contests, Admin)
│   │   ├── auth/             # JWT & OAuth2 logic
│   │   ├── db/               # PostgreSQL & Redis pool initialization
│   │   ├── judge/            # Docker & NsJail sandbox execution engine & RAM disk cache
│   │   └── middleware/       # CORS, JWT verification, RBAC, and lifecycle checks
│   └── Dockerfile            # Multi-stage production container build for Go API/Worker
├── frontend/
│   ├── js/                   # Modular JavaScript (app.js, animations.js, components.js)
│   ├── css/                  # Styling design system (style.css)
│   ├── nginx.conf            # Reverse proxy & static server configuration
│   └── *.html                # Platform pages (Arena, Contests, Profile, Admin)
├── database/
│   └── schema.sql            # PostgreSQL database tables and indices
├── docs/                     # Comprehensive architecture and deployment documentation
└── docker-compose.prod.yml   # Production stack orchestrator
```

---

## 🛡 License & Acknowledgments

Developed by **Amrita Nagercoil ICPC Club**. Built for campus competitive programming excellence and algorithmic contest hosting.
