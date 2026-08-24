-- Migration: Problem Setter Pipeline
-- Run this on existing databases that already have the problems table.
-- All operations are idempotent (IF NOT EXISTS / DO NOTHING patterns).

-- 1. Extend problems table with pipeline columns
ALTER TABLE problems ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'DRAFT';
ALTER TABLE problems ADD COLUMN IF NOT EXISTS time_limit_ms INTEGER DEFAULT 2000;
ALTER TABLE problems ADD COLUMN IF NOT EXISTS memory_limit_mb INTEGER DEFAULT 256;
ALTER TABLE problems ADD COLUMN IF NOT EXISTS checker_type VARCHAR(50) DEFAULT 'STANDARD';
ALTER TABLE problems ADD COLUMN IF NOT EXISTS checker_config JSONB DEFAULT '{}'::jsonb;
ALTER TABLE problems ADD COLUMN IF NOT EXISTS generator_config JSONB DEFAULT '{}'::jsonb;
ALTER TABLE problems ADD COLUMN IF NOT EXISTS validator_config JSONB DEFAULT '{}'::jsonb;
ALTER TABLE problems ADD COLUMN IF NOT EXISTS solution_config JSONB DEFAULT '{}'::jsonb;
ALTER TABLE problems ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Set existing problems to PUBLISHED so they remain visible
UPDATE problems SET status = 'PUBLISHED' WHERE status IS NULL OR status = 'DRAFT';

-- Set default for hidden_testcases on problems that might have NULL
ALTER TABLE problems ALTER COLUMN hidden_testcases SET DEFAULT '[]'::jsonb;

-- 2. Create testcases table
CREATE TABLE IF NOT EXISTS testcases (
    id SERIAL PRIMARY KEY,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    test_index INTEGER NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'manual',
    generator_args TEXT,
    input TEXT NOT NULL,
    expected_output TEXT,
    is_sample BOOLEAN DEFAULT FALSE,
    validation_status VARCHAR(50) DEFAULT 'pending',
    validation_message TEXT,
    generation_status VARCHAR(50) DEFAULT 'ready',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(problem_id, test_index)
);

-- 3. Create indexes
CREATE INDEX IF NOT EXISTS idx_testcases_problem_id ON testcases(problem_id);
CREATE INDEX IF NOT EXISTS idx_problems_status ON problems(status);
