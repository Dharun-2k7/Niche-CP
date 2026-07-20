# 01 - Product Requirements

## Product Goals
1. **Host Private Contests:** Enable admins to seamlessly create, manage, and execute algorithmic programming contests.
2. **Reliable Judging:** Ensure deterministic, accurate evaluations of user code against strict time and memory constraints.
3. **Low Friction UX:** Provide a state-of-the-art, visually appealing frontend that does not require heavy client-side processing, ensuring accessibility on low-end hardware.
4. **Actionable Analytics:** Provide detailed feedback on submissions (Time, Memory, Status) and dynamically update leaderboards.

## Target Audience
- **Primary:** Computer Science students and algorithmic programming enthusiasts at the university level.
- **Secondary:** Platform Administrators (Professors, Club Leads) responsible for setting problems and monitoring progress.

## Functional Requirements
- **Authentication:** Users must be able to log in securely using verified credentials (e.g., `.amrita.edu` emails) and maintain session state via JWT.
- **Problem Archive:** Users must be able to view a paginated, filterable list of coding problems.
- **In-Browser IDE:** Users must be able to write code in C++, Go, Java, or Python, and execute it against custom test cases (`/api/run`) before final submission.
- **Contest Management:** Admins must have a dedicated UI to define contest metadata, attach problems, and monitor the event.
- **Asynchronous Evaluation:** Code submissions must not block the main API thread. They must be queued and processed in the background, with the UI polling for real-time status updates (`PENDING` -> `ACCEPTED`).
- **Verdicts:** The system must accurately differentiate between `ACCEPTED`, `WRONG_ANSWER`, `TIME_LIMIT_EXCEEDED`, `RUNTIME_ERROR`, and `COMPILATION_ERROR`.

## Non-Functional Requirements
- **Performance:** 
  - API response times (excluding code execution) must be `< 100ms`.
  - Frontend Time-to-Interactive (TTI) must be rapid, leaning on Vanilla JS and CSS over heavy SPA payloads.
- **Isolation:** Untrusted user code must NEVER have network access or the ability to compromise the host operating system.
- **Scalability:** The execution engine must handle traffic spikes (e.g., 200 users submitting simultaneously at the end of a contest) without crashing the host VM.
- **Reliability:** Background jobs must not be lost if a worker process crashes.
- **Availability:** Target `99.9%` uptime during contest windows.

## Constraints
- **Hardware Limitations:** The platform is architected to run initially on an Oracle Cloud Free Tier instance (2 OCPU, 12GB RAM, ARM/x86).
- **Docker Virtualization:** The current reliance on Docker introduces a ~350ms per-testcase execution overhead, which fundamentally bounds maximum throughput until the planned NsJail migration.

## Future Features
- **Live Leaderboards:** Transitioning from HTTP polling to WebSockets for instant leaderboard updates.
- **Plagiarism Detection:** Integrating an AST-based or token-based code similarity engine (like MOSS) for post-contest analysis.
- **Editorial System:** Allowing admins to publish markdown-based solutions and explanations post-contest.
- **User Ratings:** Implementing an Elo-based rating system to track user skill progression over time.
