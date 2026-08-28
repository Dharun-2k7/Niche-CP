CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- Nullable for OAuth users
    role VARCHAR(50) DEFAULT 'student',
    roll_no VARCHAR(50),
    batch VARCHAR(50),
    college_email VARCHAR(255),
    is_college_verified BOOLEAN DEFAULT FALSE,
    pending_email VARCHAR(255),
    email_verified BOOLEAN DEFAULT FALSE,
    cf_handle VARCHAR(255),
    cf_verify_string VARCHAR(255),
    is_cf_verified BOOLEAN DEFAULT FALSE,
    profile_picture_url VARCHAR(255),
    discord_id VARCHAR(255) UNIQUE,
    discord_username VARCHAR(255),
    permissions JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_email_change TIMESTAMP
);

CREATE TABLE IF NOT EXISTS problems (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    difficulty VARCHAR(50) DEFAULT 'Medium',
    tags JSONB DEFAULT '[]'::jsonb,
    description TEXT NOT NULL,
    input_format TEXT,
    output_format TEXT,
    constraints TEXT,
    sample_testcases JSONB DEFAULT '[]'::jsonb,
    hidden_testcases JSONB NOT NULL DEFAULT '[]'::jsonb,
    -- Problem Setter Pipeline Extensions
    status VARCHAR(50) DEFAULT 'DRAFT',             -- DRAFT | READY_FOR_REVIEW | PUBLISHED
    time_limit_ms INTEGER DEFAULT 2000,             -- per-testcase time limit in ms
    memory_limit_mb INTEGER DEFAULT 256,            -- per-testcase memory limit in MB
    checker_type VARCHAR(50) DEFAULT 'STANDARD',    -- STANDARD | FLOATING_POINT | CUSTOM
    checker_config JSONB DEFAULT '{}'::jsonb,        -- e.g. {"epsilon": 1e-6} or {"code": "...", "language": "cpp"}
    generator_config JSONB DEFAULT '{}'::jsonb,      -- {"code": "...", "language": "cpp"}
    validator_config JSONB DEFAULT '{}'::jsonb,      -- {"code": "...", "language": "cpp"}
    solution_config JSONB DEFAULT '{}'::jsonb,       -- {"code": "...", "language": "cpp"}
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Individual testcase tracking (replaces monolithic hidden_testcases JSONB for new problems)
CREATE TABLE IF NOT EXISTS testcases (
    id SERIAL PRIMARY KEY,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    test_index INTEGER NOT NULL,                     -- ordering within the problem
    source VARCHAR(50) NOT NULL DEFAULT 'manual',    -- 'manual' | 'generated'
    generator_args TEXT,                              -- arguments used if generated
    input TEXT NOT NULL,
    expected_output TEXT,                             -- NULL until reference solution runs
    is_sample BOOLEAN DEFAULT FALSE,                 -- sample testcase shown to participants
    validation_status VARCHAR(50) DEFAULT 'pending', -- 'pending' | 'valid' | 'invalid'
    validation_message TEXT,                          -- diagnostic from validator
    generation_status VARCHAR(50) DEFAULT 'ready',   -- 'ready' | 'generating' | 'failed'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(problem_id, test_index)
);
CREATE INDEX IF NOT EXISTS idx_testcases_problem_id ON testcases(problem_id);
CREATE INDEX IF NOT EXISTS idx_problems_status ON problems(status);

CREATE TABLE IF NOT EXISTS contests (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'ICPC', 'IOI', 'CUSTOM'
    description TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    duration_minutes INTEGER NOT NULL,
    registration_open_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) DEFAULT 'CREATED', -- 'CREATED', 'UPCOMING', 'RUNNING', 'ENDED'
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS contest_problems (
    contest_id INTEGER REFERENCES contests(id) ON DELETE CASCADE,
    problem_id INTEGER REFERENCES problems(id) ON DELETE CASCADE,
    order_index INTEGER NOT NULL,
    PRIMARY KEY (contest_id, problem_id)
);

CREATE TABLE IF NOT EXISTS problem_sets (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS problem_set_mappings (
    set_id INTEGER REFERENCES problem_sets(id) ON DELETE CASCADE,
    problem_id INTEGER REFERENCES problems(id) ON DELETE CASCADE,
    PRIMARY KEY (set_id, problem_id)
);

CREATE TABLE IF NOT EXISTS submissions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    problem_id INTEGER REFERENCES problems(id) ON DELETE CASCADE,
    contest_id INTEGER REFERENCES contests(id) ON DELETE SET NULL, -- Nullable if practice submission
    code TEXT NOT NULL,
    language VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'PENDING',
    execution_time_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_problem_status (
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    problem_id INTEGER REFERENCES problems(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,
    PRIMARY KEY (user_id, problem_id)
);

CREATE TABLE IF NOT EXISTS contest_registrations (
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    contest_id INTEGER REFERENCES contests(id) ON DELETE CASCADE,
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) DEFAULT 'ACTIVE', -- 'ACTIVE', 'LOCKED'
    PRIMARY KEY (user_id, contest_id)
);

CREATE TABLE IF NOT EXISTS contest_violations (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    contest_id INTEGER REFERENCES contests(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL, -- 'fullscreen_exit', 'tab_switched', 'blur'
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS blogs (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    category TEXT NOT NULL,
    author_id INTEGER REFERENCES users(id),
    content TEXT NOT NULL,
    summary TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
