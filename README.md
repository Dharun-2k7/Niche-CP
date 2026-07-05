# NicheCP 🚀

A modern, highly modular, and scalable Competitive Programming Platform designed specifically as an ICPC training engine for the Amrita Nagercoil ICPC Club. 

NicheCP provides an independent Problem Bank, a robust dynamic Contest Engine (supporting ICPC, IOI, and Custom penalty systems), and a dedicated Docker-based execution worker for secure code evaluation.

## 🌟 Overview
NicheCP is built to solve the disjointed experience of practicing coding problems. It acts as both a **Practice Hub** and a **Live Contest Arena**. Problems exist independently in a central Problem Bank and can be seamlessly attached to Contests or Practice Sets without data duplication.

---

## ✨ Features
### Authentication & RBAC
- **OAuth / Custom JWT**: Secure authentication flow.
- **Role-Based Access Control (RBAC)**: Strict permission isolation (`user`, `admin`, `sub-admin`, `problem-setter`).
- **Secure Email Verification**: Custom Golang SMTP mailer with Redis-backed OTPs for profile email updates.
- **College ID Integration**: Automatic roll number extraction for university students.

### Contest Engine
- **Dynamic States**: Auto-computes `Upcoming`, `Live`, and `Past` contests based on timestamps.
- **Penalty Support**: ICPC (Time + 20min penalty), IOI (Partial scoring), and Custom scoring options.
- **Live Leaderboard**: Real-time ranking rendering.

### Problem Bank & Problem Setter System
- **Independent Problem Entities**: Problems are standalone and tagged by difficulty and category.
- **Advanced Authoring UI**: Markdown support for Problem Statements, independent fields for I/O formats and Constraints.
- **Testcase Management**: Support for JSON bulk upload of both Sample and Hidden testcases.

### Admin Dashboard
- **User Management**: View registrations, verify college emails, and assign roles.
- **Contest Builder**: Initialize contests and map problems from the central bank.

---

## 🏗 Architecture
The platform is designed for cloud-native deployment (target: Oracle Cloud Free Tier) and split into three core microservices:

1. **Frontend**: Pure Vanilla HTML/CSS/JS. Zero build steps. Utilizes a unified glassmorphism design system (`style.css`), Monaco Editor for code input, and a seamless routing experience.
2. **Backend API**: Golang (Gin Framework). Handles all user interactions, problem CRUD, JWT generation, and SMTP mailing.
3. **Execution Worker (The Judge)**: Background Golang worker that pulls unexecuted code from a Redis queue, spins up isolated Docker containers (sandbox), runs the code against hidden test cases, and updates the PostgreSQL database with the verdict (AC, WA, TLE, RE).

**Database Schema (PostgreSQL)**:
- `users`: Stores RBAC and auth info.
- `problems`: Central independent problem bank.
- `contests`: Contest configurations.
- `contest_problems`: Junction table mapping problems to contests.
- `submissions`: Records every code submission and its status.

---

## ⚙️ Setup Instructions

### Prerequisites
- Docker & Docker Compose
- Go 1.20+
- Node.js (optional, if using a simple frontend server like `serve` or Python `http.server`)

### Environment Variables
Create a `.env` file in the root directory:
```env
JWT_SECRET=super_secret_fallback_key_for_dev
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your_email@gmail.com
SMTP_PASS=your_app_password
DB_URL=postgres://user:password@localhost:5432/niche_cp?sslmode=disable
REDIS_URL=localhost:6379
```

### Run Instructions
1. **Database & Redis**:
   ```bash
   docker-compose up -d postgres redis
   ```
2. **Backend API**:
   ```bash
   cd backend
   go run cmd/server/main.go
   ```
3. **Execution Worker**:
   ```bash
   cd backend
   go run cmd/worker/main.go
   ```
4. **Frontend**:
   ```bash
   cd frontend
   python3 -m http.server 3000
   ```
   *Visit `http://localhost:3000` in your browser.*

---

## 📡 API Overview
| Endpoint | Method | Role | Description |
|---|---|---|---|
| `/api/auth/register` | POST | Public | User signup |
| `/api/auth/login` | POST | Public | JWT Generation |
| `/api/problems` | GET | Public | Fetch Problem Bank |
| `/api/contests` | GET | Public | Fetch all Contests |
| `/api/submit` | POST | User | Submit code to Judge |
| `/api/admin/users` | GET | Admin | Fetch all registered users |
| `/api/admin/problems` | POST | Admin/Setter| Upload new problem |
| `/api/admin/contests` | POST | Admin | Create a new contest |

---

## 👥 Roles Explanation
- **User**: Standard participant. Can solve practice problems and join live contests.
- **Problem-Setter**: Can access `/admin_problems.html` to author new problems and testcases. Cannot assign roles.
- **Sub-Admin**: Can manage contests and monitor leaderboards, but cannot promote other users to Admin.
- **Super-Admin**: Has full system control. Hardcoded fallback: `dharunkaarthick07@gmail.com`.

---

## 🚀 Future Improvements
- [ ] **Plagiarism Detection**: Integrate MOSS or a custom AST-comparison engine.
- [ ] **Rating System**: Implement an Elo-based ranking system (similar to Codeforces) computed post-contest.
- [ ] **ICPC Integration**: Direct integration with Amrita ICPC team formations and mock regional rankings.
- [ ] **Code Golf / Reverse Coding Mode**: Extend the execution worker to support character-count restrictions and hidden logic challenges.
