# 15 - Operations Runbook

This runbook outlines the standard operating procedures (SOPs) for deploying, managing, and troubleshooting the NicheCP platform in a production environment.

---

## 1. Deployments & Updates

### How to deploy a new version
1. SSH into the production server (e.g., OCI instance).
2. Navigate to the project directory: `cd /var/www/OnlineCodingPlatform` (or wherever it is cloned).
3. Pull the latest changes: `git pull origin main`.
4. **Rebuild Backend:**
   ```bash
   cd backend
   go build -o bin/server ./cmd/server
   go build -o bin/worker ./cmd/worker
   ```
5. Restart the `systemd` services to apply the new binaries:
   ```bash
   sudo systemctl restart nichecp-api
   sudo systemctl restart nichecp-worker
   ```
6. Check the status to ensure they started cleanly:
   ```bash
   sudo systemctl status nichecp-api
   ```

### How to update Docker images (for the Sandbox)
The current `SandboxProvider` relies on specific base images (`alpine:latest`, `python:3.9-alpine`, etc.).
1. Pull the latest images to ensure security patches are applied:
   ```bash
   docker pull alpine:latest
   docker pull python:3.9-alpine
   docker pull eclipse-temurin:17-alpine
   ```
2. No restart of the worker is required. The next submission will automatically use the newly pulled local images.

---

## 2. Service Management

### How to restart services
If the system becomes unresponsive or needs a manual cycle:
```bash
# Restart everything
sudo systemctl restart nichecp-api nichecp-worker nginx redis-server postgresql

# Restart only the worker (safe during live contests)
sudo systemctl restart nichecp-worker
```
*Note: Restarting the worker sends a `SIGTERM`. The worker will wait for currently executing sandboxes to finish before exiting cleanly. `PENDING` jobs in Redis are untouched and will be picked up upon restart.*

### How to inspect logs
Using `journalctl` to view `systemd` logs:
```bash
# View live logs for the API
sudo journalctl -u nichecp-api -f

# View live logs for the Worker Daemon (to see compile/execution errors)
sudo journalctl -u nichecp-worker -f

# Search for specific errors (e.g., panics)
sudo journalctl -u nichecp-worker | grep -i "panic"
```

---

## 3. Incident Response & Recovery

### What to do if Redis crashes
**Symptoms:** The frontend gets stuck on `PENDING` forever. `/api/run` requests timeout or return 500s. The worker stops receiving jobs.
**Action:**
1. Restart Redis: `sudo systemctl restart redis-server`.
2. Check Redis logs: `sudo journalctl -u redis-server -e`.
3. Check if the Worker reconnected. If not, restart it: `sudo systemctl restart nichecp-worker`.
*Impact:* Any jobs that were `LPUSH`ed during the crash window were lost. Users must click "Submit" again. 

### What to do if workers stop consuming jobs
**Symptoms:** Submissions pile up as `PENDING` but Redis is online.
**Diagnosis:** The worker pool might be deadlocked, or the global Semaphore might be exhausted by leaked tokens.
**Action:**
1. Check Semaphore state in Redis: 
   ```bash
   redis-cli ZCARD nichecp:execution_semaphore
   ```
   If it returns `4`, and no sandboxes are actively running, tokens leaked.
2. The heartbeat mechanism *should* auto-expire them, but to force a flush:
   ```bash
   redis-cli DEL nichecp:execution_semaphore
   ```
3. Restart the worker daemon to reset the state: `sudo systemctl restart nichecp-worker`.

### How to respond during a contest if something goes wrong
**Golden Rule:** Do not panic. The frontend and backend are decoupled.
1. **API goes down (502 Bad Gateway):** Restart `nichecp-api`. Active workers will continue executing existing jobs in the background safely.
2. **Postgres goes down:** Submissions will fail to save. Restart Postgres immediately. Users must resubmit.
3. **Server running out of memory (OOM):** This usually means `MAX_EXECUTION_WORKERS` was manually set too high. 
   - Edit `.env` and set `MAX_EXECUTION_WORKERS=3`.
   - Restart the worker daemon.

---

## 4. Database & Secrets

### How to recover PostgreSQL & Restore Backups
**Backup Strategy:** You should have a CRON job dumping the DB daily.
```bash
pg_dump -U nichecp -h localhost -d nichecp > /backups/nichecp_$(date +%F).sql
```
**Restore Procedure:**
If the database gets corrupted:
1. Drop and recreate the database:
   ```bash
   dropdb -U postgres nichecp
   createdb -U postgres nichecp -O nichecp
   ```
2. Restore the backup:
   ```bash
   psql -U nichecp -d nichecp -f /backups/nichecp_YYYY-MM-DD.sql
   ```

### How to rotate secrets
If the JWT Secret or Database password is compromised:
1. Edit the `.env` file in the project root: `nano .env`
2. Change `JWT_SECRET=new-super-secure-key`.
3. Restart the API: `sudo systemctl restart nichecp-api`.
*Impact:* Rotating the `JWT_SECRET` will immediately invalidate all active sessions. All users will be logged out and forced to log in again. Do not do this mid-contest unless absolutely necessary.
