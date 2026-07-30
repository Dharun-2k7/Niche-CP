# Architectural Decision Record: Testcase Generator Workflow

## 1. Context and Problem Statement
When admins create algorithmic problems, providing comprehensive and accurate testcases is critical. Manually creating hundreds of test cases is error-prone and time-consuming. We need a reliable system to automatically generate large testcase suites (inputs and expected outputs) based on a generator script and a reference solution.

## 2. Proposed Architecture

### 2.1 Overview
The testcase generator will be a background process integrated into our existing Docker-based sandbox ecosystem. Instead of a single code execution, it will execute a three-step pipeline:
1.  **Generate Inputs**: Run the author's generator script (e.g., Python) to produce raw input data based on predefined parameters (e.g., size, constraints, seeds).
2.  **Generate Outputs**: Feed the generated inputs into the author's reference solution (the "perfect" code).
3.  **Validation & Storage**: Validate the I/O pairs and store them in the backend database (or object storage for large datasets) linked to the `problem_id`.

### 2.2 Components
*   **Admin UI**: A dedicated interface in the problem creation dashboard to upload a generator script, a reference solution, and define parameters (number of test cases, edge cases).
*   **Backend API**: A `POST /api/admin/problems/:id/generate-testcases` endpoint that accepts the scripts and enqueues a job.
*   **Redis Queue**: A dedicated queue (`testcase_generation_queue`) to decouple heavy generation work from the main API.
*   **Worker Node**: The existing worker (or a specialized generator worker) picks up the job.
*   **Docker Sandbox**: 
    *   Container 1: Runs `generator.py` to spit out `input_1.txt`, `input_2.txt`, etc.
    *   Container 2: Compiles `reference_solution.cpp`.
    *   Container 3..N: Runs `reference_solution` against `input_X.txt` to produce `output_X.txt`.

## 3. Storage Strategy
*   Small test cases (< 1MB total) can be stored directly in PostgreSQL as JSON strings (like the current `sample_testcases` column).
*   Large test cases (common in CP) should be zipped and stored on disk (e.g., `/app/uploads/testcases/`) or an S3-compatible Object Storage, with the database holding only the file paths.

## 4. Security Considerations
*   **Time Limits**: Generator scripts can infinite loop. A strict time limit (e.g., 10 seconds for the generator, 2 seconds per reference execution) must be enforced via the sandbox.
*   **Memory Limits**: Generating massive strings can cause OOM. Enforce a 512MB limit on the generator container.
*   **Host Isolation**: Since this runs admin-provided code, it's slightly more trusted than user code, but must still run fully isolated to prevent server corruption.

## 5. Alternatives Considered
*   **Client-Side Generation**: Generating testcases in the admin's browser via WebAssembly. *Rejected* because it's slow, unpredictable, and makes saving the results to the server cumbersome and bandwidth-heavy.
*   **Synchronous API**: *Rejected* because generating 100 testcases takes time and will timeout the HTTP request.

## 6. Next Steps
*   Extend the PostgreSQL schema to handle hidden testcases if it doesn't already.
*   Implement the generation worker logic.
*   Build the Admin UI for the generator.
