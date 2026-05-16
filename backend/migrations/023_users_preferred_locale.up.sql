-- Migration 023: FR-BB61 — Add preferred_locale column to users
ALTER TABLE users ADD COLUMN preferred_locale TEXT;
