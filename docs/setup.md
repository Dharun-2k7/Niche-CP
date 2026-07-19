# Setup Guide

## Prerequisites
Before setting up the NicheCP project, ensure you have the following installed on your system:
- **Go**: (1.20+ recommended)
- **Node.js & npm**: (Optional, useful if integrating any build tools later, though currently pure JS is used).
- **PostgreSQL**: (14+ recommended)
- **Redis**: (6+ recommended)
- **Docker**: Required for the execution sandbox engine.

## 1. Database Setup (PostgreSQL)
1. Install PostgreSQL and start the service.
2. Create a new database named `online_coding_platform` (or your preferred name).
3. The backend will automatically run migrations upon startup if they are wired in `main.go` or `db.go`. Ensure your credentials in the `.env` file are correct.

## 2. Cache Setup (Redis)
1. Ensure the Redis server is running locally on the default port (`6379`).
2. No specific schema initialization is needed for Redis.

## 3. Environment Variables
Create a `.env` file in the root directory (or in `backend/`) containing the following critical values:

```env
DB_URL=postgres://postgres:yourpassword@localhost:5432/online_coding_platform?sslmode=disable
JWT_SECRET=your_super_secret_jwt_key
FRONTEND_URL=http://localhost:3000
REDIS_ADDR=localhost:6379

# Google OAuth (If configured)
GOOGLE_CLIENT_ID=your_client_id
GOOGLE_CLIENT_SECRET=your_client_secret
```

## 4. Running the Backend
1. Open a terminal.
2. Navigate to the backend directory:
   ```bash
   cd backend
   ```
3. Fetch dependencies:
   ```bash
   go mod tidy
   ```
4. Run the server:
   ```bash
   go run ./cmd/server
   ```
   The backend should start on `localhost:8080`.

## 5. Running the Frontend
The frontend consists of static files and relies on Vanilla JS. It does not require a bundler by default.
1. Open a terminal.
2. Navigate to the `frontend` directory.
3. Serve the directory using any static file server. 
   - Using Python: `python3 -m http.server 3000`
   - Using Node: `npx serve -p 3000`
4. Access the platform at `http://localhost:3000`.

## 6. Docker Execution Engine
For code submissions to run, Docker must be running on your machine. The Go backend's worker will attempt to pull language images (e.g., `gcc`, `golang`, `python`, `openjdk`) automatically, or you may need to pull them manually:
```bash
docker pull gcc:latest
docker pull golang:latest
docker pull python:3.9
docker pull openjdk:17
```
