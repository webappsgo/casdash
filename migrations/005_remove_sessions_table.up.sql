-- Migration 005: Remove sessions table - not in SPEC
-- The SPEC uses session_token and session_expires columns in users table

DROP TABLE IF EXISTS sessions;
DROP INDEX IF EXISTS idx_sessions_token;
