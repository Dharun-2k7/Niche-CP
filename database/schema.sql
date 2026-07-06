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
    permissions JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
    hidden_testcases JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS contests (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'ICPC', 'IOI', 'CUSTOM'
    start_time TIMESTAMPTZ NOT NULL,
    duration_minutes INTEGER NOT NULL,
    registration_open_time TIMESTAMPTZ NOT NULL,
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
    PRIMARY KEY (user_id, contest_id)
);
