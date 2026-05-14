-- Migration: 003_roles_departments.up.sql
-- Stub tables for roles and departments required as foreign-key targets by the users table.
-- Full implementations are delivered in FR-BB16 (RBAC) and FR-BB17 (Department Management).

CREATE TABLE roles (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO roles (name) VALUES
    ('super_admin'),
    ('hr_admin'),
    ('department_admin'),
    ('employee')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE departments (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
