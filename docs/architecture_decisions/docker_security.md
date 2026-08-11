# ADR 005: Isolation of Docker Engine Socket Access

## Status
Accepted

## Context
NicheCP relies on Docker containerization to execute untrusted user-submitted code in an isolated sandbox environment. The Docker Engine socket (`/var/run/docker.sock`) grants control over the host Docker daemon. Giving socket access to Web API containers poses a security risk, as a vulnerability in the API process could lead to host-level compromise.

## Decision
1. **API Container Isolation**: The `api` container shall **NOT** mount `/var/run/docker.sock`.
2. **Worker Container Delegation**: The `worker` container is the **ONLY** service granted socket access for spawning execution sandboxes.
3. **Queue-Based Code Execution**: Synchronous code execution requests (`RunCode`) from the API are enqueued to a Redis `run_queue`. The `worker` process pops jobs, executes code in a single sandbox container per request, and returns execution results via a Redis response key (`run_result:<request_id>`).

## Consequences
- **Security**: The API container is fully decoupled from host Docker control.
- **Scalability**: Code execution workload is isolated from HTTP API request handling.
- **Compliance**: Adheres to principal of least privilege for production deployments.
