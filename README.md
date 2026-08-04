# NicheCP 🚀

[![Go](https://img.shields.io/badge/Go-Backend-00ADD8?style=flat&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-Database-4169E1?style=flat&logo=postgresql)](https://www.postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-Caching-DC382D?style=flat&logo=redis)](https://redis.io)
[![Docker](https://img.shields.io/badge/Docker-Containerized-2496ED?style=flat&logo=docker)](https://www.docker.com)
[![Nginx](https://img.shields.io/badge/Nginx-Reverse%20Proxy-009639?style=flat&logo=nginx)](https://nginx.org)

**NicheCP** is a production-grade, highly modular, and scalable Competitive Programming Platform developed for the **Amrita Nagercoil ICPC Club**. It provides the complete infrastructure required to host automated algorithmic programming contests, archive problem banks, execute code securely inside containerized sandboxes, and track user ratings and problem-solving analytics campus-wide.

---

## 🏛 System Architecture

The platform follows a decoupled, cloud-native microservices architecture:

```
                       ┌─────────────────────────┐
                       │   Client Web Browser    │
                       └───────────┬─────────────┘
                                   │ HTTPS / WSS
                                   ▼
                       ┌─────────────────────────┐
                       │  Nginx Reverse Proxy    │
                       └─────┬──────────────┬────┘
                             │              │
                             ▼              ▼
                     ┌──────────────┬──────────────┐
                     │ Static Files │  Go REST API │
                     └──────────────┴───────┬──────┘
                                            │
                     ┌──────────────────────┼──────────────────────┐
                     │                      │                      │
                     ▼                      ▼                      ▼
           ┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐
           │  PostgreSQL DB   │   │   Redis Cache    │   │ Execution Worker │
           │ (Users, Problems,│   │ (Jobs, OTP Limit,│   │  (Judge Engine)  │
           │  Submissions)    │   │  Session Cache)  │   │                  │
           └──────────────────┘   └──────────────────┘   └──────────────────┘
```

---

## ✨ Key Features

### 🔒 Authentication & Identity Security
* **Multi-Provider Auth**: Support for traditional credentials, **Google OAuth 2.0**, and **Discord OAuth**.
* **Redis OTP System**: Rate-limited email OTP verification for password resets and email modifications.
* **University Verification**: Automated roll-number extraction and verification for institutional email domains.
* **Codeforces Handle Verification**: Automated verification of Codeforces handles via Codeforces API integration.
* **Role-Based Access Control (RBAC)**: Fine-grained permissions (`student`, `problem-setter`, `admin`, `superadmin`).

### ⚡ Secure Code Execution Engine (The Judge)
* **Multi-Language Support**: Support for popular competitive programming runtimes including C++, C, Python, Go, and Java.
* **Secure Sandbox**: Secure containerized code execution with resource isolation, sandboxing, and optimized execution pipelines.
* **High-Performance Caching**: High-performance caching and optimized compilation workflows.

### 🏆 Contest & Problem Engine
* **Dynamic Lifecycle States**: Automatic transitions between `Upcoming`, `Live`, and `Past` contest states.
* **Scoring Rules**: Supports **ICPC Rules** (Time penalty + submission penalties), **IOI Rules** (Partial testcase scoring), and **Custom scoring**.
* **Anti-Cheat Monitoring**: Tracks and logs real-time user violation events during live contests.
* **Standalone Problem Bank**: Problems exist independently from contests, supporting markdown descriptions, sample/hidden testcase suites, and tag-based discovery.

---

## 🛠 Tech Stack

| Layer | Technologies Used |
|---|---|
| **Frontend** | Vanilla HTML5, CSS3 (Glassmorphism design system), ES6+ JavaScript, Monaco Editor, GSAP Animations |
| **Backend API** | Go (Golang), Gin Web Framework, JWT Auth, OAuth2 |
| **Database** | PostgreSQL (Relational storage, JSONB permissions) |
| **Caching & Queues** | Redis (OTP rate limiting, session storage, task queues) |
| **Execution Sandbox** | Docker Engine API, containerized execution runtime |
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

> ⚠️ **Security Note:** Never commit production environment variables, API secrets, or database credentials to a public repository.

```env
PORT=8080
DATABASE_URL=postgres://user:password@postgres:5432/niche_cp?sslmode=disable
REDIS_URL=redis://redis:6379
BACKEND_URL=https://api.yourdomain.com
FRONTEND_URL=https://yourdomain.com
JWT_SECRET=your_production_jwt_secret_key
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URL=https://api.yourdomain.com/api/auth/google/callback
DISCORD_CLIENT_ID=your_discord_client_id
DISCORD_CLIENT_SECRET=your_discord_client_secret
DISCORD_REDIRECT_URL=https://api.yourdomain.com/api/auth/discord/callback
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=your_email@example.com
SMTP_PASS=your_app_password
```

### 3. Launching with Docker Compose
To deploy the production stack:
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

4. **Run Diagnostic Suite**:
   Run the sandbox diagnostic suite to verify compilers and runtime execution:
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
| `/api/auth/login` | `POST` | Public | Authenticates user credentials and returns JWT token |
| `/api/auth/register` | `POST` | Public | Registers a new user account |
| `/api/auth/google/login` | `GET` | Public | Initiates Google OAuth 2.0 authentication flow |
| `/api/auth/discord/login` | `GET` | Public | Initiates Discord OAuth authentication flow |
| `/api/problems` | `GET` | Public | Lists all problems in the public Problem Bank |
| `/api/contests` | `GET` | Public | Lists all active, upcoming, and past contests |
| `/api/submit` | `POST` | `User` | Submits code for execution and evaluation |
| `/api/profile/upload-dp` | `POST` | `User` | Uploads profile picture with file validation |
| `/api/profile/cf/verify` | `POST` | `User` | Verifies linked Codeforces handle |
| `/api/contests/:id/log` | `POST` | `User` | Logs anti-cheat violation event during contest |
| `/api/admin/problems` | `POST` | `Admin` | Authors new problems and testcase suites |
| `/api/admin/contests` | `POST` | `Admin` | Configures and schedules a new contest |

---

## 📂 Project Structure

```
├── backend/
│   ├── cmd/
│   │   ├── server/           # Backend REST API entry point
│   │   ├── worker/           # Background execution judge worker entry point
│   │   └── validate_sandbox/ # Standalone sandbox verification diagnostic tool
│   ├── internal/
│   │   ├── api/              # HTTP Route Handlers (Auth, Profile, Problems, Contests, Admin)
│   │   ├── auth/             # JWT & OAuth2 authentication logic
│   │   ├── db/               # PostgreSQL & Redis pool connections
│   │   ├── judge/            # Code execution engine and sandbox management
│   │   └── middleware/       # CORS, JWT verification, and RBAC middleware
│   └── Dockerfile            # Multi-stage production container build
├── frontend/
│   ├── js/                   # Modular JavaScript (app.js, animations.js, components.js)
│   ├── css/                  # Styling design system (style.css)
│   ├── nginx.conf            # Reverse proxy & static server configuration
│   └── *.html                # Platform pages (Arena, Contests, Profile, Admin)
├── database/
│   └── schema.sql            # PostgreSQL database tables and indices
├── docs/                     # Platform architectural documentation
└── docker-compose.prod.yml   # Production stack orchestrator
```

---

## 🛡 License & Acknowledgments

Developed by **Amrita Nagercoil ICPC Club**. Built for campus competitive programming excellence and algorithmic contest hosting.
