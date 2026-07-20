# 09 - API Reference

## Authentication

All protected endpoints require a JWT passed in the `Authorization` header.
- **Format:** `Authorization: Bearer <token>`
- **Status Codes:** `401 Unauthorized` if missing/expired. `403 Forbidden` if the user role lacks permissions.

---

## Endpoints

### 1. `POST /api/auth/register`
- **Purpose:** Register a new user.
- **Auth:** None
- **Payload:**
  ```json
  {
    "username": "dharun",
    "email": "user@amrita.edu",
    "password": "securepassword"
  }
  ```
- **Response:** `200 OK` (Success message).

### 2. `POST /api/auth/login`
- **Purpose:** Authenticate and receive a JWT.
- **Auth:** None
- **Payload:**
  ```json
  {
    "email": "user@amrita.edu",
    "password": "securepassword"
  }
  ```
- **Response:** `200 OK` `{"token": "eyJhb..."}`

### 3. `GET /api/problems`
- **Purpose:** Fetch a list of active problems.
- **Auth:** Required (Any Role)
- **Response:**
  ```json
  [
    {
      "id": 1,
      "title": "Two Sum",
      "difficulty": "Easy",
      "time_limit_ms": 2000
    }
  ]
  ```

### 4. `POST /api/submit`
- **Purpose:** Submit code for a problem to the asynchronous queue.
- **Auth:** Required (Any Role)
- **Payload:**
  ```json
  {
    "problem_id": 1,
    "language": "cpp",
    "code": "#include <iostream>..."
  }
  ```
- **Response:** `200 OK` `{"submission_id": 42, "status": "PENDING"}`

### 5. `GET /api/submission/{id}`
- **Purpose:** Poll the status of a specific submission.
- **Auth:** Required (User who owns the submission)
- **Response:**
  ```json
  {
    "id": 42,
    "status": "ACCEPTED",
    "execution_time_ms": 15
  }
  ```

### 6. `POST /api/run`
- **Purpose:** Synchronously compile and execute code against custom inputs (IDE Testing). Blocks until complete.
- **Auth:** Required (Any Role)
- **Payload:**
  ```json
  {
    "language": "python",
    "code": "print(input())",
    "input": "Hello World"
  }
  ```
- **Response:** `200 OK`
  ```json
  {
    "stdout": "Hello World\n",
    "stderr": "",
    "time_exceeded": false
  }
  ```

### 7. `POST /api/admin/contest`
- **Purpose:** Create a new contest.
- **Auth:** Required (`admin` or `superadmin` role)
- **Payload:**
  ```json
  {
    "title": "Weekly Contest 1",
    "start_time": "2026-08-01T10:00:00Z",
    "end_time": "2026-08-01T12:00:00Z"
  }
  ```
- **Response:** `200 OK` `{"contest_id": 5}`
