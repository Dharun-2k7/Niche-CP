# 08 - Performance Analysis

## Current Bottlenecks

### 1. Docker Startup Overhead
The absolute ceiling on NicheCP's throughput is the OS-level virtualization overhead of the Docker daemon.
- **Benchmark Observation:** Executing a simple "Hello World" in C++ takes ~1ms of CPU time, but the Docker lifecycle (`docker run ...`) takes ~350ms to 400ms.
- **Impact:** For a submission with 100 testcases, this virtualization penalty multiplies to **35 - 40 seconds** of pure overhead. This monopolizes a worker thread and a semaphore token for nearly a minute, drastically reducing platform throughput.
- **Future Optimization:** The architecture is fully prepared (via `SandboxProvider`) to migrate to `NsJailSandbox`, which uses raw kernel `cgroups v2` and namespaces, bypassing the Docker daemon entirely and dropping overhead to `~2ms` per execution.

## Implemented Optimizations

### 1. LRU Binary Caching & Singleflight
- **Problem:** Compiling code (especially C++ and Go) is extremely CPU intensive. During a contest, 50 students might submit the exact same code logic.
- **Optimization:** 
  1. `Singleflight` groups identical concurrent compilations into a single thread.
  2. The `lru.Cache` stores the path to the compiled binary.
- **Benchmark:** A cold Go compile takes ~2.3 seconds. A warm cache hit takes **~79µs**, effectively eliminating compilation overhead for repeat logic.

### 2. Tmpfs RAM Disk
- **Optimization:** `/dev/shm` (which maps directly to system RAM) is used for the artifact directory and the `GOCACHE`. This ensures that file writes (source code) and binary reads (execution) bypass slow SSD/HDD disk I/O entirely.

### 3. Semaphore Architecture
- **Optimization:** The worker pool and the API (`/api/run`) share a strict Redis-based distributed semaphore (`MAX_EXECUTION_WORKERS = 4`). This ensures that no matter the traffic spike, the 12GB Oracle Cloud instance never OOM crashes due to runaway Docker processes.

## Future Optimization Opportunities
- **Redis Pipelining:** If we need to bulk-update leaderboards or fetch statuses, Redis commands should be pipelined.
- **Database Connection Pooling:** Ensure `pgxpool` limits are aggressively tuned to match the OCI instance size.
- **WebSockets:** Currently, the frontend polls `/api/submission/{id}` every second. For 200 users, this is 200 req/sec doing database lookups. Transitioning to WebSockets or Server-Sent Events (SSE) will drastically reduce HTTP overhead and database load.
