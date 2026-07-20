# 10 - Deployment Handbook

## Target Architecture: Oracle Cloud Free Tier (OCI)
The platform is optimized to run on a 2 OCPU, 12GB RAM instance.

### 1. Prerequisites
- Docker Engine installed.
- Current user added to the `docker` group (crucial for unprivileged sandbox execution).
- Nginx installed.
- PostgreSQL 15+ and Redis 7+ installed (can be bare-metal or Dockerized).

### 2. Environment Variables (`.env`)
The backend and worker require the following environment variables in the project root:
```ini
DB_HOST=localhost
DB_USER=nichecp
DB_PASSWORD=strongpassword
DB_NAME=nichecp
REDIS_ADDR=localhost:6379
JWT_SECRET=super-secret-key-change-in-production
SANDBOX_PROVIDER=docker
MAX_EXECUTION_WORKERS=4
```

### 3. Systemd Services
To ensure the backend survives host reboots, create two systemd services.

**/etc/systemd/system/nichecp-api.service**
```ini
[Unit]
Description=NicheCP API
After=network.target postgresql.service redis.service

[Service]
User=dharun
WorkingDirectory=/home/dharun/Desktop/OnlineCodingPlatform/backend
ExecStart=/usr/local/go/bin/go run ./cmd/server/main.go
Restart=always

[Install]
WantedBy=multi-user.target
```

**/etc/systemd/system/nichecp-worker.service**
```ini
[Unit]
Description=NicheCP Worker Daemon
After=network.target redis.service nichecp-api.service

[Service]
User=dharun
WorkingDirectory=/home/dharun/Desktop/OnlineCodingPlatform/backend
ExecStart=/usr/local/go/bin/go run ./cmd/worker/main.go
Restart=always

[Install]
WantedBy=multi-user.target
```

### 4. Reverse Proxy & HTTPS
Nginx should be configured to serve the `frontend/` directory statically on `/`, and `proxy_pass` requests to `/api` to `http://localhost:8080`.
HTTPS must be enforced via Let's Encrypt (`certbot --nginx`).

---

# 11 - Monitoring & Observability

## Current State
- **Logging:** Basic `log.Printf` to stdout/stderr. Systemd captures this into `journalctl`.
- **Metrics:** None.
- **Health Checks:** Basic HTTP connection success on the API.

## Future Roadmap (Production Must-Haves)
1. **Prometheus:** Instrument the Go API and Worker pool using `github.com/prometheus/client_golang` to track:
   - Queue depth (Redis list size).
   - Semaphore token exhaustion rate.
   - 95th percentile execution latency.
2. **Grafana:** Visualize the Prometheus metrics to detect if the worker pool is falling behind the submission rate during a contest.
3. **Alerting:** Configure Grafana to alert (via Slack/Discord) if `active_sandboxes == MAX_EXECUTION_WORKERS` for more than 5 minutes (indicating a potential deadlock or TLE storm).

---

# 12 - Engineering Decisions

### Why Go?
**Problem:** Need high concurrency and low memory usage.
**Solution:** Go's goroutines allow massive parallelization without thread-per-request overhead. The compiled binary uses drastically less RAM than a Node.js or Python backend, saving precious MBs for the Docker sandboxes.

### Why PostgreSQL?
**Problem:** Need ACID compliance and relational integrity for user accounts, problems, and contests.
**Solution:** PostgreSQL is the industry standard for reliable relational data.

### Why Redis?
**Problem:** Synchronous code execution crashes APIs. We need a fast job queue and a distributed lock mechanism.
**Solution:** Redis provides `BRPOP` for instant job queueing and `ZSET` Lua scripts for atomic semaphore operations.

### Why Docker (Currently)?
**Problem:** Need to execute untrusted C++/Python without compromising the host.
**Solution:** Docker provides excellent cgroups and namespace isolation. 
**Tradeoff:** It introduces ~350ms of daemon overhead per execution.

### Why NsJail (Future)?
**Problem:** 100 testcases take 40 seconds due to Docker overhead.
**Solution:** NsJail bypasses the daemon and talks directly to the kernel, executing in ~2ms.
**Tradeoff:** Extremely complex to install and configure (requires unprivileged cgroup delegation), hence it is currently an experimental fallback.

### Why a Worker Pool & Semaphore?
**Problem:** Without limits, a contest spike will spawn 100s of Docker containers, causing an OOM crash.
**Solution:** The worker pool isolates the API from execution. The Semaphore limits absolute global concurrency (across `/api/run` and background workers) to exactly 4, guaranteeing host stability.

### Why LRU & Tmpfs Compiler Caching?
**Problem:** `go build` and `g++` are CPU heavy. Identical submissions waste CPU.
**Solution:** SHA256 hashing the source code and storing the binary in `/dev/shm` (RAM). Drops Go compile time from 12s to 79µs.
